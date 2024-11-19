package coding

import (
	"encoding/hex"
	"fmt"
	"strings"
)

func DecodeHex(in string) ([]byte, error) {
	normalized := in
	if strings.HasPrefix(in, "0x") || strings.HasPrefix(in, "0X") {
		normalized = normalized[2:]
	}

	return hex.DecodeString(normalized)
}

func NormalizeBytesToHex(input []byte) string {
	return strings.ToLower("0x" + hex.EncodeToString(input))
}

// PayloadFingerprint pretty prints a hex payload in an identifiable and succint way.
func PayloadFingerprint(payload []byte) string {
	if len(payload) == 0 {
		return NormalizeMaybeEmptyBytes(payload)
	}

	return fmt.Sprintf("[%s...%s]", hex.EncodeToString(payload[0:4]), hex.EncodeToString(payload[len(payload)-4:]))
}

// Returns an empty byte slice rather than no output for empty byte arrays
func NormalizeMaybeEmptyBytes(bytes []byte) string {
	if len(bytes) > 0 {
		return hex.EncodeToString(bytes)
	}
	return "[]"
}
