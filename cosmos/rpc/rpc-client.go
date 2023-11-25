package rpc

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
)

// Handles RPCs for Restake
type RpcClient interface {
	Broadcast(ctx context.Context, txBytes []byte) (*txtypes.BroadcastTxResponse, error)
	CheckConfirmed(ctx context.Context, txHash string) error

	Simulate(ctx context.Context, txBytes []byte) (*txtypes.SimulateResponse, error)

	GetAccountData(ctx context.Context, address string) (*AccountData, error)
	GetBalance(ctx context.Context, address, denom string) (*sdk.Coin, error)
	GetDelegators(ctx context.Context, validatorAddress string) ([]string, error)
	GetGrants(ctx context.Context, botAddress string) ([]*authztypes.GrantAuthorization, error)
	GetPendingRewards(ctx context.Context, delegator, validator, stakingDenom string) (sdk.Dec, error)
}
