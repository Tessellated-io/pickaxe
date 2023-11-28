package registry

import "context"

type ChainRegistryClient interface {
	AllChainNames(ctx context.Context) ([]string, error)
	ChainNameForChainID(ctx context.Context, targetChainID string, refreshCache bool) (string, error)
	GetChainInfo(ctx context.Context, chainName string) (*ChainInfo, error)
}
