package util

import "fmt"

// Returns an error that represents a panic, if there was a panic. Otherwise, returns nil.
func ErrorFromRecovery() error {
	// Attempt recovery and ditch if there's nothing to recover from.
	recovered := recover()
	if recovered == nil {
		return nil
	}

	// Attempt to coerce into error
	err, ok := recovered.(error)
	if ok {
		return err
	}

	// Otherwise attempt to coerce into string
	stringifiedErr, ok := recovered.(string)
	if ok {
		return fmt.Errorf(stringifiedErr)
	}

	// Otherwise, just ditch with a generic error.
	return fmt.Errorf("recovered from a panic that was neither a string nor an error")
}
