package testutil

import (
	"os"
	"testing"
)

func MustLstat(t *testing.T, path string) os.FileInfo {
	t.Helper()

	st, err := os.Lstat(path)
	if err != nil {
		t.Errorf("Lstat(%q) failed: %v", path, err)
	}

	return st
}
