package testutil

import (
	"errors"
	"os"
	"testing"
)

func MustNotExist(t *testing.T, path string) {
	t.Helper()

	st, err := os.Lstat(path)

	if !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			t.Errorf("Path %q should not exist: %+v", path, st)
		} else {
			t.Errorf("Lstat(%q) failed: %v", path, err)
		}
	}
}
