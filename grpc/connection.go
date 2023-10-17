package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func GetGrpcConnection(grpcUri string) (*grpc.ClientConn, error) {
	// Handle connections using SSL
	transportCredentials := grpc.WithTransportCredentials(insecure.NewCredentials())
	if strings.HasSuffix(grpcUri, "443") {
		// Load the root certificates for TLS. Since you don't know the server, you can use an empty pool.
		certPool := x509.NewCertPool()

		// Connect to the gRPC server over TLS with insecure SkipVerify options
		creds := credentials.NewTLS(&tls.Config{
			RootCAs:    certPool,
			MinVersion: tls.VersionTLS12,
		})
		transportCredentials = grpc.WithTransportCredentials(creds)
	}

	// Set up gRPC dial options with custom keep alive and timeout values
	opts := []grpc.DialOption{
		transportCredentials,
	}

	return grpc.Dial(
		grpcUri,
		opts...,
	)
}
