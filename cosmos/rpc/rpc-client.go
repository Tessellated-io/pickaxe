package rpc

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Handles RPCs for Restake
type RpcClient interface {
	Broadcast(ctx context.Context, txBytes []byte) (*txtypes.BroadcastTxResponse, error)
	CheckConfirmed(ctx context.Context, txHash string) error

	Simulate(ctx context.Context, txBytes []byte) (*txtypes.SimulateResponse, error)

	Account(ctx context.Context, address string) (authtypes.AccountI, error)

	GetBalance(ctx context.Context, address, denom string) (*sdk.Coin, error)
	GetDelegators(ctx context.Context, validatorAddress string) ([]string, error)
	GetDenomMetadata(ctx context.Context, denom string) (*banktypes.Metadata, error)
	GetGrants(ctx context.Context, botAddress string) ([]*authztypes.GrantAuthorization, error)
	GetPendingRewards(ctx context.Context, delegator, validator, stakingDenom string) (sdk.Dec, error)
}
