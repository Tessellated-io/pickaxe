package tx

import (
	"fmt"
	"regexp"
	"strconv"
	"sync"

	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// GasManager interprets tx results and associated outcomes.
type GasManager interface {
	InitializePrice(chainName string, gasPrice float64) error

	// Get a suggested gas price for the chainName
	GetGasPrice(chainName string) (float64, error)

	// Get a gas factor for the chain name
	GetGasFactor(chainName string) (float64, error)

	// Given a broadcast result for a chainName, update gas prices. Calling with a successful broadcast result is a no-op. Tally successes
	// using ManageTransactionStatus.
	ManageFailingBroadcastResult(chainName string, broadcastResult *txtypes.BroadcastTxResponse) error

	// Manage a transaction status once it has settled on chain. Statuses could be positive or negative.
	ManageIncludedTransactionStatus(chainName string, txStatus *txtypes.GetTxResponse) error

	// Manage a failure in the case a tx was successfully broadcasted, but never landed on chain and we thus are unable to provide a tx status
	ManageInclusionFailure(chainName string) error
}

// defaultGasManager implements a naive gas management scheme.
// type defaultGasManager struct {
// 	priceIncrement float64

// 	consecutiveSuccesses map[string]int
// 	lock                 *sync.Mutex

// 	gasPriceProvider GasPriceProvider
// 	logger           *log.Logger
// }

// var _ GasManager = (*defaultGasManager)(nil)

// func NewDefaultGasManager(priceIncrement float64, gasPriceProvider GasPriceProvider, logger *log.Logger) (GasManager, error) {
// 	gasLogger := logger.ApplyPrefix(" ⛽️")

// 	gasManager := &defaultGasManager{
// 		priceIncrement: priceIncrement,

// 		consecutiveSuccesses: make(map[string]int),
// 		lock:                 &sync.Mutex{},

// 		gasPriceProvider: gasPriceProvider,
// 		logger:           gasLogger,
// 	}

// 	return gasManager, nil
// }

// func (gm *defaultGasManager) InitializePrice(chainName string, gasPrice float64) error {
// 	// Check if the price is initialized and warn if so
// 	hasPrice, err := gm.gasPriceProvider.HasGasPrice(chainName)
// 	if err != nil {
// 		return err
// 	}

// 	if hasPrice {
// 		gm.logger.Warn().Str("chain_name", chainName).Msg("requested initialization of previously initialized price. this is a no-op.")
// 		return nil
// 	}

// 	return gm.gasPriceProvider.SetGasPrice(chainName, gasPrice)
// }

// // Get a gas price
// func (gm *defaultGasManager) GetGasPrice(chainName string) (float64, error) {
// 	// Attempt to get a gas price, and return if successful.
// 	gasPrice, err := gm.gasPriceProvider.GetGasPrice(chainName)
// 	if err == ErrNoGasPrice {
// 		gm.logger.Warn().Str("chain_name", chainName).Msg("no gas price found for chain, setting gas price to be 0.0")
// 		return 0, nil
// 	} else if err != nil {
// 		return 0, err
// 	}
// 	return gasPrice, err
// }

// func (gm *defaultGasManager) GetGasFactor(chainName string) (float64, error) {
// 	// Attempt to get a gas factor, and return if successful.
// 	gasFactor, err := gm.gasPriceProvider.GetGasFactor(chainName)
// 	if err == ErrNoGasFactor {
// 		gm.logger.Warn().Str("chain_name", chainName).Msg("no gas factor found for chain, setting gas factor to be 1.0")
// 		return 1.0, nil
// 	} else if err != nil {
// 		return 1.0, err
// 	}
// 	return gasFactor, err
// }

// func (gm *defaultGasManager) ManageBroadcastResult(chainName string, broadcastResult *txtypes.BroadcastTxResponse) error {
// 	// Extract the code and logs from broadcasting
// 	codespace := broadcastResult.TxResponse.Codespace
// 	code := broadcastResult.TxResponse.Code
// 	logs := broadcastResult.TxResponse.RawLog

// 	// If code is 0 (success) then do nothing
// 	isSuccess, err := IsSuccess(broadcastResult)
// 	if err != nil {
// 		return err
// 	}

// 	if isSuccess {
// 		gm.trackSuccess(chainName)
// 		return nil
// 	}

// 	// If code is a gas error, we should try to adjust our gas
// 	if IsGasError(codespace, code) {
// 		// Get the old gas price
// 		oldPrice, err := gm.GetGasPrice(chainName)
// 		if err != nil {
// 			return err
// 		}

// 		// Try to extract a fee from the error message.
// 		chainSuggestedFee, err := extractMinGlobalFee(logs)
// 		if err == nil {
// 			// Determine the gas price by dividing the fee by the gas units requested
// 			gasWanted := broadcastResult.TxResponse.GasWanted
// 			if gasWanted == 0 {
// 				return fmt.Errorf("gas wanted cannot be zero")
// 			}
// 			newGasPrice := chainSuggestedFee / float64(gasWanted)

// 			// Set and log
// 			err = gm.gasPriceProvider.SetGasPrice(chainName, newGasPrice)
// 			if err != nil {
// 				return err
// 			}
// 			gm.logger.Info().Str("chain_name", chainName).Float64("old_gas_price", oldPrice).Float64("new_gas_price", newGasPrice).Msg("updated gas price due to transaction broadcast failure")
// 		} else {
// 			// Otherwise, simply increment the fee
// 			newPrice := oldPrice + gm.priceIncrement

// 			// Set and log
// 			err = gm.gasPriceProvider.SetGasPrice(chainName, newPrice)
// 			if err != nil {
// 				return err
// 			}
// 			gm.logger.Info().Str("chain_name", chainName).Float64("old_gas_price", oldPrice).Float64("new_gas_price", newPrice).Msg("updated gas price due to transaction broadcast failure")
// 		}

// 		gm.trackFailure(chainName)
// 		return nil
// 	}

// 	// Otherwise, we got an unrelated error for broadcasting. Audibly drop it on the floor.
// 	gm.logger.Info().Str("chain_name", chainName).Str("logs", logs).Uint32("code", code).Msg("transaction failed to broadcast but failure not related to gas")
// 	return nil
// }

// // In our naive implementation, simply bump the gas price if we didn't get a confirmation.
// func (gm *defaultGasManager) ManageResult(chainName string, successfulGasPrice bool) error {
// 	// Don't process further if it confirmed successfully.
// 	if successfulGasPrice {
// 		gm.trackSuccess(chainName)
// 		return nil
// 	}
// 	gm.trackFailure(chainName)

// 	// Get the old gas price
// 	oldPrice, err := gm.GetGasPrice(chainName)
// 	if err != nil {
// 		return err
// 	}

// 	// Bump price and set
// 	newPrice := oldPrice + gm.priceIncrement
// 	err = gm.gasPriceProvider.SetGasPrice(chainName, newPrice)
// 	if err != nil {
// 		return err
// 	}
// 	gm.logger.Info().Str("chain_name", chainName).Float64("old_gas_price", oldPrice).Float64("new_gas_price", newPrice).Msg("updated gas price due to failure in transaction confirmation")

// 	return nil
// }

// Management functions for tracking consecutive successes
// func (gm *defaultGasManager) trackSuccess(chainName string) {
// 	gm.lock.Lock()
// 	defer gm.lock.Unlock()

// 	// Increment
// 	oldValue := gm.consecutiveSuccesses[chainName]
// 	newValue := oldValue + 1

// 	// Update the value
// 	gm.consecutiveSuccesses[chainName] = newValue

// 	// Try to jitter the gas down.
// 	consecutiveSuccessThreshold := 3
// 	if newValue >= consecutiveSuccessThreshold {
// 		// Get the old gas price
// 		oldPrice, err := gm.GetGasPrice(chainName)
// 		if err != nil {
// 			gm.logger.Error().Err(err).Str("chain_name", chainName).Int("consecutive_successes", newValue).Msg("attempted to decrement gas but failed to fetch old price")
// 			return
// 		}

// 		// Decrement price, bounding for zero
// 		newPrice := oldPrice - (gm.priceIncrement / 2.0)
// 		if newPrice < 0 {
// 			newPrice = 0
// 		}

// 		// Set and log
// 		err = gm.gasPriceProvider.SetGasPrice(chainName, newPrice)
// 		if err != nil {
// 			gm.logger.Error().Err(err).Str("chain_name", chainName).Int("consecutive_successes", newValue).Msg("attempted to decrement gas but failed to setnew price")
// 			return
// 		}
// 		gm.logger.Info().Str("chain_name", chainName).Float64("old_gas_price", oldPrice).Int("consecutive_successes", newValue).Float64("new_gas_price", newPrice).Msg("decremented gas price because of consecutive successes")

// 	}
// }

// func (gm *defaultGasManager) trackFailure(chainName string) {
// 	gm.lock.Lock()
// 	defer gm.lock.Unlock()

// 	gm.consecutiveSuccesses[chainName] = 0
// }

// GasPriceProvider is a simple KV store for gas.
type GasPriceProvider interface {
	HasGasPrice(chainName string) (bool, error)
	GetGasPrice(chainName string) (float64, error)
	SetGasPrice(chainName string, gasPrice float64) error

	HasGasFactor(chainName string) (bool, error)
	GetGasFactor(chainName string) (float64, error)
	SetGasFactor(chainName string, gasFactor float64) error
}

// InMemoryGasPriceProvider stores gas prices in memory.
type InMemoryGasPriceProvider struct {
	prices  map[string]float64
	factors map[string]float64

	lock *sync.Mutex
}

var _ GasPriceProvider = (*InMemoryGasPriceProvider)(nil)

func NewInMemoryGasPriceProvider() (GasPriceProvider, error) {
	provider := &InMemoryGasPriceProvider{
		prices:  make(map[string]float64),
		factors: make(map[string]float64),

		lock: &sync.Mutex{},
	}
	return provider, nil
}

func (gp *InMemoryGasPriceProvider) HasGasPrice(chainName string) (bool, error) {
	gp.lock.Lock()
	defer gp.lock.Unlock()

	_, found := gp.prices[chainName]
	return found, nil
}

func (gp *InMemoryGasPriceProvider) GetGasPrice(chainName string) (float64, error) {
	gp.lock.Lock()
	defer gp.lock.Unlock()

	gasPrice, found := gp.prices[chainName]
	if !found {
		return 0, ErrNoGasPrice
	}

	return gasPrice, nil
}

func (gp *InMemoryGasPriceProvider) SetGasPrice(chainName string, gasPrice float64) error {
	gp.lock.Lock()
	defer gp.lock.Unlock()

	gp.prices[chainName] = gasPrice
	return nil
}

func (gp *InMemoryGasPriceProvider) HasGasFactor(chainName string) (bool, error) {
	gp.lock.Lock()
	defer gp.lock.Unlock()

	_, found := gp.factors[chainName]
	return found, nil
}

func (gp *InMemoryGasPriceProvider) GetGasFactor(chainName string) (float64, error) {
	gp.lock.Lock()
	defer gp.lock.Unlock()

	gasFactor, found := gp.factors[chainName]
	if !found {
		return 0, ErrNoGasFactor
	}

	return gasFactor, nil
}

func (gp *InMemoryGasPriceProvider) SetGasFactor(chainName string, gasFactor float64) error {
	gp.lock.Lock()
	defer gp.lock.Unlock()

	gp.factors[chainName] = gasFactor
	return nil
}

// Helper function to know if an error had to do with gas.
func IsGasRelatedError(codespace string, code uint32) bool {
	return IsGasPriceError(codespace, code) || isGasAmountError(codespace, code)
}

// Helper function to determine if an error is related to too small of a gas price
func IsGasPriceError(codespace string, code uint32) bool {
	return (codespace == "sdk" && code == 13) || (codespace == "gaia" && code == 4)
}

// Helper function to determine if an error is related to to few gas units
func isGasAmountError(codespace string, code uint32) bool {
	return (codespace == "sdk" && code == 11)
}

// extractMinGlobalFee is basically a kludge for Evmos, and EIP-1559.
func extractMinGlobalFee(errMsg string) (float64, error) {
	// Regular expression to match the desired number
	pattern := `(\d+)\w+\)\. Please increase`
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(errMsg)
	if len(matches) > 1 {
		converted, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, err
		}
		return float64(converted), nil

	}
	return 0, fmt.Errorf("unrecognized error format")
}
