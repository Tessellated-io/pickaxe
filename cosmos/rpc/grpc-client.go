package rpc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	retry "github.com/avast/retry-go/v4"
	"github.com/tessellated-io/pickaxe/arrays"
	"github.com/tessellated-io/pickaxe/grpc"
	"github.com/tessellated-io/pickaxe/log"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Page size to use
const pageSize = 100

// grpcClient is the private and default implementation for restake.
type grpcClient struct {
	cdc *codec.ProtoCodec

	authClient         authtypes.QueryClient
	authzClient        authztypes.QueryClient
	distributionClient distributiontypes.QueryClient
	stakingClient      stakingtypes.QueryClient
	txClient           txtypes.ServiceClient

	attempts retry.Option
	delay    retry.Option

	log *log.Logger
}

// A struct that came back from an RPC query
type paginatedRpcResponse[dataType any] struct {
	data    []dataType
	nextKey []byte
}

// Ensure that grpcClient implements RpcClient
var _ RpcClient = (*grpcClient)(nil)

// NewRpcClient makes a new RpcClient for the Restake Go App.
func NewGRpcClient(nodeGrpcUri string, cdc *codec.ProtoCodec, log *log.Logger) (RpcClient, error) {
	conn, err := grpc.GetGrpcConnection(nodeGrpcUri)
	if err != nil {
		log.Error().Str("grpc url", nodeGrpcUri).Err(err).Msg("Unable to connect to gRPC")
		return nil, err
	}

	authClient := authtypes.NewQueryClient(conn)
	authzClient := authztypes.NewQueryClient(conn)
	distributionClient := distributiontypes.NewQueryClient(conn)
	stakingClient := stakingtypes.NewQueryClient(conn)
	txClient := txtypes.NewServiceClient(conn)

	return &grpcClient{
		cdc: cdc,

		authClient:         authClient,
		authzClient:        authzClient,
		distributionClient: distributionClient,
		stakingClient:      stakingClient,
		txClient:           txClient,

		attempts: retry.Attempts(5),
		delay:    retry.Delay(1 * time.Second),

		log: log,
	}, nil
}

func (r *grpcClient) GetPendingRewards(ctx context.Context, delegator, validator, stakingDenom string) (sdk.Dec, error) {
	var chainInfo sdk.Dec
	var err error

	err = retry.Do(func() error {
		chainInfo, err = r.getPendingRewards(ctx, delegator, validator, stakingDenom)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return chainInfo, err
}

// private function with retries
func (r *grpcClient) getPendingRewards(ctx context.Context, delegator, validator, stakingDenom string) (sdk.Dec, error) {
	request := &distributiontypes.QueryDelegationTotalRewardsRequest{
		DelegatorAddress: delegator,
	}

	response, err := r.distributionClient.DelegationTotalRewards(ctx, request)
	if err != nil {
		return sdk.NewDec(0), err
	}

	for _, reward := range response.Rewards {
		if strings.EqualFold(validator, reward.ValidatorAddress) {
			for _, coin := range reward.Reward {
				if strings.EqualFold(coin.Denom, stakingDenom) {
					return coin.Amount, nil
				}
			}
		}
	}

	r.log.Debug().Str("delegator", delegator).Str("validator", validator).Msg("unable to find any rewards attributable to validator")
	return sdk.NewDec(0), nil
}

// Broadcast may or may not be populated in the response.
func (r *grpcClient) Broadcast(
	ctx context.Context,
	txBytes []byte,
) (*txtypes.BroadcastTxResponse, error) {
	// Form a query
	query := &txtypes.BroadcastTxRequest{
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
		TxBytes: txBytes,
	}

	// Send tx
	return r.txClient.BroadcastTx(
		ctx,
		query,
	)
}

// Returns nil if the transaction is in a block
func (r *grpcClient) CheckConfirmed(ctx context.Context, txHash string) error {
	status, err := r.getTxStatus(ctx, txHash)
	if err != nil {
		r.log.Error().Err(err).Msg("Error querying tx status")
		return err
	} else {
		height := status.TxResponse.Height
		if height != 0 {
			r.log.Debug().Err(err).Str("tx_hash", txHash).Int64("height", height).Msg("Transaction confirmed")
			return nil
		} else {
			r.log.Debug().Msg("Transaction still not confirmed, still waiting...")
			return fmt.Errorf("transaction not yet confirmed")
		}
	}
}

func (r *grpcClient) getTxStatus(ctx context.Context, txHash string) (*txtypes.GetTxResponse, error) {
	request := &txtypes.GetTxRequest{Hash: txHash}
	return r.txClient.GetTx(ctx, request)
}

func (r *grpcClient) GetAccountData(ctx context.Context, address string) (*AccountData, error) {
	var accountData *AccountData
	var err error

	err = retry.Do(func() error {
		accountData, err = r.getAccountData(ctx, address)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return accountData, err
}

// private function without retries
func (r *grpcClient) getAccountData(ctx context.Context, address string) (*AccountData, error) {
	// Make a query
	query := &authtypes.QueryAccountRequest{Address: address}
	res, err := r.authClient.Account(
		ctx,
		query,
	)
	if err != nil {
		return nil, err
	}

	// Deserialize response
	var account authtypes.AccountI
	if err := r.cdc.UnpackAny(res.Account, &account); err != nil {
		return nil, err
	}

	return &AccountData{
		Address:       address,
		AccountNumber: account.GetAccountNumber(),
		Sequence:      account.GetSequence(),
	}, nil
}

func (r *grpcClient) SimulateTx(
	ctx context.Context,
	tx authsigning.Tx,
	txConfig client.TxConfig,
	gasFactor float64,
) (*SimulationResult, error) {
	var simulationResult *SimulationResult
	var err error

	err = retry.Do(func() error {
		simulationResult, err = r.simulateTx(ctx, tx, txConfig, gasFactor)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return simulationResult, err
}

// private function without retries
func (r *grpcClient) simulateTx(
	ctx context.Context,
	tx authsigning.Tx,
	txConfig client.TxConfig,
	gasFactor float64,
) (*SimulationResult, error) {
	// Form a query
	encoder := txConfig.TxEncoder()
	txBytes, err := encoder(tx)
	if err != nil {
		return nil, err
	}

	query := &txtypes.SimulateRequest{
		TxBytes: txBytes,
	}
	simulationResponse, err := r.txClient.Simulate(ctx, query)
	if err != nil {
		return nil, err
	}

	return &SimulationResult{
		GasRecommendation: uint64(math.Ceil(float64(simulationResponse.GasInfo.GasUsed) * gasFactor)),
	}, nil
}

func (r *grpcClient) GetGrants(ctx context.Context, botAddress string) ([]*authztypes.GrantAuthorization, error) {
	var grants []*authztypes.GrantAuthorization
	var err error

	err = retry.Do(func() error {
		grants, err = r.getGrants(ctx, botAddress)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return grants, err
}

// private function without retries
func (r *grpcClient) getGrants(ctx context.Context, botAddress string) ([]*authztypes.GrantAuthorization, error) {
	getGrantsFunc := func(ctx context.Context, pageKey []byte) (*paginatedRpcResponse[*authztypes.GrantAuthorization], error) {
		pagination := &query.PageRequest{
			Key:   pageKey,
			Limit: pageSize,
		}

		request := &authztypes.QueryGranteeGrantsRequest{
			Grantee:    botAddress,
			Pagination: pagination,
		}

		response, err := r.authzClient.GranteeGrants(ctx, request)
		if err != nil {
			return nil, err
		}

		return &paginatedRpcResponse[*authztypes.GrantAuthorization]{
			data:    response.Grants,
			nextKey: response.Pagination.NextKey,
		}, nil
	}

	grants, err := retrievePaginatedData(ctx, r, "grants", getGrantsFunc)
	if err != nil {
		r.log.Error().Err(err).Str("bot address", botAddress).Msg("Failed to retrieve grants")
	}
	r.log.Debug().Int("num grants", len(grants)).Str("bot address", botAddress).Msg("Retrieved grants")

	return grants, nil
}

func (r *grpcClient) GetDelegators(ctx context.Context, validatorAddress string) ([]string, error) {
	var delegators []string
	var err error

	err = retry.Do(func() error {
		delegators, err = r.getDelegators(ctx, validatorAddress)
		return err
	}, r.delay, r.attempts, retry.Context(ctx))
	if err != nil {
		err = errors.Unwrap(err)
	}

	return delegators, err
}

func (r *grpcClient) getDelegators(ctx context.Context, validatorAddress string) ([]string, error) {
	transformFunc := func(input stakingtypes.DelegationResponse) string { return input.Delegation.DelegatorAddress }

	fetchDelegatorPageFunc := func(ctx context.Context, pageKey []byte) (*paginatedRpcResponse[string], error) {
		pagination := &query.PageRequest{
			Key:   pageKey,
			Limit: pageSize,
		}

		request := &stakingtypes.QueryValidatorDelegationsRequest{
			ValidatorAddr: validatorAddress,
			Pagination:    pagination,
		}
		response, err := r.stakingClient.ValidatorDelegations(ctx, request)
		if err != nil {
			return nil, err
		}
		delegators := arrays.Map(response.DelegationResponses, transformFunc)

		return &paginatedRpcResponse[string]{
			data:    delegators,
			nextKey: response.Pagination.NextKey,
		}, nil
	}

	delegators, err := retrievePaginatedData(ctx, r, "delegations", fetchDelegatorPageFunc)
	if err != nil {
		return nil, err
	}
	r.log.Debug().Str("validator address", validatorAddress).Int("num delegators", len(delegators)).Msg("Retrieved delegations")

	return delegators, nil
}

// Pagination
// NOTE: Implemented as a private standalone func since go doesn't seem to support generics on struct methods.
func retrievePaginatedData[DataType any](
	ctx context.Context,
	r *grpcClient,
	noun string,
	retrievePageFn func(
		ctx context.Context,
		nextKey []byte,
	) (*paginatedRpcResponse[DataType], error),
) ([]DataType, error) {
	// Running list of data
	data := []DataType{}
	var err error

	// Loop through all pages
	var nextKey []byte
	for {
		// Query the page, retrying and then giving up if we exceed attempts
		var rpcResponse *paginatedRpcResponse[DataType]
		err = retry.Do(func() error {
			rpcResponse, err = retrievePageFn(ctx, nextKey)
			if err != nil {
				return err
			}
			return nil
		}, r.delay, r.attempts, retry.Context(ctx))
		if err != nil {
			err = errors.Unwrap(err)
		}

		if err != nil {
			return nil, err
		}

		// Append the data
		data = append(data, rpcResponse.data...)
		r.log.Debug().Int("num in page", len(rpcResponse.data)).Int("total fetched", len(data)).Msg(fmt.Sprintf("Fetched page of %s", noun))

		// Update next key or break out of loop if we have finished
		if len(rpcResponse.nextKey) == 0 {
			break
		}
		nextKey = rpcResponse.nextKey
	}

	return data, nil
}