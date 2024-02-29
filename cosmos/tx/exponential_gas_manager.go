package tx

import (
	"fmt"
	"math"
	"sync"

	"github.com/tessellated-io/pickaxe/log"

	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// Gas manager using exponential backoff.
//
// Rough algorithm:
//   - Given some number of consecutive successes, decrement price by a step size.
//     Formula: price_new = price_old - step_size
//   - Given a failure, increase step sizes exponentially.
//     Formula: price_new = price_old - (step_size * (1 + scale_factor)^(consecutive_failures))
//
// / 	 bounded to maxStepSize
type geometricGasManager struct {
	// Parameters
	stepSize    float64
	maxStepSize float64
	scaleFactor float64

	// State
	consecutiveGasPriceSuccesses map[string]int
	consecutiveGasPriceFailures  map[string]int

	consecutiveGasFactorSuccesses map[string]int
	consecutiveGasFactorFailures  map[string]int

	lock *sync.Mutex

	// Core Services
	gasPriceProvider GasPriceProvider
	logger           *log.Logger
}

var _ GasManager = (*geometricGasManager)(nil)

func NewGeometricGasManager(
	stepSize float64,
	maxStepSize float64,
	scaleFactor float64,
	gasPriceProvider GasPriceProvider,
	logger *log.Logger,
) (GasManager, error) {
	if scaleFactor < 0 || scaleFactor >= 1 {
		return nil, fmt.Errorf("invalid scale factor: %f. Must conform to: 0 < scale_factor < 1", scaleFactor)
	}
	gasLogger := logger.ApplyPrefix("⛽️")

	lock := &sync.Mutex{}

	gasManager := &geometricGasManager{
		stepSize:    stepSize,
		maxStepSize: maxStepSize,
		scaleFactor: scaleFactor,

		consecutiveGasPriceSuccesses: make(map[string]int),
		consecutiveGasPriceFailures:  make(map[string]int),

		consecutiveGasFactorSuccesses: make(map[string]int),
		consecutiveGasFactorFailures:  make(map[string]int),
		lock:                          lock,

		logger:           gasLogger,
		gasPriceProvider: gasPriceProvider,
	}

	return gasManager, nil
}

// Initialize a price. If already initialized, this is a no-op.
func (g *geometricGasManager) InitializePrice(chainName string, gasPrice float64) error {
	// Check if the price is initialized and warn if so
	hasPrice, err := g.gasPriceProvider.HasGasPrice(chainName)
	if err != nil {
		return err
	}

	if hasPrice {
		g.logger.Warn().Str("chain_name", chainName).Msg("requested initialization of previously initialized price. this is a no-op.")
		return nil
	}

	return g.gasPriceProvider.SetGasPrice(chainName, gasPrice)
}

// Get a gas price
func (g *geometricGasManager) GetGasPrice(chainName string) (float64, error) {
	// Attempt to get a gas price, and return if successful.
	gasPrice, err := g.gasPriceProvider.GetGasPrice(chainName)
	if err == ErrNoGasPrice {
		g.logger.Warn().Str("chain_name", chainName).Msg("no gas price found for chain, setting gas to be zero")
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	return gasPrice, err
}

func (gm *geometricGasManager) GetGasFactor(chainName string) (float64, error) {
	// Attempt to get a gas factor, and return if successful.
	gasFactor, err := gm.gasPriceProvider.GetGasFactor(chainName)
	if err == ErrNoGasFactor {
		gm.logger.Warn().Str("chain_name", chainName).Msg("no gas factor found for chain, setting gas factor to be 1.2")
		return 1.2, nil
	} else if err != nil {
		return 0.0, err
	}
	return gasFactor, err
}

// Feedback methods

// Provides feedback to the gas manager.
// Call one of these function after you know if the last provided gas price was high enough. Generally this is after either:
// - A `broadcast` RPC call (but you don't necessarily know that it is a gas error)
// - Polling for a transaction after a call and finding it included or not, or a broadcast result you know is a gas error.
// NOTE: You probably don't want to call after both, as that provides duplicate feedback.

func (g *geometricGasManager) ManageFailingBroadcastResult(chainName string, broadcastResult *txtypes.BroadcastTxResponse) error {
	if broadcastResult == nil {
		return fmt.Errorf("received nil broadcast tx result")
	}
	if broadcastResult.TxResponse == nil {
		return fmt.Errorf("received nil tx response in broadcast tx result")
	}

	// Extract the code and logs from broadcast tx response
	codespace := broadcastResult.TxResponse.Codespace
	code := broadcastResult.TxResponse.Code
	logs := broadcastResult.TxResponse.RawLog

	// 1. If code was success, then ditch since this method only manages failures.
	isSuccess, err := IsSuccess(broadcastResult)
	if err != nil {
		return err
	}

	if isSuccess {
		g.logger.Warn().Str("chain_name", chainName).Msg("tx broadcast result was successful, but asked gas manager to track a failure.")
		return nil
	}

	return g.trackFailingCodeAndCodespace(code, codespace, chainName, logs, uint(broadcastResult.TxResponse.GasWanted))
}

func (g *geometricGasManager) ManageIncludedTransactionStatus(chainName string, txStatus *txtypes.GetTxResponse) error {
	// Extract the code and logs from broadcast tx response
	codespace := txStatus.TxResponse.Codespace
	code := txStatus.TxResponse.Code
	logs := txStatus.TxResponse.RawLog

	// 1. If code was success, then ditch since this method only manages failures.
	if IsSuccessTxStatus(txStatus) {
		err := g.trackGasPriceSuccess(chainName)
		if err != nil {
			return err
		}

		err = g.trackGasFactorSuccess(chainName)
		if err != nil {
			return err
		}

		return nil
	}

	// Otherwise, use core tracking logic
	return g.trackFailingCodeAndCodespace(code, codespace, chainName, logs, uint(txStatus.TxResponse.GasWanted))
}

// This only tracks gas price
func (g *geometricGasManager) ManageInclusionFailure(chainName string) error {
	return g.trackGasPriceFailure(chainName)
}

// Helpers - state tracking

func (g *geometricGasManager) trackFailingCodeAndCodespace(code uint32, codespace, chainName, logs string, gasWanted uint) error {
	// 2. If the code was not a gas error, then it is non-deterministic, so do nothing.
	if !IsGasRelatedError(codespace, code) {
		g.logger.Info().Str("chain_name", chainName).Uint32("code", code).Str("logs", logs).Str("codespace", codespace).Msg("broadcast result was unrelated to gas. not adjusting gas prices or gas factor")
		return nil
	}

	// 3. Manage failures do to gas price
	if IsGasPriceError(codespace, code) {
		// 3a. Grab the old price, which is useful for logging.
		oldGasPrice, err := g.GetGasPrice(chainName)
		if err != nil {
			return err
		}

		// 3b. Otherwise, it is a gas error so track a failure (which might auto adjust)
		err = g.trackGasPriceFailure(chainName)
		if err != nil {
			return err
		}

		// 3c. If the network gave us a price, we can just use that one though.
		chainSuggestedFee, err := extractMinGlobalFee(logs)
		if err == nil {
			// Determine the gas price by dividing the fee by the gas units requested
			if gasWanted == 0 {
				return fmt.Errorf("gas wanted cannot be zero")
			}
			newGasPrice := chainSuggestedFee / float64(gasWanted)

			// Set and log
			err = g.gasPriceProvider.SetGasPrice(chainName, newGasPrice)
			if err != nil {
				return err
			}
			g.logger.Info().Str("chain_name", chainName).Float64("new_gas_price", newGasPrice).Float64("old_gas_price", oldGasPrice).Str("logs", logs).Msg("calculated exact price from chain suggestion")
		}
		return nil
	} else if isGasAmountError(codespace, code) {
		return g.trackGasFactorFailure(chainName)
	} else {
		// This should never happen...
		panic(fmt.Errorf("unexpected condition in gas manager adjustments with code %d and codespace %s", code, codespace))
	}
}

func (g *geometricGasManager) trackGasFactorFailure(chainName string) error {
	// Lock for map updates
	g.lock.Lock()
	defer g.lock.Unlock()

	// Accounting
	g.consecutiveGasFactorSuccesses[chainName] = 0

	failures := g.consecutiveGasFactorFailures[chainName] + 1
	g.consecutiveGasFactorFailures[chainName] = failures

	return g.adjustFactor(chainName, 0, failures)
}

func (g *geometricGasManager) trackGasFactorSuccess(chainName string) error {
	// Lock for map updates
	g.lock.Lock()
	defer g.lock.Unlock()

	// Accounting
	g.consecutiveGasFactorFailures[chainName] = 0

	successes := g.consecutiveGasFactorSuccesses[chainName] + 1
	g.consecutiveGasFactorSuccesses[chainName] = successes

	return g.adjustFactor(chainName, successes, 0)
}

func (g *geometricGasManager) trackGasPriceFailure(chainName string) error {
	// Lock for map updates
	g.lock.Lock()
	defer g.lock.Unlock()

	// Accounting
	g.consecutiveGasPriceSuccesses[chainName] = 0

	failures := g.consecutiveGasPriceFailures[chainName] + 1
	g.consecutiveGasPriceFailures[chainName] = failures

	// Adjustments
	return g.adjustPrice(chainName, 0, failures)
}

func (g *geometricGasManager) trackGasPriceSuccess(chainName string) error {
	// Lock for map updates
	g.lock.Lock()
	defer g.lock.Unlock()

	// Accounting
	g.consecutiveGasPriceFailures[chainName] = 0

	successes := g.consecutiveGasPriceSuccesses[chainName] + 1
	g.consecutiveGasPriceSuccesses[chainName] = successes

	g.consecutiveGasPriceFailures[chainName]++

	// Adjustments
	return g.adjustPrice(chainName, successes, 0)
}

// Helpers -  adjustments

// TODO: config params
const (
	baseFactorSuccessThreshold = 10
	factorStepSize             = 0.01
)

var (
	maxFactorThreshold            = 100
	currentFactorSuccessThreshold = baseFactorSuccessThreshold
	isTryingToStepDownFactor      = false
)

// TODO: Theoretically this could just be injected to allow generalization. That feels over-optimizey for now.
func (g *geometricGasManager) adjustFactor(chainName string, successes, failures int) error {
	// Get starting factor
	oldFactor, err := g.GetGasFactor(chainName)
	if err != nil {
		return err
	}

	var newFactor float64

	// See if we were testing a lower gas factor
	if isTryingToStepDownFactor {
		// We're through our stepping.
		isTryingToStepDownFactor = false

		// If we were trying to step down and we failed, increase threshold (bounding) and step back
		if failures > 0 {
			currentFactorSuccessThreshold += baseFactorSuccessThreshold
			if currentFactorSuccessThreshold > maxFactorThreshold {
				currentFactorSuccessThreshold = maxFactorThreshold
			}

			newFactor = oldFactor + factorStepSize
		} else {
			// New gas factor worked. Reset factor to baseline and reset successes to zero
			currentFactorSuccessThreshold = baseFactorSuccessThreshold
			g.consecutiveGasFactorSuccesses[chainName] = 0
			return nil
		}
	} else {
		// Do nothing if we don't have a failure or a consecutive success
		if failures == 0 && successes < currentFactorSuccessThreshold {
			return nil
		}

		if failures > 0 {
			newFactor = oldFactor + factorStepSize
		} else {
			newFactor = oldFactor - factorStepSize
			if newFactor < 0 {
				newFactor = 0
			}

			// Set that we're trying to step down.
			isTryingToStepDownFactor = true
		}
	}

	err = g.gasPriceProvider.SetGasFactor(chainName, newFactor)
	if err != nil {
		return err
	}

	g.logger.Info().Str("chain_name", chainName).Float64("old_gas_factor", oldFactor).Int("consecutive_successes", successes).Int("consecutive_failures", failures).Float64("new_gas_factor", newFactor).Msg("adjusted gas factor in response to feedback")
	return nil
}

// TODO: Theoretically this could just be injected to allow generalization. That feels over-optimizey for now.
func (g *geometricGasManager) adjustPrice(chainName string, successes, failures int) error {
	// Do nothing if we don't have a failure or a consecutive success
	successThreshold := 5
	if failures == 0 && successes < 5 {
		return nil
	}

	// Get starting price
	oldPrice, err := g.GetGasPrice(chainName)
	if err != nil {
		return err
	}

	var newPrice float64
	if failures > 0 {
		// Failures increasing. Scale price up according to consecutive failures
		scale := math.Pow((1 + g.scaleFactor), float64(failures))
		scaledStepSize := g.stepSize * scale

		if scaledStepSize > g.maxStepSize {
			g.logger.Warn().Float64("desired_step_size", scaledStepSize).Float64("max_step_size", g.maxStepSize).Msg("bounding step size")
			scaledStepSize = g.maxStepSize
		}

		newPrice = oldPrice + scaledStepSize
	} else {
		// // Successes increasing, test a decrease of one step
		// newPrice = oldPrice - g.stepSize
		// if newPrice < 0 {
		// 	newPrice = 0
		// }

		// Failures increasing. Scale price up according to consecutive failures
		scale := math.Pow((1 + g.scaleFactor), float64(successes-successThreshold))
		scaledStepSize := g.stepSize * scale

		if scaledStepSize > g.maxStepSize {
			g.logger.Warn().Float64("desired_step_size", scaledStepSize).Float64("max_step_size", g.maxStepSize).Msg("bounding step size")
			scaledStepSize = g.maxStepSize
		}

		newPrice = oldPrice - scaledStepSize
		if newPrice < 0 {
			newPrice = 0
		}
	}

	err = g.gasPriceProvider.SetGasPrice(chainName, newPrice)
	if err != nil {
		return err
	}

	g.logger.Info().Str("chain_name", chainName).Float64("old_gas_price", oldPrice).Int("consecutive_successes", successes).Int("consecutive_failures", failures).Float64("new_gas_price", newPrice).Msg("adjusted gas price in response to feedback")
	return nil
}
