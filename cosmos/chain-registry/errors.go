package registry

import "errors"

var (
	ErrNoChainFoundForChainID = errors.New("no chain found for chain ID")
	ErrNoStakingTokenFound    = errors.New("no staking tokens found in registry")
	ErrNoFeeTokenFound        = errors.New("no fee tokens found in registry")
	ErrNoMatchingAsset        = errors.New("no matching asset found")
	ErrNoMatchingDenom        = errors.New("no matching denom found")
)
