package rpc

import (
	"context"
	"errors"
	"time"

	retry "github.com/avast/retry-go/v4"
	"github.com/tessellated-io/pickaxe/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Implements retryable rpcs and returns the last error
type retryableRpcClient struct {
	wrappedClient RpcClient

	attempts retry.Option
	delay    retry.Option

	logger *log.Logger
}

// Ensure that retryableRpcClient implements RpcClient
var _ RpcClient = (*retryableRpcClient)(nil)

// NewRetryableRPCClient returns a new retryableRpcClient
func NewRetryableRpcClient(attempts uint, delay time.Duration, rpcClient RpcClient, logger *log.Logger) (RpcClient, error) {
	return &retryableRpcClient{
		wrappedClient: rpcClient,

		attempts: retry.Attempts(attempts),
		delay:    retry.Delay(delay),

		logger: logger,
	}, nil
}

// RpcClient Interface

func (r *retryableRpcClient) Broadcast(ctx context.Context, txBytes []byte) (*txtypes.BroadcastTxResponse, error) {
	var result *txtypes.BroadcastTxResponse
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.Broadcast(ctx, txBytes)
		if err != nil {
			r.logger.Error().Err(err).Str("method", "broadcast").Msg("failed call in rpc client, will retry")
		}

		return err
	}, r.delay, r.attempts, retry.Context(ctx))

	if err != nil {
		// If err is an error from a context, unwrapping will write out nil
		unwrappedErr := errors.Unwrap(err)
		if unwrappedErr != nil {
			return nil, unwrappedErr
		} else {
			return nil, err
		}
	}

	return result, nil
}

func (r *retryableRpcClient) GetTxStatus(ctx context.Context, txHash string) (*txtypes.GetTxResponse, error) {
	var result *txtypes.GetTxResponse
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.GetTxStatus(ctx, txHash)
		if err != nil {
			r.logger.Error().Err(err).Str("method", "tx_status").Msg("failed call in rpc client, will retry")
		}

		return err
	}, r.delay, r.attempts, retry.Context(ctx))

	if err != nil {
		// If err is an error from a context, unwrapping will write out nil
		unwrappedErr := errors.Unwrap(err)
		if unwrappedErr != nil {
			return nil, unwrappedErr
		} else {
			return nil, err
		}
	}

	return result, nil
}

func (r *retryableRpcClient) Simulate(ctx context.Context, txBytes []byte) (*txtypes.SimulateResponse, error) {
	var result *txtypes.SimulateResponse
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.Simulate(ctx, txBytes)
		if err != nil {
			r.logger.Error().Err(err).Str("method", "simulate").Msg("failed call in rpc client, will retry")
		}
		return err
	}, r.delay, r.attempts, retry.Context(ctx))

	if err != nil {
		// If err is an error from a context, unwrapping will write out nil
		unwrappedErr := errors.Unwrap(err)
		if unwrappedErr != nil {
			return nil, unwrappedErr
		} else {
			return nil, err
		}
	}

	return result, nil
}

func (r *retryableRpcClient) Account(ctx context.Context, address string) (authtypes.AccountI, error) {
	var result authtypes.AccountI
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.Account(ctx, address)

		if err != nil {
			r.logger.Error().Err(err).Str("method", "account").Msg("failed call in rpc client, will retry")
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

func (r *retryableRpcClient) GetBalance(ctx context.Context, address, denom string) (*sdk.Coin, error) {
	var result *sdk.Coin
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.GetBalance(ctx, address, denom)
		if err != nil {
			r.logger.Error().Err(err).Str("method", "balance").Msg("failed call in rpc client, will retry")
		}
		return err
	}, r.delay, r.attempts, retry.Context(ctx))

	if err != nil {
		// If err is an error from a context, unwrapping will write out nil
		unwrappedErr := errors.Unwrap(err)
		if unwrappedErr != nil {
			return nil, unwrappedErr
		} else {
			return nil, err
		}
	}

	return result, nil
}

func (r *retryableRpcClient) GetDenomMetadata(ctx context.Context, denom string) (*banktypes.Metadata, error) {
	var result *banktypes.Metadata
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.GetDenomMetadata(ctx, denom)
		if err != nil {
			r.logger.Error().Err(err).Str("method", "denom_metadata").Msg("failed call in rpc client, will retry")
		}
		return err
	}, r.delay, r.attempts, retry.Context(ctx))

	if err != nil {
		// If err is an error from a context, unwrapping will write out nil
		unwrappedErr := errors.Unwrap(err)
		if unwrappedErr != nil {
			return nil, unwrappedErr
		} else {
			return nil, err
		}
	}

	return result, nil
}

func (r *retryableRpcClient) GetDelegators(ctx context.Context, validatorAddress string) ([]string, error) {
	var result []string
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.GetDelegators(ctx, validatorAddress)
		if err != nil {
			r.logger.Error().Err(err).Str("method", "delegators").Msg("failed call in rpc client, will retry")
		}
		return err
	}, r.delay, r.attempts, retry.Context(ctx))

	if err != nil {
		// If err is an error from a context, unwrapping will write out nil
		unwrappedErr := errors.Unwrap(err)
		if unwrappedErr != nil {
			return nil, unwrappedErr
		} else {
			return nil, err
		}
	}

	return result, nil
}

func (r *retryableRpcClient) GetGrants(ctx context.Context, address string) ([]*authztypes.GrantAuthorization, error) {
	var result []*authztypes.GrantAuthorization
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.GetGrants(ctx, address)
		if err != nil {
			r.logger.Error().Err(err).Str("method", "grants").Msg("failed call in rpc client, will retry")
		}
		return err
	}, r.delay, r.attempts, retry.Context(ctx))

	if err != nil {
		// If err is an error from a context, unwrapping will write out nil
		unwrappedErr := errors.Unwrap(err)
		if unwrappedErr != nil {
			return nil, unwrappedErr
		} else {
			return nil, err
		}
	}

	return result, nil
}

func (r *retryableRpcClient) GetPendingRewards(ctx context.Context, delegator, validator, stakingDenom string) (sdk.Dec, error) {
	var result sdk.Dec
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.GetPendingRewards(ctx, delegator, validator, stakingDenom)
		if err != nil {
			r.logger.Error().Err(err).Str("method", "pending_rewards").Msg("failed call in rpc client, will retry")
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
