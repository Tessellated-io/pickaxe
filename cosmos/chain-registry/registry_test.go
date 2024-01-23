package registry_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	registry "github.com/tessellated-io/pickaxe/cosmos/chain-registry"
	"github.com/tessellated-io/pickaxe/log"
)

func TestChainRegistryIsAlive(t *testing.T) {
	client := registry.NewChainRegistryClient(log.NewLogger(zerolog.FatalLevel))

	hubInfo, err := client.ChainInfo(context.Background(), "cosmoshub")
	assert.Nil(t, err, "error should be nil")

	assert.Equal(t, hubInfo.ChainID, "cosmoshub-4", "incorrect chain id")
}
