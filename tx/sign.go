package tx

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

// TODO: Consider making this return a pointer to bytes
func signTx(
	txb client.TxBuilder,
	signer bytesSigner,
	accountData *accountData,
	chainID string,
	txConfig client.TxConfig,
) ([]byte, error) {
	// Form signing data
	signerData := authsigning.SignerData{
		ChainID:       chainID,
		Sequence:      accountData.Sequence,
		AccountNumber: accountData.AccountNumber,
	}

	// Encode to bytes to sign
	signMode := signing.SignMode_SIGN_MODE_DIRECT
	unsignedTxBytes, err := txConfig.SignModeHandler().GetSignBytes(signMode, signerData, txb.GetTx())
	if err != nil {
		return []byte{}, err
	}

	// Sign the bytes
	signatureBytes, err := signer.SignBytes(unsignedTxBytes)
	if err != nil {
		return []byte{}, err
	}

	// Reconstruct the signature proto
	signatureData := &signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: signatureBytes,
	}
	signatureProto := signing.SignatureV2{
		PubKey:   signer.GetPublicKey(),
		Data:     signatureData,
		Sequence: accountData.Sequence,
	}
	err = txb.SetSignatures(signatureProto)
	if err != nil {
		return []byte{}, err
	}

	// Encode to bytes
	encoder := txConfig.TxEncoder()
	return encoder(txb.GetTx())
}
