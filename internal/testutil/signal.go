package testutil

import (
	"fmt"
	"os"
)

// Raise sends a signal to the current process.
func Raise(sig os.Signal) error {
	if proc, err := os.FindProcess(os.Getpid()); err != nil {
		return fmt.Errorf("FindProcess() failed: %w", err)
	} else if err := proc.Signal(sig); err != nil {
		return fmt.Errorf("Signal() failed: %w", err)
	}

	return nil
}
