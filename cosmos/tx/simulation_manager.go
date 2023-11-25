package tx

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/tessellated-io/pickaxe/cosmos/rpc"
)

// SimulationManager manages simulating gas from transactions.
type SimulationManager interface {
	SimulateTx(ctx context.Context, tx authsigning.Tx, txConfig client.TxConfig, gasFactor float64) (*SimulationResult, error)
}

// simulationManager is the default implementation
type simulationManager struct {
	rpcClient rpc.RpcClient
}

// Ensure type conformance
var _ SimulationManager = (*simulationManager)(nil)

// NewSimulationManager makes a new default simulationManager
func NewSimulationManager(rpcClient rpc.RpcClient) (SimulationManager, error) {
	return &simulationManager{
		rpcClient: rpcClient,
	}, nil
}

// Simulation Managar interface

func (sm *simulationManager) SimulateTx(ctx context.Context, tx authsigning.Tx, txConfig client.TxConfig, gasFactor float64) (*SimulationResult, error) {
	
}
