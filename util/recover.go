package util

import (
	"errors"
	"fmt"
)

func InterfaceToError(errorInterface interface{}) error {
	// Attempt to coerce into error
	err, ok := errorInterface.(error)
	if ok {
		return err
	}

	// Otherwise attempt to coerce into string
	stringifiedErr, ok := errorInterface.(string)
	if ok {
		return errors.New(stringifiedErr)
	}

	// Otherwise, just ditch with a generic error.
	return fmt.Errorf("recovered from a panic that was neither a string nor an error")
}
