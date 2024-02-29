package tx

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"sync"

	"github.com/tessellated-io/pickaxe/log"

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

	getGasData() (*GasData, error)
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

func (gp *InMemoryGasPriceProvider) getGasData() (*GasData, error) {
	gp.lock.Lock()
	defer gp.lock.Unlock()

	gasPrices := make(map[string]float64)
	for chainName, price := range gp.prices {
		gasPrices[chainName] = price
	}

	gasFactors := make(map[string]float64)
	for chainName, factor := range gp.factors {
		gasFactors[chainName] = factor
	}

	gasData := &GasData{
		GasPrices:  gasPrices,
		GasFactors: gasFactors,
	}

	return gasData, nil
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

// FileGasPriceProvider writes gas prices to a file by internally wrapping calls to an InMemoryGasPriceProvider.
type FileGasPriceProvider struct {
	wrapped GasPriceProvider

	logger      *log.Logger
	gasDataFile string

	lock *sync.Mutex
}

// gasDataFile is the file name inside the data directory
const gasDataFile = "gas_prices.json"

// Assert all FileGasPriceProvider are GasPriceProviders
var _ GasPriceProvider = (*FileGasPriceProvider)(nil)

// Data format for gas file.
type GasData struct {
	GasFactors map[string]float64 `json:"gas_factors"`
	GasPrices  map[string]float64 `json:"gas_prices"`
}

// Create a new FileGasProvider which will wrap an in-memory gas price provider
func NewFileGasPriceProvider(logger *log.Logger, dataDirectory string) (GasPriceProvider, error) {
	// Wrap an in memory provider, so that the logic is reused
	// TODO: InMemoryGasPriceProvider should probably be renamed to BaseGasPriceProvider
	wrapped, err := NewInMemoryGasPriceProvider()
	if err != nil {
		return nil, err
	}

	// Create a provider
	gasDataFile := fmt.Sprintf("%s/%s", dataDirectory, gasDataFile)
	provider := &FileGasPriceProvider{
		wrapped:     wrapped,
		logger:      logger,
		gasDataFile: gasDataFile,
		lock:        &sync.Mutex{},
	}

	// Initialize the wrapped provider.
	err = provider.initialize()
	if err != nil {
		return nil, err
	}

	return provider, nil
}

func (p *FileGasPriceProvider) HasGasPrice(chainName string) (bool, error) {
	return p.wrapped.HasGasPrice(chainName)
}

func (p *FileGasPriceProvider) GetGasPrice(chainName string) (float64, error) {
	return p.wrapped.GetGasPrice(chainName)
}

func (p *FileGasPriceProvider) SetGasPrice(chainName string, gasPrice float64) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	err := p.wrapped.SetGasPrice(chainName, gasPrice)
	if err != nil {
		return err
	}

	return p.writeToFile()
}

func (p *FileGasPriceProvider) HasGasFactor(chainName string) (bool, error) {
	return p.wrapped.HasGasFactor(chainName)
}

func (p *FileGasPriceProvider) GetGasFactor(chainName string) (float64, error) {
	return p.wrapped.GetGasFactor(chainName)
}

func (p *FileGasPriceProvider) SetGasFactor(chainName string, gasFactor float64) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	err := p.wrapped.SetGasFactor(chainName, gasFactor)
	if err != nil {
		return err
	}

	return p.writeToFile()
}

func (p *FileGasPriceProvider) getGasData() (*GasData, error) {
	return p.wrapped.getGasData()
}

func (p *FileGasPriceProvider) writeToFile() error {
	gasData, err := p.getGasData()
	if err != nil {
		return err
	}

	jsonBytes, err := json.MarshalIndent(gasData, "", "    ")
	if err != nil {
		return err
	}

	err = os.WriteFile(p.gasDataFile, jsonBytes, 0o600)
	if err != nil {
		return err
	}

	p.logger.Info().Str("file", p.gasDataFile).Msg("💾 saved gas prices to disk")
	return nil
}

// Initialize the wrapped provider with data from a file.
func (p *FileGasPriceProvider) initialize() error {
	p.logger.Info().Str("file", p.gasDataFile).Msg("💾 initializing gas prices from disk")
	gasData, err := p.loadData()
	if err != nil {
		return err
	}

	for chainName, gasFactor := range gasData.GasFactors {
		err := p.wrapped.SetGasFactor(chainName, gasFactor)
		if err != nil {
			return err
		}
		p.logger.Info().Str("chain_name", chainName).Float64("gas_factor", gasFactor).Msg("💾 initialized gas factor")
	}

	for chainName, gasPrice := range gasData.GasPrices {
		err := p.wrapped.SetGasPrice(chainName, gasPrice)
		if err != nil {
			return err
		}
		p.logger.Info().Str("chain_name", chainName).Float64("gas_price", gasPrice).Msg("💾 initialized gas price")
	}

	p.logger.Info().Str("file", p.gasDataFile).Msg("gas price state initialization complete")
	return nil
}

// Load data from the file
func (p *FileGasPriceProvider) loadData() (*GasData, error) {
	_, err := os.Stat(p.gasDataFile)
	if os.IsNotExist(err) {
		p.logger.Info().Str("file", p.gasDataFile).Msg("💾 no gas price cache found on disk. will not initialize.")
		return &GasData{}, nil
	}

	file, err := os.Open(p.gasDataFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gasData := &GasData{}
	if err := json.NewDecoder(file).Decode(gasData); err != nil {
		return nil, err
	}

	return gasData, nil
}
