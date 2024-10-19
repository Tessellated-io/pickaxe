package registry_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	registry "github.com/tessellated-io/pickaxe/cosmos/chain-registry"
)

const chainsBaseUrl = "https://planetarium.tessellated.io/v1/chains"
const validatorsBaseUrl = "https://planetarium.tessellated.io/v1/validators"

func TestCanRetrieveChain(t *testing.T) {
	client := registry.NewChainRegistryClient(slog.Default(), chainsBaseUrl, validatorsBaseUrl)

	hubInfo, err := client.ChainInfo(context.Background(), "cosmoshub")
	assert.Nil(t, err, "error should be nil")

	assert.Equal(t, hubInfo.ChainID, "cosmoshub-4", "incorrect chain id")
}

func TestCanRetrieveAssets(t *testing.T) {
	client := registry.NewChainRegistryClient(lslog.Default(), chainsBaseUrl, validatorsBaseUrl)

	_, err := client.AssetList(context.Background(), "cosmoshub")
	assert.Nil(t, err, "error should be nil")
}

func TestCanRetrieveAllChains(t *testing.T) {
	client := registry.NewChainRegistryClient(slog.Default(), chainsBaseUrl, validatorsBaseUrl)

	_, err := client.AllChainNames(context.Background())
	assert.Nil(t, err, "error should be nil")
}

func TestCanRetrieveValidator(t *testing.T) {
	client := registry.NewChainRegistryClient(slog.Default(), chainsBaseUrl, validatorsBaseUrl)

	_, err := client.Validator(context.Background(), "tessellated")
	assert.Nil(t, err, "error should be nil")
}
