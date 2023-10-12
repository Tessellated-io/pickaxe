package tx

import cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

type bytesSigner interface {
	GetPublicKey() cryptotypes.PubKey
	SignBytes(bytes []byte) ([]byte, error)
	GetAddress(prefix string) string
}
