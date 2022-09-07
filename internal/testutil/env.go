package testutil

import (
	"os"
	"testing"
)

func MustUnsetenv(t *testing.T, key string) {
	t.Helper()

	var restore func() error

	if value, exists := os.LookupEnv(key); exists {
		restore = func() error { return os.Setenv(key, value) }
	} else {
		restore = func() error { return os.Unsetenv(key) }
	}

	t.Cleanup(func() {
		if err := restore(); err != nil {
			t.Errorf("Restoring environment variable %s changes failed: %v", key, err)
		}
	})

	if err := os.Unsetenv(key); err != nil {
		t.Errorf("Unsetting environment variable %s failed: %v", key, err)
	}
}

func MustSetenv(t *testing.T, key, value string) {
	t.Helper()

	if err := os.Setenv(key, value); err != nil {
		t.Errorf("Setting environment variable %s failed: %v", key, err)
	}
}
