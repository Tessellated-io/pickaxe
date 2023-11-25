package tx

import (
	"context"
	"math"

	"github.com/cosmos/cosmos-sdk/client"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/tessellated-io/pickaxe/cosmos/rpc"
)

// SimulationManager manages simulating gas from transactions.
type SimulationManager interface {
	SimulateTx(ctx context.Context, tx authsigning.Tx) (*SimulationResult, error)
	SimulateTxBytes(ctx context.Context, txBytes []byte) (*SimulationResult, error)
}

// simulationManager is the default implementation
type simulationManager struct {
	gasFactor float64

	txConfig  client.TxConfig
	rpcClient rpc.RpcClient
}

// Ensure type conformance
var _ SimulationManager = (*simulationManager)(nil)

// NewSimulationManager makes a new default simulationManager
func NewSimulationManager(gasFactor float64, rpcClient rpc.RpcClient, txConfig client.TxConfig) (SimulationManager, error) {
	return &simulationManager{
		gasFactor: gasFactor,

		txConfig:  txConfig,
		rpcClient: rpcClient,
	}, nil
}

// Simulation Managar interface

func (sm *simulationManager) SimulateTx(ctx context.Context, tx authsigning.Tx) (*SimulationResult, error) {
	// Form transaction bytes
	encoder := sm.txConfig.TxEncoder()
	txBytes, err := encoder(tx)
	if err != nil {
		return nil, err
	}

	return sm.SimulateTxBytes(ctx, txBytes)
}

func (sm *simulationManager) SimulateTxBytes(ctx context.Context, txBytes []byte) (*SimulationResult, error) {
	simulationResponse, err := sm.rpcClient.Simulate(ctx, txBytes)
	if err != nil {
		return nil, err
	}

	return &SimulationResult{
		GasRecommendation: uint64(math.Ceil(float64(simulationResponse.GasInfo.GasUsed) * sm.gasFactor)),
	}, nil
}
