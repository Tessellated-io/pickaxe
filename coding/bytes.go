package coding

import (
	"encoding/hex"
	"fmt"
)

func UnsafeHexToBytes(input string) []byte {
	bytes, err := hex.DecodeString(input)
	if err != nil {
		fmt.Printf(
			"Warning: failed to decode hex string. Returning anyway, but this will likely result in a downstream error. Error: %s\nHex: %s\n",
			err,
			input,
		)
	}

	return bytes
}
