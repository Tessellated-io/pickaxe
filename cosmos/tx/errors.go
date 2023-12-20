package tx

import "errors"

var (
	ErrNoGasPrice  = errors.New("no known gas price")
	ErrNoGasFactor = errors.New("no known gas factor")
)
