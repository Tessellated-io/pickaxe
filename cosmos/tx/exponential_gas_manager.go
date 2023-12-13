package tx

import (
	"fmt"
	"math"
	"sync"

	"github.com/tessellated-io/pickaxe/log"
)

// Gas manager using exponential backoff.
//
// Rough algorithm:
//   - Given some number of consecutive successes, decrement price by a step size.
//     Formula: price_new = price_old - step_size
//   - Given a failure, increase step sizes exponentially.
//     Formula: price_new = price_old - (step_size * (1 + scale_factor)^(consecutive_failures))
type geometricGasManager struct {
	// Parameters
	stepSize    float64
	scaleFactor float64

	// State
	consecutiveSuccesses map[string]int
	consecutiveFailures  map[string]int
	lock                 *sync.Mutex

	// Core Services
	gasPriceProvider GasPriceProvider
	logger           *log.Logger
}

// TODO
// var _ GasManager = (*geometricGasManager)(nil)

// TODO: return gas manager
func NewGeometricGasManager(
	stepSize float64,
	scaleFactor float64,
	gasPriceProvider GasPriceProvider,
	logger *log.Logger,
) (*geometricGasManager, error) {
	if scaleFactor < 0 || scaleFactor >= 1 {
		return nil, fmt.Errorf("invalid scale factor: %f. Must conform to: 0 < scale_factor < 1", scaleFactor)
	}
	gasLogger := logger.ApplyPrefix("⛽️ ")

	lock := &sync.Mutex{}

	gasManager := &geometricGasManager{
		stepSize:    stepSize,
		scaleFactor: scaleFactor,

		consecutiveSuccesses: make(map[string]int),
		consecutiveFailures:  make(map[string]int),
		lock:                 lock,

		logger:           gasLogger,
		gasPriceProvider: gasPriceProvider,
	}

	return gasManager, nil
}

// Initialize a price. If already initialized, this is a no-op.
func (g *geometricGasManager) InitializePrice(chainID string, gasPrice float64) error {
	// Check if the price is initialized and warn if so
	hasPrice, err := g.gasPriceProvider.HasGasPrice(chainID)
	if err != nil {
		return err
	}

	if hasPrice {
		g.logger.Warn().Str("chain_id", chainID).Msg("requested initialization of previously initialized price. this is a no-op.")
		return nil
	}

	return g.gasPriceProvider.SetGasPrice(chainID, gasPrice)
}

// Get a gas price
func (g *geometricGasManager) GetGasPrice(chainID string) (float64, error) {
	// Attempt to get a gas price, and return if successful.
	gasPrice, err := g.gasPriceProvider.GetGasPrice(chainID)
	if err == ErrNoGasPrice {
		g.logger.Warn().Str("chain_id", chainID).Msg("no price found for chain, setting gas to be zero")
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	return gasPrice, err
}

// Feedback methods

// Provides feedback to the gas manager.
// Call one of these function after you know if the last provided gas price was high enough. Generally this is after either:
// - A `broadcast` RPC call (but you don't necessarily know that it is a gas error)
// - Polling for a transaction after a call and finding it included or not, or a broadcast result you know is a gas error.
// NOTE: You probably don't want to call after both, as that provides duplicate feedback.


func (g *geometricGasManager) ManageBroadcastResult(ctx context.Context, chainID string, broadcastResult *txtypes.BroadcastTxResponse) error {
	if broadcastResult == nil {
		return fmt.Errorf("received nil broadcast tx result")
	}

	if IsGasError()
}

func (g *geometricGasManager) ManageInclusionResult(ctx context.Context, chainID string, confirmed bool) error {}
}

// Helpers - state tracking

func (g *geometricGasManager) trackFailure(chainID string) error {
	// Lock for map updates
	g.lock.Lock()
	defer g.lock.Unlock()

	// Accounting
	g.consecutiveSuccesses[chainID] = 0
	g.consecutiveFailures[chainID]++

	// Adjustments
	return g.adjustPrice(chainID)
}

func (g *geometricGasManager) trackSuccess(chainID string) error {
	// Lock for map updates
	g.lock.Lock()
	defer g.lock.Unlock()

	// Accounting
	g.consecutiveSuccesses[chainID] = 0
	g.consecutiveFailures[chainID]++

	// Adjustments
	return g.adjustPrice(chainID)
}

// Helpers - price adjustment

// TODO: Theoretically this could just be injected to allow generalization. That feels over-optimizey for now.
func (g *geometricGasManager) adjustPrice(chainID string) error {
	successes := g.consecutiveSuccesses[chainID]
	failures := g.consecutiveFailures[chainID]

	// Do nothing if we don't have a failure or a consecutive success
	if failures == 0 && successes < 5 {
		return nil
	}

	// Get starting price
	oldPrice, err := g.gasPriceProvider.GetGasPrice(chainID)
	if err != nil {
		return err
	}

	newPrice := oldPrice
	if failures > 0 {
		// Failures increasing. Scale price up according to consecutive failures
		scaledStepSize := math.Pow((1 + g.scaleFactor), float64(failures))
		newPrice = oldPrice + scaledStepSize
	} else {
		// Successes increasing, test a decrease of one step
		newPrice = oldPrice - g.stepSize
	}

	err = g.gasPriceProvider.SetGasPrice(chainID, newPrice)
	if err != nil {
		return err
	}

	g.logger.Info().Str("chain_name", chainID).Float64("old_gas_price", oldPrice).Int("consecutive_successes", successes).Int("consecutive_failures", failures).Float64("new_gas_price", newPrice).Msg("adjusted gas price in response to feedback")
	return nil
}
