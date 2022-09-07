package waryio

import (
	"os"

	"go.uber.org/multierr"
)

// SameStat checks whether two paths point to the same file.
func SameStat(a, b string) (bool, error) {
	fiA, err := os.Lstat(a)
	fiB, errB := os.Lstat(b)

	if err := multierr.Combine(err, errB); err != nil {
		return false, err
	}

	return os.SameFile(fiA, fiB), nil
}
