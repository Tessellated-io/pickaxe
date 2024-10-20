package registry

import (
	"context"
	"errors"
	"time"

	retry "github.com/avast/retry-go/v4"
	"github.com/tessellated-io/pickaxe/log"
)

// Implements a retryable and returns the last error
type retryableChainRegistryClient struct {
	wrappedClient ChainRegistryClient

	attempts retry.Option
	delay    retry.Option

	logger *log.Logger
}

// Ensure that retryableChainRegistryClient implements ChainRegistryClient
var _ ChainRegistryClient = (*retryableChainRegistryClient)(nil)

// NewRetryableChainRegistryClient returns a new retryableChainRegistryClient
func NewRetryableChainRegistryClient(attempts uint, delay time.Duration, chainRegistryClient ChainRegistryClient, logger *log.Logger) (ChainRegistryClient, error) {
	return &retryableChainRegistryClient{
		wrappedClient: chainRegistryClient,

		attempts: retry.Attempts(attempts),
		delay:    retry.Delay(delay),

		logger: logger,
	}, nil
}

// ChainRegistryClient Interface

func (r *retryableChainRegistryClient) AllChainNames(ctx context.Context) ([]string, error) {
	var result []string
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.AllChainNames(ctx)
		if err != nil {
			r.logger.Error("failed call in registry client, will retry", "error", err.Error(), "method", "all_chain_names")
		}
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		// If err is an error from a context, unwrapping will write out nil
		unwrappedErr := errors.Unwrap(err)
		if unwrappedErr != nil {
			return result, unwrappedErr
		} else {
			return result, err
		}
	}

	return result, nil
}

func (r *retryableChainRegistryClient) ChainNameForChainID(ctx context.Context, targetChainID string, refreshCache bool) (string, error) {
	var result string
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.ChainNameForChainID(ctx, targetChainID, refreshCache)
		if err != nil {
			r.logger.Error("failed call in registry client, will retry", "error", err.Error(), "method", "chain_name_for_id")
		}
		return err
	}, r.delay, r.attempts, retry.Context(ctx))

	if err != nil {
		// If err is an error from a context, unwrapping will write out nil
		unwrappedErr := errors.Unwrap(err)
		if unwrappedErr != nil {
			return result, unwrappedErr
		} else {
			return result, err
		}
	}

	return result, nil
}

func (r *retryableChainRegistryClient) ChainInfo(ctx context.Context, chainName string) (*ChainInfo, error) {
	var result *ChainInfo
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.ChainInfo(ctx, chainName)
		if err != nil {
			r.logger.Error("failed call in registry client, will retry", "error", err.Error(), "method", "chain_info")
		}
		return err
	}, r.delay, r.attempts, retry.Context(ctx))

	if err != nil {
		// If err is an error from a context, unwrapping will write out nil
		unwrappedErr := errors.Unwrap(err)
		if unwrappedErr != nil {
			return result, unwrappedErr
		} else {
			return result, err
		}
	}

	return result, nil
}

func (r *retryableChainRegistryClient) AssetList(ctx context.Context, chainName string) (*AssetList, error) {
	var result *AssetList
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.AssetList(ctx, chainName)
		if err != nil {
			r.logger.Error("failed call in registry client, will retry", "error", err.Error(), "method", "asset_list")
		}
		return err
	}, r.delay, r.attempts, retry.Context(ctx))

	if err != nil {
		// If err is an error from a context, unwrapping will write out nil
		unwrappedErr := errors.Unwrap(err)
		if unwrappedErr != nil {
			return result, unwrappedErr
		} else {
			return result, err
		}
	}

	return result, nil
}

func (r *retryableChainRegistryClient) Validator(ctx context.Context, targetValidator string) (*Validator, error) {
	var result *Validator
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.Validator(ctx, targetValidator)
		if err != nil {
			r.logger.Error("failed call in registry client, will retry", "error", err.Error(), "method", "validator")
		}
		return err
	}, r.delay, r.attempts, retry.Context(ctx))

	if err != nil {
		// If err is an error from a context, unwrapping will write out nil
		unwrappedErr := errors.Unwrap(err)
		if unwrappedErr != nil {
			return result, unwrappedErr
		} else {
			return result, err
		}
	}

	return result, nil
}
