package tx

import (
	"context"

	"github.com/tessellated-io/pickaxe/cosmos/rpc"
)

type SigningMetadataProvider struct {
	rpcClient rpc.RpcClient
}

func NewSigningMetadataProvider(rpcClient rpc.RpcClient) (*SigningMetadataProvider, error) {
	return &SigningMetadataProvider{
		rpcClient: rpcClient,
	}, nil
}

func (sip *SigningMetadataProvider) SigningMetadataForAccount(ctx context.Context, address string) (*SigningMetadata, error) {
	account, err := sip.rpcClient.Account(ctx, address)
	if err != nil {
		return nil, err
	}

	return &SigningMetadata{
		address:       address,
		accountNumber: account.GetAccountNumber(),
		sequence:      account.GetSequence(),
	}, nil
}
