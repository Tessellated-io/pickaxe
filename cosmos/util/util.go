package util

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func ExtractCoin(targetDenom string, coins []sdk.Coin) (*sdk.Coin, error) {
	for _, coin := range coins {
		if strings.EqualFold(targetDenom, coin.Denom) {
			return &coin, nil
		}
	}
	return nil, fmt.Errorf("unable not find denom: %s", targetDenom)
}
