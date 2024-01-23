package registry_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	registry "github.com/tessellated-io/pickaxe/cosmos/chain-registry"
	"github.com/tessellated-io/pickaxe/log"
)

const chainsBaseUrl = "https://planetarium.tessellated.io/v1/chains"
const validatorsBaseUrl = "https://planetarium.tessellated.io/v1/validators"

func TestCanRetrieveChain(t *testing.T) {
	client := registry.NewChainRegistryClient(log.NewLogger(zerolog.FatalLevel), chainsBaseUrl, validatorsBaseUrl)

	hubInfo, err := client.ChainInfo(context.Background(), "cosmoshub")
	assert.Nil(t, err, "error should be nil")

	assert.Equal(t, hubInfo.ChainID, "cosmoshub-4", "incorrect chain id")
}

func TestCanRetrieveAssets(t *testing.T) {
	client := registry.NewChainRegistryClient(log.NewLogger(zerolog.FatalLevel), chainsBaseUrl, validatorsBaseUrl)

	_, err := client.AssetList(context.Background(), "cosmoshub")
	assert.Nil(t, err, "error should be nil")
}

func TestCanRetrieveAllChains(t *testing.T) {
	client := registry.NewChainRegistryClient(log.NewLogger(zerolog.FatalLevel), chainsBaseUrl, validatorsBaseUrl)

	_, err := client.AllChainNames(context.Background())
	assert.Nil(t, err, "error should be nil")
}
