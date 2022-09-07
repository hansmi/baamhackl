package testutil

import (
	"os"
	"testing"
)

func MustMkdir(t *testing.T, path string) string {
	t.Helper()

	if err := os.Mkdir(path, os.ModePerm); err != nil {
		t.Errorf("Mkdir(%q) failed: %v", path, err)
	}

	return path
}
