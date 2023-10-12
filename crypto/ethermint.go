package crypto

import (
	btcec "github.com/btcsuite/btcd/btcec/v2"
	"github.com/evmos/evmos/v14/crypto/hd"
	"golang.org/x/crypto/sha3"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

type EthermintKeyPair struct {
	Public  cryptotypes.PubKey
	Private cryptotypes.PrivKey
}

var _ BytesSigner = (*KeyPair)(nil)

// Return a key pair derived from the given mnemonic with
func NewEthermintKeyPairFromMnemonic(mnemonic string) *EthermintKeyPair {
	// create master key and derive first key for keyring
	algo := hd.EthSecp256k1
	derivedPriv, _ := algo.Derive()(mnemonic, keyring.DefaultBIP39Passphrase, "m/44'/60'/0'/0/0")
	privKey := algo.Generate()(derivedPriv)

	pubKey := privKey.PubKey()

	return &EthermintKeyPair{
		Public:  pubKey,
		Private: privKey,
	}
}

func (e *EthermintKeyPair) GetAddress(prefix string) string {
	compressedPublicKey := e.Public
	parsed, err := btcec.ParsePubKey(compressedPublicKey.Bytes())
	decompressedPublicKey := parsed.SerializeUncompressed()
	if err != nil {
		panic(err)
	}

	hash := sha3.NewLegacyKeccak256()
	hash.Write(decompressedPublicKey[1:]) // Remove the prefix byte from the uncompressed public key
	addressBytes := hash.Sum(nil)[12:]

	address := sdk.AccAddress(addressBytes)
	encoded, _ := bech32.ConvertAndEncode(prefix, address)
	return encoded
}

func (e *EthermintKeyPair) SignBytes(
	bytesToSign []byte,
) ([]byte, error) {
	return e.Private.Sign(bytesToSign)
}

func (e *EthermintKeyPair) GetPublicKey() cryptotypes.PubKey {
	return e.Public
}
