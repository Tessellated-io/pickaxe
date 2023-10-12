package tx

import (
	"context"
	"math"

	"github.com/tessellated-io/pickaxe/grpc"

	"github.com/cosmos/cosmos-sdk/client"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

type simulationResult struct {
	GasRecommendation uint64
}

func simulateTx(
	tx authsigning.Tx,
	gasFactor float64,
	nodeGrpcUri string,
	txConfig client.TxConfig,
) (*simulationResult, error) {
	// Connect to gRPC
	conn, err := grpc.GetGrpcConnection(nodeGrpcUri)
	if err != nil {
		return nil, err
	}
	queryclient := txtypes.NewServiceClient(conn)

	// Form a query
	encoder := txConfig.TxEncoder()
	txBytes, err := encoder(tx)
	if err != nil {
		return nil, err
	}

	query := &txtypes.SimulateRequest{
		TxBytes: txBytes,
	}
	simulationResponse, err := queryclient.Simulate(context.Background(), query)
	if err != nil {
		return nil, err
	}

	return &simulationResult{
		GasRecommendation: uint64(math.Ceil(float64(simulationResponse.GasInfo.GasUsed) * gasFactor)),
	}, nil
}
