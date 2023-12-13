package tx

import (
	"context"

	"github.com/tessellated-io/pickaxe/crypto"
	"github.com/tessellated-io/pickaxe/log"

	"github.com/cosmos/cosmos-sdk/client"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

type TxProvider interface {
	ProvideTx(ctx context.Context, gasPrice float64, messages []sdk.Msg, metadata *SigningMetadata) ([]byte, uint64, error)
}

// txProvider is the default implementation of the Signer interface
type txProvider struct {
	bytesSigner crypto.BytesSigner
	feeDenom    string
	memo        string

	logger            *log.Logger
	simulationManager SimulationManager

	txConfig  client.TxConfig
	txFactory cosmostx.Factory
}

// Assert type conformance
var _ TxProvider = (*txProvider)(nil)

func NewTxProvider(bytesSigner crypto.BytesSigner, chainID, feeDenom, memo string, logger *log.Logger, simulationManager SimulationManager, txConfig client.TxConfig) (TxProvider, error) {
	txFactory := cosmostx.Factory{}.WithChainID(chainID).WithTxConfig(txConfig)

	return &txProvider{
		bytesSigner: bytesSigner,
		feeDenom:    feeDenom,
		memo:        memo,

		logger:            logger,
		simulationManager: simulationManager,

		txConfig:  txConfig,
		txFactory: txFactory,
	}, nil
}

// Signer Interface

// Sign returns the set of messages, encoded with metadata, and includes a valid signature.
// It also includes the gas that was desired. This API is kinda nuts, but I can't find a sane way around it.
func (txp *txProvider) ProvideTx(ctx context.Context, gasPrice float64, messages []sdk.Msg, metadata *SigningMetadata) ([]byte, uint64, error) {
	txp.logger.Debug().Str("chain_id", metadata.chainID).Str("account", metadata.address).Uint64("sequence", metadata.sequence).Uint64("account_number", metadata.accountNumber).Msg("preparing to sign transaction")

	// Build a transaction
	txb, err := txp.txFactory.BuildUnsignedTx(messages...)
	if err != nil {
		return nil, 0, err
	}

	txb.SetMemo(txp.memo)
	signatureProto := signing.SignatureV2{
		PubKey: txp.bytesSigner.GetPublicKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: metadata.Sequence(),
	}
	err = txb.SetSignatures(signatureProto)
	if err != nil {
		return nil, 0, err
	}

	// Simulate the tx
	simulationResult, err := txp.simulationManager.SimulateTx(ctx, txb.GetTx())
	if err != nil {
		return nil, 0, err
	}
	txp.logger.Debug().Uint64("gas_units", simulationResult.GasRecommendation).Msg("simulated gas")
	txb.SetGasLimit(simulationResult.GasRecommendation)

	fee := []sdk.Coin{
		{
			Denom:  txp.feeDenom,
			Amount: sdk.NewInt(int64(gasPrice*float64(simulationResult.GasRecommendation)) + 1),
		},
	}
	txb.SetFeeAmount(fee)

	// Shim metadata into the format Cosmos SDK wants
	signerData := authsigning.SignerData{
		ChainID:       metadata.ChainID(),
		Sequence:      metadata.Sequence(),
		AccountNumber: metadata.AccountNumber(),
	}

	// Encode to bytes to sign
	signMode := signing.SignMode_SIGN_MODE_DIRECT
	unsignedTxBytes, err := txp.txConfig.SignModeHandler().GetSignBytes(signMode, signerData, txb.GetTx())
	if err != nil {
		return nil, 0, err
	}

	// Sign the bytes
	signatureBytes, err := txp.bytesSigner.SignBytes(unsignedTxBytes)
	if err != nil {
		return nil, 0, err
	}

	// Reconstruct the signature proto
	signatureData := &signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: signatureBytes,
	}
	signatureProto = signing.SignatureV2{
		PubKey:   txp.bytesSigner.GetPublicKey(),
		Data:     signatureData,
		Sequence: metadata.Sequence(),
	}
	err = txb.SetSignatures(signatureProto)
	if err != nil {
		return []byte{}, 0, err
	}

	// Encode to bytes
	encoder := txp.txConfig.TxEncoder()
	txBytes, err := encoder(txb.GetTx())
	if err != nil {
		return nil, 0, err
	}

	return txBytes, simulationResult.GasRecommendation, nil
}
