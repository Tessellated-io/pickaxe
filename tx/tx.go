package tx

import (
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	txauth "github.com/cosmos/cosmos-sdk/x/auth/tx"
)

func SendMessages(
	msgs []sdk.Msg,
	addressPrefix string,
	chainID string,
	cdc *codec.ProtoCodec,
	signer bytesSigner,
	broadcastMode txtypes.BroadcastMode,
	nodeGrpcUri string,
	gasFactor float64,
	gasPrice float64,
	feeDenom string,
) (*txtypes.BroadcastTxResponse, error) {
	// Get account data
	address := signer.GetAddress(addressPrefix)
	accountData, err := getAccountData(nodeGrpcUri, address, cdc)
	if err != nil {
		return nil, err
	}

	// Start building a tx
	txConfig := txauth.NewTxConfig(cdc, txauth.DefaultSignModes)
	factory := cosmostx.Factory{}.WithChainID(chainID).WithTxConfig(txConfig)
	txb, err := factory.BuildUnsignedTx(msgs...)
	if err != nil {
		return nil, err
	}

	signatureProto := signing.SignatureV2{
		PubKey: signer.GetPublicKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: accountData.Sequence,
	}
	err = txb.SetSignatures(signatureProto)
	if err != nil {
		return nil, err
	}

	// Simulate the tx
	simulationResult, err := simulateTx(txb.GetTx(), gasFactor, nodeGrpcUri, txConfig)
	if err != nil {
		return nil, err
	}
	txb.SetGasLimit(simulationResult.GasRecommendation)

	fee := []sdk.Coin{
		{
			Denom:  feeDenom,
			Amount: sdk.NewInt(int64(gasPrice*float64(simulationResult.GasRecommendation)) + 1),
		},
	}
	txb.SetFeeAmount(fee)

	// Sign the tx
	signedTx, err := signTx(txb, signer, accountData, chainID, txConfig)
	if err != nil {
		panic(err)
	}

	return broadcastTx(signedTx, broadcastMode, nodeGrpcUri)
}
