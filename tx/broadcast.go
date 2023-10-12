package tx

import (
	"context"

	"github.com/tessellated-io/pickaxe/grpc"

	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

func broadcastTx(
	txBytes []byte,
	broadcastMode txtypes.BroadcastMode,
	nodeGrpcUri string,
) (*txtypes.BroadcastTxResponse, error) {
	// Connect to gRPC
	conn, err := grpc.GetGrpcConnection(nodeGrpcUri)
	if err != nil {
		return nil, err
	}
	queryClient := txtypes.NewServiceClient(conn)

	// Form a query
	query := &txtypes.BroadcastTxRequest{
		Mode:    broadcastMode,
		TxBytes: txBytes,
	}
	return queryClient.BroadcastTx(
		context.Background(),
		query,
	)
}
