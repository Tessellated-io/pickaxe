package rpc

import (
	"context"
	"errors"
	"time"

	retry "github.com/avast/retry-go/v4"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
)

// Implements retryable rpcs and returns the last error
type retryableRpcClient struct {
	wrappedClient RpcClient

	attempts retry.Option
	delay    retry.Option
}

// Ensure that retryableRpcClient implements RpcClient
var _ RpcClient = (*retryableRpcClient)(nil)

// NewRetryableRPCClient returns a new retryableRpcClient
func NewRetryableRpcClient(attempts uint, delay time.Duration, rpcClient RpcClient) (RpcClient, error) {
	return &retryableRpcClient{
		wrappedClient: rpcClient,

		attempts: retry.Attempts(attempts),
		delay:    retry.Delay(delay),
	}, nil
}

// RpcClient Interface

func (r *retryableRpcClient) Broadcast(ctx context.Context, txBytes []byte) (*txtypes.BroadcastTxResponse, error) {
	var result *txtypes.BroadcastTxResponse
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.Broadcast(ctx, txBytes)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return result, err
}

func (r *retryableRpcClient) CheckConfirmed(ctx context.Context, txHash string) error {
	var err error

	err = retry.Do(func() error {
		err = r.wrappedClient.CheckConfirmed(ctx, txHash)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return err
}

func (r *retryableRpcClient) Simulate(ctx context.Context, txBytes []byte) (*txtypes.SimulateResponse, error) {
	var result *txtypes.SimulateResponse
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.Simulate(ctx, txBytes)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return result, err
}

func (r *retryableRpcClient) SimulateTx(ctx context.Context, tx authsigning.Tx, txConfig client.TxConfig, gasFactor float64) (*SimulationResult, error) {
	var result *SimulationResult
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.SimulateTx(ctx, tx, txConfig, gasFactor)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return result, err
}

func (r *retryableRpcClient) GetAccountData(ctx context.Context, address string) (*AccountData, error) {
	var result *AccountData
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.GetAccountData(ctx, address)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return result, err
}

func (r *retryableRpcClient) GetBalance(ctx context.Context, address, denom string) (*sdk.Coin, error) {
	var result *sdk.Coin
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.GetBalance(ctx, address, denom)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return result, err
}

func (r *retryableRpcClient) GetDelegators(ctx context.Context, validatorAddress string) ([]string, error) {
	var result []string
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.GetDelegators(ctx, validatorAddress)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return result, err
}

func (r *retryableRpcClient) GetGrants(ctx context.Context, address string) ([]*authztypes.GrantAuthorization, error) {
	var result []*authztypes.GrantAuthorization
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.GetGrants(ctx, address)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return result, err
}

func (r *retryableRpcClient) GetPendingRewards(ctx context.Context, delegator, validator, stakingDenom string) (sdk.Dec, error) {
	var result sdk.Dec
	var err error

	err = retry.Do(func() error {
		result, err = r.wrappedClient.GetPendingRewards(ctx, delegator, validator, stakingDenom)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return result, err
}
