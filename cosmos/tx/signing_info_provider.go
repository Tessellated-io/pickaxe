package tx

import (
	"context"

	"github.com/tessellated-io/pickaxe/cosmos/rpc"
)

type SigningInfoProvider struct {
	rpcClient rpc.RpcClient
}

func NewSigningInfoProvider(rpcClient rpc.RpcClient) (*SigningInfoProvider, error) {
	return &SigningInfoProvider{
		rpcClient: rpcClient,
	}, nil
}

func (sip *SigningInfoProvider) SigningMetadataForAccount(ctx context.Context, address string) (*SigningMetadata, error) {
	account, err := sip.rpcClient.Account(ctx, address)
	if err != nil {
		return nil, err
	}

	return &SigningMetadata{
		Address:       address,
		AccountNumber: account.GetAccountNumber(),
		Sequence:      account.GetSequence(),
	}, nil
}
