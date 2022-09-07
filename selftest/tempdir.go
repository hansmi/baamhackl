package selftest

import (
	"os"

	"go.uber.org/multierr"
)

func withTempDir(keep bool, fn func(string) error) error {
	dir, err := os.MkdirTemp("", "selftest*")
	if err != nil {
		return err
	}

	if !keep {
		defer func() {
			multierr.AppendInto(&err, os.RemoveAll(dir))
		}()
	}

	return fn(dir)
}
