package registry

import (
	"encoding/json"
	"math"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AssetList struct {
	Schema    string  `json:"$schema"`
	ChainName string  `json:"chain_name"`
	Assets    []Asset `json:"assets"`
}

type Asset struct {
	Description string       `json:"description"`
	DenomUnits  []DenomUnit  `json:"denom_units"`
	Base        string       `json:"base"`
	Name        string       `json:"name"`
	Display     string       `json:"display"`
	Symbol      string       `json:"symbol"`
	LogoURIs    LogoURIs     `json:"logo_URIs"`
	CoingeckoID string       `json:"coingecko_id"`
	Images      []ImageLinks `json:"images"`
}

type DenomUnit struct {
	Denom    string `json:"denom"`
	Exponent int    `json:"exponent"`
}

type ImageLinks struct {
	Png string `json:"png"`
	Svg string `json:"svg"`
}

func parseAssetListResponse(assetListBytes []byte) (*AssetList, error) {
	// Unmarshal JSON data
	var assetList AssetList
	if err := json.Unmarshal(assetListBytes, &assetList); err != nil {
		return nil, err
	}
	return &assetList, nil
}

// Convenience methods

func (al *AssetList) ExtractAssetByBaseSymbol(baseAssetSymbol string) (*Asset, error) {
	assets := al.Assets
	for _, asset := range assets {
		if strings.EqualFold(asset.Base, baseAssetSymbol) {
			return &asset, nil
		}
	}
	return nil, ErrNoMatchingAsset
}

func (a *Asset) ExtractDenomByUnit(needleDenomUnit string) (*DenomUnit, error) {
	denomUnits := a.DenomUnits
	for _, denomUnit := range denomUnits {
		if strings.EqualFold(denomUnit.Denom, needleDenomUnit) {
			return &denomUnit, nil
		}
	}
	return nil, ErrNoMatchingDenom
}

// Get a single token, given a base symbol. Ex. 1 JUNO = sdk.Coin{ denom: "ujuno", amount: "1_000_000" }
func (al *AssetList) OneToken(baseAssetSymbol string) (*sdk.Coin, error) {
	// Extract the asset
	asset, err := al.ExtractAssetByBaseSymbol(baseAssetSymbol)
	if err != nil {
		return nil, err
	}

	// Extract the denom unit
	denomUnit, err := asset.ExtractDenomByUnit(baseAssetSymbol)
	if err != nil {
		return nil, err
	}

	decimals := denomUnit.Exponent

	// Create one coin.
	oneAmount := sdk.NewInt(int64(math.Pow(10, float64(decimals))))
	oneCoin := sdk.NewCoin(baseAssetSymbol, oneAmount)

	return &oneCoin, nil
}
