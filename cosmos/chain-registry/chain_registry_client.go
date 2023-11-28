package registry

import "context"

type ChainRegistryClient interface {
	// Cosmos Chain Registry
	AllChainNames(ctx context.Context) ([]string, error)
	ChainNameForChainID(ctx context.Context, targetChainID string, refreshCache bool) (string, error)
	ChainInfo(ctx context.Context, chainName string) (*ChainInfo, error)

	// Restake Validator Registry
	Validator(ctx context.Context, targetValidator string) (*Validator, error)
}
