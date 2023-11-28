package registry

import (
	"context"
	"errors"
	"time"

	retry "github.com/avast/retry-go/v4"
)

// Implements a retryable and returns the last error
type retryableChainRegistryClient struct {
	wrappedClient ChainRegistryClient

	attempts retry.Option
	delay    retry.Option
}

// Ensure that retryableChainRegistryClient implements ChainRegistryClient
var _ ChainRegistryClient = (*retryableChainRegistryClient)(nil)

// NewRetryableChainRegistryClient returns a new retryableChainRegistryClient
func NewRetryableChainRegistryClient(attempts uint, delay time.Duration, chainRegistryClient ChainRegistryClient) (ChainRegistryClient, error) {
	return &retryableChainRegistryClient{
		wrappedClient: chainRegistryClient,

		attempts: retry.Attempts(attempts),
		delay:    retry.Delay(delay),
	}, nil
}

// ChainRegistryClient Interface

func (r *retryableChainRegistryClient) AllChainNames(ctx context.Context) ([]string, error) {
	var result []string
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.AllChainNames(ctx)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return result, err
}

func (r *retryableChainRegistryClient) ChainNameForChainID(ctx context.Context, targetChainID string, refreshCache bool) (string, error) {
	var result string
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.ChainNameForChainID(ctx, targetChainID, refreshCache)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return result, err
}

func (r *retryableChainRegistryClient) ChainInfo(ctx context.Context, chainName string) (*ChainInfo, error) {
	var result *ChainInfo
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.ChainInfo(ctx, chainName)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return result, err
}
