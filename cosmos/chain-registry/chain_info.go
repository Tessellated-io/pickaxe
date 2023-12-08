package registry

import (
	"encoding/json"

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

func (ci *ChainInfo) FeeDenom(chainInfo *ChainInfo) (string, error) {

	return gasTokens[0].Denom
}

func (ci *ChainInfo) OneToken() (sdk.Coin, error) {

}

func (ci *ChainInfo) StakingDenom() (string, error) {
	stakingTokens := ci.Staking.StakingTokens
	if len(stakingTokens) == 0 {
		return "", ErrNoStakingTokenFound
	}

	return stakingTokens[0].Denom, nil
}

func (ci *ChainInfo) FeeToken() (*sdk.Coin, error) {
	feeTokens := ci.Fees.FeeTokens
	if len(feeTokens) == 0 {
		return nil, ErrNoFeeTokenFound
	}
	return feeTokens[0], nil
}
