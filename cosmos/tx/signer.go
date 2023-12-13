package tx

import (
	"fmt"

	"github.com/tessellated-io/pickaxe/crypto"
)

// Get a signer given a SLIP44 value.
func GetSoftSigner(slip44 uint, mnemonic string) (crypto.BytesSigner, error) {
	switch slip44 {
	case 564:
		return crypto.NewKeyPairFromMnemonic(mnemonic, 564), nil
	case 118:
		return crypto.NewCosmosKeyPairFromMnemonic(mnemonic), nil
	case 60:
		return crypto.NewEthermintKeyPairFromMnemonic(mnemonic), nil
	}

	return nil, fmt.Errorf("unknown slip44 value: %d", slip44)
}
