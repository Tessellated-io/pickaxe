package tx

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"sync"

	chainregistry "github.com/tessellated-io/pickaxe/cosmos/chain-registry"
	"github.com/tessellated-io/pickaxe/log"

	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// GasManager interprets tx results and associated outcomes.
type GasManager interface {
	// Get a suggested gas price for the chainName
	GetGasPrice(ctx context.Context, chainName string) (float64, error)

	// Given a broadcast result for a chainName , update gas prices.
	ManageBroadcastResult(ctx context.Context, chainName string, broadcastResult *txtypes.BroadcastTxResponse, gasWanted uint64) error

	// Update gas prices on the given chainName  with whether or not a transaction confirmed in a given time period.
	ManageInclusionResult(ctx context.Context, chainName string, confirmed bool) error

	SetChainRegistryClient(newClient chainregistry.ChainRegistryClient)
}

// defaultGasManager implements a naive gas management scheme.
type defaultGasManager struct {
	priceIncrement float64

	consecutiveSuccesses map[string]int
	lock                 *sync.Mutex

	chainRegistryClient chainregistry.ChainRegistryClient
	gasPriceProvider    GasPriceProvider
	logger              *log.Logger
}

var _ GasManager = (*defaultGasManager)(nil)

func NewDefaultGasManager(priceIncrement float64, gasPriceProvider GasPriceProvider, logger *log.Logger, chainRegistryClient chainregistry.ChainRegistryClient) (GasManager, error) {
	gasLogger := logger.ApplyPrefix(" ⛽️")

	gasManager := &defaultGasManager{
		priceIncrement: priceIncrement,

		consecutiveSuccesses: make(map[string]int),
		lock:                 &sync.Mutex{},

		gasPriceProvider:    gasPriceProvider,
		chainRegistryClient: chainRegistryClient,
		logger:              gasLogger,
	}

	return gasManager, nil
}

func (gm *defaultGasManager) SetChainRegistryClient(newClient chainregistry.ChainRegistryClient) {
	gm.chainRegistryClient = newClient
}

func (gm *defaultGasManager) GetGasPrice(ctx context.Context, chainName string) (float64, error) {
	// Attempt to get a gas price, and return if successful.
	gasPrice, err := gm.gasPriceProvider.GetGasPrice(chainName)
	if err == nil {
		gm.logger.Debug().Str("chain_id", chainName).Float64("gas_price", gasPrice).Msg("got gas price from cache")

		return gasPrice, nil
	}

	// Otherwise, fetch the chain info, and get a gas price from it.
	chainInfo, err := gm.chainRegistryClient.ChainInfo(ctx, chainName)
	if err != nil {
		return 0, err
	}
	gasPrice, err = chainInfo.MinGasFee()
	if err != nil {
		return 0, err
	}

	// Set it for the next run
	err = gm.gasPriceProvider.SetGasPrice(chainName, gasPrice)
	if err != nil {
		return 0, err
	}

	gm.logger.Info().Str("chain_name", chainName).Float64("gas_price", gasPrice).Msg("fetched gas price from chain registry")
	return gasPrice, nil
}

func (gm *defaultGasManager) ManageBroadcastResult(ctx context.Context, chainName string, broadcastResult *txtypes.BroadcastTxResponse, gasWanted uint64) error {
	// Extract the code and logs from broadcasting
	codespace := broadcastResult.TxResponse.Codespace
	code := broadcastResult.TxResponse.Code
	logs := broadcastResult.TxResponse.RawLog

	// If code is 0 (success) then do nothing
	if code == 0 {
		gm.trackSuccess(ctx, chainName)
		return nil
	}

	// If code is a gas error, we should try to adjust our gas
	if IsGasError(codespace, code) {
		// Get the old gas price
		oldPrice, err := gm.GetGasPrice(ctx, chainName)
		if err != nil {
			return err
		}

		// Try to extract a fee from the error message.
		chainSuggestedFee, err := gm.extractMinGlobalFee(logs)
		if err == nil {
			// Determine the gas price by dividing the fee by the gas units requested
			if gasWanted == 0 {
				return fmt.Errorf("gas wanted cannot be zero")
			}
			newGasPrice := chainSuggestedFee / float64(gasWanted)

			// Set and log
			err = gm.gasPriceProvider.SetGasPrice(chainName, newGasPrice)
			if err != nil {
				return err
			}
			gm.logger.Info().Str("chain_name", chainName).Float64("old_gas_price", oldPrice).Float64("new_gas_price", newGasPrice).Msg("updated gas price due to transaction broadcast failure")
		} else {
			// Otherwise, simply increment the fee
			newPrice := oldPrice + gm.priceIncrement

			// Set and log
			err = gm.gasPriceProvider.SetGasPrice(chainName, newPrice)
			if err != nil {
				return err
			}
			gm.logger.Info().Str("chain_name", chainName).Float64("old_gas_price", oldPrice).Float64("new_gas_price", newPrice).Msg("updated gas price due to transaction broadcast failure")
		}

		gm.trackFailure(chainName)
		return nil
	} else if codespace == "gaia" && code == 4 {

	}

	// Otherwise, we got an unrelated error for broadcasting. Audibly drop it on the floor.
	gm.logger.Info().Str("chain_name", chainName).Str("logs", logs).Uint32("code", code).Msg("transaction failed to broadcast but failure not related to gas")
	return nil
}

// In our naive implementation, simply bump the gas price if we didn't get a confirmation.
func (gm *defaultGasManager) ManageInclusionResult(ctx context.Context, chainName string, confirmed bool) error {
	// Don't process further if it confirmed successfully.
	if confirmed {
		gm.trackSuccess(ctx, chainName)
		return nil
	}
	gm.trackFailure(chainName)

	// Get the old gas price
	oldPrice, err := gm.GetGasPrice(ctx, chainName)
	if err != nil {
		return err
	}

	// Bump price and set
	newPrice := oldPrice + gm.priceIncrement
	err = gm.gasPriceProvider.SetGasPrice(chainName, newPrice)
	if err != nil {
		return err
	}
	gm.logger.Info().Str("chain_name", chainName).Float64("old_gas_price", oldPrice).Float64("new_gas_price", newPrice).Msg("updated gas price due to failure in transaction confirmation")

	return nil
}

// extractMinGlobalFee is basically a kludge for Evmos, and EIP-1559.
func (gm *defaultGasManager) extractMinGlobalFee(errMsg string) (float64, error) {
	// Regular expression to match the desired number
	pattern := `(\d+)\w+\)\. Please increase`
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(errMsg)
	if len(matches) > 1 {
		converted, err := strconv.Atoi(matches[1])
		if err != nil {
			gm.logger.Error().Err(err).Msg("found a matching min global fee error, but failed to atoi it")
			return 0, nil
		}
		return float64(converted), nil

	}
	return 0, fmt.Errorf("unrecognized error format")
}

// Management functions for tracking consecutive successes
func (gm *defaultGasManager) trackSuccess(ctx context.Context, chainName string) {
	gm.lock.Lock()
	defer gm.lock.Unlock()

	// Increment
	oldValue := gm.consecutiveSuccesses[chainName]
	newValue := oldValue + 1

	// Update the value
	gm.consecutiveSuccesses[chainName] = newValue

	// Try to jitter the gas down.
	consecutiveSuccessThreshold := 3
	if newValue >= consecutiveSuccessThreshold {
		// Get the old gas price
		oldPrice, err := gm.GetGasPrice(ctx, chainName)
		if err != nil {
			gm.logger.Error().Err(err).Str("chain_name", chainName).Int("consecutive_successes", newValue).Msg("attempted to decrement gas but failed to fetch old price")
			return
		}

		// Decrement price, bounding for zero
		newPrice := oldPrice - (gm.priceIncrement / 2.0)
		if newPrice < 0 {
			newPrice = 0
		}

		// Set and log
		err = gm.gasPriceProvider.SetGasPrice(chainName, newPrice)
		if err != nil {
			gm.logger.Error().Err(err).Str("chain_name", chainName).Int("consecutive_successes", newValue).Msg("attempted to decrement gas but failed to setnew price")
			return
		}
		gm.logger.Info().Str("chain_name", chainName).Float64("old_gas_price", oldPrice).Int("consecutive_successes", newValue).Float64("new_gas_price", newPrice).Msg("decremented gas price because of consecutive successes")

	}
}

func (gm *defaultGasManager) trackFailure(chainName string) {
	gm.lock.Lock()
	defer gm.lock.Unlock()

	gm.consecutiveSuccesses[chainName] = 0
}

// GasPriceProvider is a simple KV store for gas.
type GasPriceProvider interface {
	GetGasPrice(chainName string) (float64, error)
	SetGasPrice(chainName string, gasPrice float64) error
}

// InMemoryGasPriceProvider stores gas prices in memory.
type InMemoryGasPriceProvider struct {
	prices map[string]float64

	lock *sync.Mutex
}

var _ GasPriceProvider = (*InMemoryGasPriceProvider)(nil)

func NewInMemoryGasPriceProvider() (GasPriceProvider, error) {
	provider := &InMemoryGasPriceProvider{
		prices: make(map[string]float64),

		lock: &sync.Mutex{},
	}
	return provider, nil
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

// Helper function to determine if an error is related to insufficient gas
func IsGasError(codespace string, code uint32) bool {
	if (codespace == "sdk" && code == 13) || (codespace == "gaia" && code == 4) {
		return true
	}
	return false
}
