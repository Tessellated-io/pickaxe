package util

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
)

func NumberToBigInt(num json.Number) (*big.Int, error) {
	if _, err := strconv.Atoi(string(num)); err == nil {
		n := new(big.Int)
		n.SetString(string(num), 10)
		return n, nil
	} else {
		return nil, fmt.Errorf("unexpected floating point value for min rewards: %s", num)
	}
}
