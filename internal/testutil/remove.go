package testutil

import (
	"os"
	"testing"
)

func MustRemove(t *testing.T, path string) {
	t.Helper()

	if err := os.Remove(path); err != nil {
		t.Errorf("Remove(%q) failed: %v", path, err)
	}
}
