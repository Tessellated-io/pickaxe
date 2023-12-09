package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/tessellated-io/pickaxe/cosmos/rpc"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func parseChainResponse(responseBytes []byte) (*ChainInfo, error) {
	// Unmarshal the JSON data into the ChainInfo struct
	var chainInfo ChainInfo
	err := json.Unmarshal(responseBytes, &chainInfo)
	if err != nil {
		return nil, err
	}
	return &chainInfo, nil
}

// Convenience helper methods

func (ci *ChainInfo) FeeDenom() (string, error) {
	feeToken, err := ci.FeeToken()
	if err != nil {
		return "", nil
	}

	return feeToken.Denom, nil
}

func (ci *ChainInfo) OneFeeToken(ctx context.Context, rpcClient rpc.RpcClient) (*sdk.Coin, error) {
	feeDenom, err := ci.FeeDenom()
	if err != nil {
		return nil, err
	}

	// Fetch denom metadata from the registry to get decimals. It's a bummer chain registry doesn't seem to include this field.
	denomMetadata, err := rpcClient.GetDenomMetadata(ctx, feeDenom)
	if err != nil {
		return nil, err
	}
	denomUnits := denomMetadata.DenomUnits
	if len(denomUnits) == 0 {
		return nil, fmt.Errorf("no denom unit found for %s", feeDenom)
	}
	decimals := denomUnits[0].Exponent

	// Create one coin.
	oneAmount := sdk.NewInt(int64(math.Pow(10, float64(decimals))))
	oneCoin := sdk.NewCoin(feeDenom, oneAmount)

	return &oneCoin, nil
}

func (ci *ChainInfo) StakingDenom() (string, error) {
	stakingTokens := ci.Staking.StakingTokens
	if len(stakingTokens) == 0 {
		return "", ErrNoStakingTokenFound
	}

	return stakingTokens[0].Denom, nil
}

func (ci *ChainInfo) FeeToken() (*FeeToken, error) {
	feeTokens := ci.Fees.FeeTokens
	if len(feeTokens) == 0 {
		return nil, ErrNoFeeTokenFound
	}
	return &feeTokens[0], nil
}

func (ci *ChainInfo) MinGasFee() (float64, error) {
	feeToken, err := ci.FeeToken()
	if err != nil {
		return 0, err
	}

	return feeToken.FixedMinGasPrice, nil
}
