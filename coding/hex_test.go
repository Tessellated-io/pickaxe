package coding_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tessellated-io/pickaxe/coding"
)

func TestDecodeHex(t *testing.T) {
	expected := []byte{0x01, 0x02, 0x03, 0x04, 0xa, 0xb}

	cases := []string{
		"010203040a0b",
		"0x010203040a0b",
		"0X010203040A0B",
	}

	for _, in := range cases {
		decoded, err := coding.DecodeHex(in)
		assert.Nil(t, err, "should not have an error")
		assert.Equal(t, expected, decoded, "unexpected decoded value")
	}
}

func TestNormalizeHex(t *testing.T) {
	input := []byte{0x00, 0x01, 0x10, 0x11}

	normalized := coding.NormalizeBytesToHex(input)

	assert.Equal(t, "0x00011011", normalized)
}
