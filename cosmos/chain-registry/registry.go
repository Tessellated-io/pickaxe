package registry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	retry "github.com/avast/retry-go/v4"
)

type RegistryClient struct {
	attempts retry.Option
	delay    retry.Option
}

func NewRegistryClient() *RegistryClient {
	return &RegistryClient{
		attempts: retry.Attempts(5),
		delay:    retry.Delay(1 * time.Second),
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
