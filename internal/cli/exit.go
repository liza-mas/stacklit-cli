package cli

import (
	"errors"
	"fmt"
)

const (
	// ExitStale reports that a check completed successfully and found stale output.
	ExitStale = 1
	// ExitFailure reports an operational failure in a command with differentiated exits.
	ExitFailure = 2
)

type exitError struct {
	code int
	err  error
}

func newExitError(code int, err error) error {
	return &exitError{code: code, err: err}
}

func (e *exitError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return fmt.Sprintf("exit status %d", e.code)
}

func (e *exitError) Unwrap() error {
	return e.err
}

// ErrorToExit converts command errors into process exit codes and printable errors.
func ErrorToExit(err error) (int, error) {
	if err == nil {
		return 0, nil
	}

	var exitErr *exitError
	if errors.As(err, &exitErr) {
		return exitErr.code, exitErr.err
	}

	return 1, err
}
