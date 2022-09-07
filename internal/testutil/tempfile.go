package testutil

import (
	"testing"

	"github.com/spf13/afero"
)

func MustTempFile(t *testing.T, fs afero.Fs) afero.File {
	file, err := afero.TempFile(fs, "", "")
	if err != nil {
		t.Errorf("TempFile() failed: %v", err)
	}

	return file
}
