package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/tessellated-io/pickaxe/log"
)

// Default implementation
type chainRegistryClient struct {
	// Cache of all chain names
	chainNames []string

	// Cache of chain names to chain ID
	chainNameToChainID map[string]string

	log *log.Logger
}

// Type assertion
var _ ChainRegistryClient = (*chainRegistryClient)(nil)

// NewRegistryClient makes a new default registry client.
func NewChainRegistryClient(log *log.Logger) *chainRegistryClient {
	return &chainRegistryClient{
		// Initially empty chain name cache
		chainNames:         []string{},
		chainNameToChainID: make(map[string]string),

		log: log,
	}
}

// ChainRegistryClient interface

func (rc *chainRegistryClient) GetChainInfo(ctx context.Context, chainName string) (*ChainInfo, error) {
	url := fmt.Sprintf("https://proxy.atomscan.com/directory/%s/chain.json", chainName)
	bytes, err := rc.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	chainInfo, err := parseChainResponse(bytes)
	if err != nil {
		return nil, err
	}
	return chainInfo, nil
}

func (rc *chainRegistryClient) ChainNameForChainID(ctx context.Context, targetChainID string, refreshCache bool) (string, error) {
	// If refresh cache is requested, clear the local values
	if refreshCache {
		rc.chainNames = []string{}
		rc.chainNameToChainID = make(map[string]string)
		rc.log.Debug().Msg("reset chain names and chain ids caches")
	}

	// Fetch chain names if they are not cached, or if we requested a refetch from the cache
	chainNames := rc.chainNames
	if len(chainNames) == 0 {
		rc.log.Debug().Msg("no cached name found, reloading from registry")

		var err error
		chainNames, err = rc.AllChainNames(ctx)
		if err != nil {
			return "", err
		}

		rc.log.Debug().Int("num_chains", len(chainNames)).Msg("loaded chains from the registry")
		rc.chainNames = chainNames
	}

	// For each chain name, get the chain id from the cache or from fetching
	for _, chainName := range chainNames {
		rc.log.Debug().Str("chain_name", chainName).Msg("processing chain")

		chainID, isSet := rc.chainNameToChainID[chainName]
		if !isSet {
			// Fetch the chain ID for the chain
			// NOTE: No retries because GetChainInfo manages that for us.
			chainInfo, err := rc.GetChainInfo(ctx, chainName)
			if err != nil {
				return "", err
			}

			chainID = chainInfo.ChainID
		}

		if strings.EqualFold(chainName, chainID) {
			return chainName, nil
		}
	}

	return "", ErrNoChainFoundForChainID
}

func (rc *chainRegistryClient) AllChainNames(ctx context.Context) ([]string, error) {
	// Get all chain names
	url := "https://cosmos-chain.directory/chains"
	bytes, err := rc.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	chainNames, err := parseAllChainsResponse(bytes)
	if err != nil {
		return nil, err
	}

	return chainNames, nil
}

func (rc *chainRegistryClient) Validator(ctx context.Context, targetValidator string) (*Validator, error) {
	validators, err := rc.Validators(ctx)
	if err != nil {
		return nil, err
	}

	validator, err := rc.extractValidator(targetValidator, validators)
	if err != nil {
		return nil, err
	}
	return validator, nil
}

func (rc *chainRegistryClient) Validators(ctx context.Context) ([]Validator, error) {
	bytes, err := rc.makeRequest(ctx, "https://validators.cosmos.directory/")
	if err != nil {
		return nil, err
	}

	response, err := parseValidatorRegistryResponse(bytes)
	if err != nil {
		return nil, err
	}
	return response.Validators, nil
}

// Private helpers

func (rc *chainRegistryClient) makeRequest(ctx context.Context, url string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return data, nil
	} else {
		return nil, fmt.Errorf("received non-OK HTTP status: %d", resp.StatusCode)
	}
}

func (rc *chainRegistryClient) extractValidator(targetValidator string, validators []Validator) (*Validator, error) {
	for _, validator := range validators {
		if strings.EqualFold(targetValidator, validator.Name) {
			return &validator, nil
		}
	}
	return nil, fmt.Errorf("unable to find a validator with name \"%s\"", targetValidator)
}
