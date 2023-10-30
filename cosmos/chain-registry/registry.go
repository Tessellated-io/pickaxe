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
)

type RegistryClient struct {
	attempts retry.Option
	delay    retry.Option

	// TODO
	// Cache of all chains
	// allChainNames []string
}

func NewRegistryClient() *RegistryClient {
	return &RegistryClient{
		attempts: retry.Attempts(5),
		delay:    retry.Delay(1 * time.Second),

		// Initially empty chain name cache
		// allChainNames: []string{},
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

// TODO: Add retries
func (rc *RegistryClient) ChainNameForChainID(ctx context.Context, chainID string) (string, error) {
	// Fetch and return if in cache
	chainName, err := rc.chainNameForChainID(ctx, chainID, false)
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

// TODO: use cache param
// TODO: enable caching in this method
func (rc *RegistryClient) chainNameForChainID(ctx context.Context, chainID string, refreshCache bool) (string, error) {
	// Get all chain names
	url := "https://cosmos-chain.directory/chains"
	bytes, err := rc.makeRequest(ctx, url)
	if err != nil {
		return "", err
	}

	chainNames, err := parseAllChainsResponse(bytes)
	if err != nil {
		return "", err
	}

	// For each chain name, cache the chain id
	for _, chainName := range chainNames {
		// Fetch the chain ID for the chain
		chainInfo, err := rc.GetChainInfo(ctx, chainName)
		if err != nil {
			return "", err
		}

		if strings.EqualFold(chainInfo.ChainID, chainID) {
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
