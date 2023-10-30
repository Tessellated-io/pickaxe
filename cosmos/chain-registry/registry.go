package registry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	retry "github.com/avast/retry-go/v4"
	"github.com/tessellated-io/pickaxe/log"
)

type RegistryClient struct {
	attempts retry.Option
	delay    retry.Option

	// Cache of all chain names
	chainNames []string

	// Cache of chain names to chain ID
	chainNameToChainID map[string]string

	log *log.Logger
}

func NewRegistryClient(log *log.Logger) *RegistryClient {
	return &RegistryClient{
		attempts: retry.Attempts(5),
		delay:    retry.Delay(1 * time.Second),

		// Initially empty chain name cache
		chainNames:         []string{},
		chainNameToChainID: make(map[string]string),

		log: log,
	}
}

func (rc *RegistryClient) GetChainInfo(ctx context.Context, chainName string) (*ChainInfo, error) {
	var chainInfo *ChainInfo
	var err error

	err = retry.Do(func() error {
		chainInfo, err = rc.getChainInfo(ctx, chainName)
		return err
	}, rc.delay, rc.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return chainInfo, err
}

func (rc *RegistryClient) ChainNameForChainID(ctx context.Context, chainID string) (string, error) {
	var chainName string
	var err error

	// Fetch chain ID with caching enabled
	err = retry.Do(func() error {
		chainName, err = rc.chainNameForChainID(ctx, chainID, false)
		return err
	}, rc.delay, rc.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	// If no error, return the chain ID
	if err == nil {
		return chainName, nil
	}

	// Otherwise, if there was no chain found, try again, breaking the cache.
	if err == ErrNoChainFoundForChainID {
		return rc.chainNameForChainID(ctx, chainID, true)
	} else {
		return "", err
	}
}

func (rc *RegistryClient) chainNameForChainID(ctx context.Context, targetChainID string, refreshCache bool) (string, error) {
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
		chainNames, err = rc.getAllChainNames(ctx)
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

// Internal method without retries
func (rc *RegistryClient) getChainInfo(ctx context.Context, chainName string) (*ChainInfo, error) {
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

// Internal method without retries
func (rc *RegistryClient) getAllChainNames(ctx context.Context) ([]string, error) {
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

func (rc *RegistryClient) makeRequest(ctx context.Context, url string) ([]byte, error) {
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
