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
