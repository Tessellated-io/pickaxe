package crypto

import cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

type BytesSigner interface {
	GetAddress(prefix string) string
	SignBytes(
		bytesToSign []byte,
	) ([]byte, error)
	GetPublicKey() cryptotypes.PubKey
}
