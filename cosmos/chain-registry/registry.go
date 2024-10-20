package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/tessellated-io/pickaxe/log"
)

/**
 * A chain registry client.
 *
 * This class is made to be compatible with Planetarium (https://github.com/tessellated-io/planetarium), which is a hosted version
 * of the Chain Registry provided by Tessellated, although it should be compatible with other services by providing custom base urls.
 */

// Default implementation
type chainRegistryClient struct {
	// Cache of all chain names
	chainNames []string

	// Cache of chain names to chain ID
	chainNameToChainID map[string]string

	// Base url of an API service
	chainRegistryBaseUrl     string
	validatorRegistryBaseUrl string

	log *log.Logger
}

// Type assertion
var _ ChainRegistryClient = (*chainRegistryClient)(nil)

// NewRegistryClient makes a new default registry client.
func NewChainRegistryClient(log *log.Logger, chainRegistryBaseUrl, validatorRegistryBaseUrl string) *chainRegistryClient {
	return &chainRegistryClient{
		// Initially empty chain name cache
		chainNames:         []string{},
		chainNameToChainID: make(map[string]string),

		chainRegistryBaseUrl:     chainRegistryBaseUrl,
		validatorRegistryBaseUrl: validatorRegistryBaseUrl,

		log: log,
	}
}

// ChainRegistryClient interface

func (rc *chainRegistryClient) ChainInfo(ctx context.Context, chainName string) (*ChainInfo, error) {
	url := fmt.Sprintf("%s/%s/chain.json", rc.chainRegistryBaseUrl, chainName)

	bytes, err := rc.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	chainInfo, err := parseChainResponse(bytes)
	if err != nil {
		return nil, err
	}

	// Add data to cache
	rc.chainNameToChainID[chainName] = chainInfo.ChainID

	return chainInfo, nil
}

func (rc *chainRegistryClient) AssetList(ctx context.Context, chainName string) (*AssetList, error) {
	url := fmt.Sprintf("%s/%s/assetlist.json", rc.chainRegistryBaseUrl, chainName)

	bytes, err := rc.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	chainInfo, err := parseAssetListResponse(bytes)
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
		rc.log.Debug("reset chain names and chain ids caches per client request")
	}

	// Fetch chain names if they are not cached, or if we requested a refetch from the cache
	chainNames := rc.chainNames
	if len(chainNames) == 0 {
		rc.log.Debug("no index of chain names, reloading from registry")

		var err error
		chainNames, err = rc.AllChainNames(ctx)
		if err != nil {
			return "", err
		}

		rc.log.Debug("loaded chains from the registry", "num_chains", len(chainNames))
		rc.chainNames = chainNames
		rc.log.Debug("updated chain name index in client", "num_chains", len(rc.chainNames))
	}

	// For each chain name, get the chain id from the cache or from fetching
	for chainIdx, chainName := range chainNames {
		logger := rc.log.With("chain_name ", chainName)
		logger.Debug("processing chain", "chain_index", chainIdx)

		chainID, isSet := rc.chainNameToChainID[chainName]
		if !isSet {
			logger.Debug("no chain data found in cache, requesting from registry")

			// Fetch the chain ID for the chain
			// NOTE: No retries because GetChainInfo manages that for us.
			chainInfo, err := rc.ChainInfo(ctx, chainName)
			if err != nil {
				logger.Warn("error fetching chain information during chain id refresh, this chain will not be supported")
			} else {

				chainID = chainInfo.ChainID

				// Set in cache
				rc.chainNameToChainID[chainName] = chainID
			}
		} else {
			logger.Debug("found chain id in cache")
		}

		if strings.EqualFold(targetChainID, chainID) {
			return chainName, nil
		}
	}

	return "", ErrNoChainFoundForChainID
}

func (rc *chainRegistryClient) AllChainNames(ctx context.Context) ([]string, error) {
	// Get all chain names
	url := fmt.Sprintf("%s/all", rc.chainRegistryBaseUrl)
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
	url := fmt.Sprintf("%s/%s/chains.json", rc.validatorRegistryBaseUrl, targetValidator)
	bytes, err := rc.makeRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	response, err := parseValidator(bytes)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// Private helpers

func (rc *chainRegistryClient) makeRequest(ctx context.Context, url string) ([]byte, error) {
	rc.log.Debug("making GET request to url", "url", url)

	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")

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
		rc.log.Debug("received http 200 response from chain registry")

		return data, nil
	} else {
		data, err := io.ReadAll(resp.Body)
		if err == nil {
			rc.log.Debug("received bad response from chain registry", "response", string(data), "status_code", resp.StatusCode)
		}

		return nil, fmt.Errorf("received non-OK HTTP status: %d", resp.StatusCode)
	}
}
