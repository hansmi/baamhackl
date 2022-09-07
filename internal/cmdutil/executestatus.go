package cmdutil

import (
	"fmt"
	"io"
	"os"

	"github.com/google/subcommands"
)

func executeStatus(w io.Writer, err error) subcommands.ExitStatus {
	if err != nil {
		fmt.Fprintf(w, "Error: %v\n", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

// ExecuteStatus converts a Go error value to a command exit status. Non-nil
// error values are printed to standard error.
func ExecuteStatus(err error) subcommands.ExitStatus {
	return executeStatus(os.Stderr, err)
}
