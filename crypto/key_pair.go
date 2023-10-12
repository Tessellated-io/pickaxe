package crypto

import (
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

type KeyPair struct {
	Public  cryptotypes.PubKey
	Private cryptotypes.PrivKey
}

var _ BytesSigner = (*KeyPair)(nil)

// NewCosmosKeyPairFromMnemonic returns a key pair derived from the given mnemonic, with coin type 118 (cosmos)
func NewCosmosKeyPairFromMnemonic(mnemonic string) *KeyPair {
	bip44Path := sdk.GetConfig().GetFullBIP44Path()
	return newKeyPairFromMnemonic(mnemonic, bip44Path)
}

// Return a key pair derived from the given mnemonic with
func NewKeyPairFromMnemonic(mnemonic string, coinType uint32) *KeyPair {
	// Futz with the config but reset it so that we don't get confused in future calls
	config := sdk.GetConfig()
	config.SetCoinType(coinType)
	bip44Path := config.GetFullBIP44Path()
	config.SetCoinType(118)

	return newKeyPairFromMnemonic(mnemonic, bip44Path)
}

func newKeyPairFromMnemonic(mnemonic, bip44Path string) *KeyPair {
	// create master key and derive first key for keyring
	algo := hd.Secp256k1
	derivedPriv, _ := algo.Derive()(mnemonic, keyring.DefaultBIP39Passphrase, bip44Path)
	privKey := algo.Generate()(derivedPriv)
	pubKey := privKey.PubKey()

	return &KeyPair{
		Public:  pubKey,
		Private: privKey,
	}
}

func (kp *KeyPair) GetAddress(prefix string) string {
	address := sdk.AccAddress(kp.Public.Address())
	encoded, _ := bech32.ConvertAndEncode(prefix, address)
	return encoded
}

func (kp *KeyPair) SignBytes(
	bytesToSign []byte,
) ([]byte, error) {
	return kp.Private.Sign(bytesToSign)
}

func (kp *KeyPair) GetPublicKey() cryptotypes.PubKey {
	return kp.Public
}
