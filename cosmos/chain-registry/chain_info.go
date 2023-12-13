package registry

import (
	"encoding/json"
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
