package rpc

import (
	"context"
	"fmt"
	"strings"

	"github.com/tessellated-io/pickaxe/arrays"
	"github.com/tessellated-io/pickaxe/cosmos/util"
	"github.com/tessellated-io/pickaxe/grpc"
	"github.com/tessellated-io/pickaxe/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
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
	bankClient         banktypes.QueryClient
	distributionClient distributiontypes.QueryClient
	stakingClient      stakingtypes.QueryClient
	txClient           txtypes.ServiceClient

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
func NewGrpcClient(nodeGrpcUri string, cdc *codec.ProtoCodec, log *log.Logger) (RpcClient, error) {
	conn, err := grpc.GetGrpcConnection(nodeGrpcUri)
	if err != nil {
		log.Error().Str("grpc url", nodeGrpcUri).Err(err).Msg("Unable to connect to gRPC")
		return nil, err
	}

	authClient := authtypes.NewQueryClient(conn)
	authzClient := authztypes.NewQueryClient(conn)
	bankClient := banktypes.NewQueryClient(conn)
	distributionClient := distributiontypes.NewQueryClient(conn)
	stakingClient := stakingtypes.NewQueryClient(conn)
	txClient := txtypes.NewServiceClient(conn)

	return &grpcClient{
		cdc: cdc,

		authClient:         authClient,
		authzClient:        authzClient,
		bankClient:         bankClient,
		distributionClient: distributionClient,
		stakingClient:      stakingClient,
		txClient:           txClient,

		log: log,
	}, nil
}

func (r *grpcClient) GetBalance(ctx context.Context, address, denom string) (*sdk.Coin, error) {
	getBalancesFunc := func(ctx context.Context, pageKey []byte) (*paginatedRpcResponse[sdk.Coin], error) {
		pagination := &query.PageRequest{
			Key:   pageKey,
			Limit: pageSize,
		}

		request := &banktypes.QueryAllBalancesRequest{
			Address:    address,
			Pagination: pagination,
		}

		response, err := r.bankClient.AllBalances(ctx, request)
		if err != nil {
			return nil, err
		}

		return &paginatedRpcResponse[sdk.Coin]{
			data:    response.Balances,
			nextKey: response.Pagination.NextKey,
		}, nil
	}

	balances, err := retrievePaginatedData(ctx, r, "balances", getBalancesFunc)
	if err != nil {
		return nil, err
	}
	r.log.Debug().Int("num_balances", len(balances)).Str("address", address).Str("denom", denom).Msg("Retrieved balances")

	return util.ExtractCoin(denom, balances)
}

func (r *grpcClient) GetPendingRewards(ctx context.Context, delegator, validator, stakingDenom string) (sdk.Dec, error) {
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

func (r *grpcClient) CheckIncluded(ctx context.Context, txHash string) (bool, error) {
	txStatus, err := r.getTxStatus(ctx, txHash)
	if err != nil {
		// Check if the error was gRPC not found
		grpcErr, ok := status.FromError(err)
		if ok && grpcErr.Code() == codes.NotFound {
			return false, nil
		} else if ok {
			// TODO: Deleteme
			fmt.Println("FYI, unwrapped grpc error to ", grpcErr)
		}

		return false, err
	}

	height := txStatus.TxResponse.Height
	if height != 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func (r *grpcClient) getTxStatus(ctx context.Context, txHash string) (*txtypes.GetTxResponse, error) {
	request := &txtypes.GetTxRequest{Hash: txHash}
	return r.txClient.GetTx(ctx, request)
}

func (r *grpcClient) Account(ctx context.Context, address string) (authtypes.AccountI, error) {
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

	return account, nil
}

func (r *grpcClient) Simulate(
	ctx context.Context,
	txBytes []byte,
) (*txtypes.SimulateResponse, error) {
	// Form a query
	query := &txtypes.SimulateRequest{
		TxBytes: txBytes,
	}
	simulationResponse, err := r.txClient.Simulate(ctx, query)
	if err != nil {
		return nil, err
	}

	return simulationResponse, nil
}

func (r *grpcClient) GetDenomMetadata(ctx context.Context, denom string) (*banktypes.Metadata, error) {
	query := &banktypes.QueryDenomMetadataRequest{
		Denom: denom,
	}
	response, err := r.bankClient.DenomMetadata(ctx, query)
	if err != nil {
		return nil, err
	}

	return &response.Metadata, nil
}

func (r *grpcClient) GetGrants(ctx context.Context, botAddress string) ([]*authztypes.GrantAuthorization, error) {
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
		return nil, err
	}
	r.log.Debug().Int("num grants", len(grants)).Str("bot address", botAddress).Msg("Retrieved grants")

	return grants, nil
}

func (r *grpcClient) GetDelegators(ctx context.Context, validatorAddress string) ([]string, error) {
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

	// Loop through all pages
	var nextKey []byte
	for {
		// Query the page, retrying and then giving up if we exceed attempts
		rpcResponse, err := retrievePageFn(ctx, nextKey)
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
