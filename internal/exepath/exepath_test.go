package exepath

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var errTest = errors.New("test error")

func validate(t *testing.T, path string) {
	t.Helper()

	if !filepath.IsAbs(path) {
		t.Errorf("Path is not absolute: %q", path)
	}

	if fi, err := os.Stat(path); err != nil {
		t.Errorf("Stat() failed on executable: %v", err)
	} else if fi.Size() < 1024 {
		t.Errorf("Executable is too small: %#v", fi)
	}
}

func TestResolve(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() failed: %v", err)
	}

	tmpdir := t.TempDir()

	for _, tc := range []struct {
		name    string
		fn      executableFunc
		want    string
		wantErr error
	}{
		{
			name: "success",
			fn: func() (string, error) {
				return filepath.Join(tmpdir, "program"), nil
			},
			want: filepath.Join(tmpdir, "program"),
		},
		{
			name: "relative path",
			fn: func() (string, error) {
				return "path/to/program", nil
			},
			want: filepath.Join(wd, "path", "to", "program"),
		},
		{
			name: "error",
			fn: func() (string, error) {
				return "", errTest
			},
			wantErr: errTest,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := resolve(tc.fn)

			if d.err == nil && !filepath.IsAbs(d.path) {
				t.Errorf("Path is not absolute: %s", d.path)
			}

			if diff := cmp.Diff(tc.want, d.path); diff != "" {
				t.Errorf("Path diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantErr, d.err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGlobalGet(t *testing.T) {
	path, err := Get()
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	validate(t, path)
}

func TestGlobalMustGet(t *testing.T) {
	validate(t, MustGet())
}
