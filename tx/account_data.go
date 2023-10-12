package tx

import (
	"context"

	"github.com/tessellated-io/pickaxe/grpc"

	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type accountData struct {
	Address       string
	AccountNumber uint64
	Sequence      uint64
}

func getAccountData(nodeGrpcUri, address string, cdc *codec.ProtoCodec) (*accountData, error) {
	// Connect to gRPC
	conn, err := grpc.GetGrpcConnection(nodeGrpcUri)
	if err != nil {
		return nil, err
	}
	queryClient := authtypes.NewQueryClient(conn)

	// Make a query
	query := &authtypes.QueryAccountRequest{Address: address}
	res, err := queryClient.Account(
		context.Background(),
		query,
	)
	if err != nil {
		return nil, err
	}

	// Deserialize response
	var account authtypes.AccountI
	if err := cdc.UnpackAny(res.Account, &account); err != nil {
		return nil, err
	}

	return &accountData{
		Address:       address,
		AccountNumber: account.GetAccountNumber(),
		Sequence:      account.GetSequence(),
	}, nil
}
