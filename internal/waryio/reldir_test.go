package waryio

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/testutil"
)

func TestEnsureRelDir(t *testing.T) {
	type testCase struct {
		name        string
		base        string
		path        string
		want        string
		wantErr     error
		wantMissing bool
	}

	tests := []testCase{
		(func() testCase {
			tmpdir := t.TempDir()

			return testCase{
				name: "parent exists, same dir, relative",
				base: tmpdir,
				path: ".",
				want: tmpdir,
			}
		})(),
		(func() testCase {
			tmpdir := t.TempDir()

			return testCase{
				name: "parent exists, same dir, absolute",
				base: tmpdir,
				path: tmpdir,
				want: tmpdir,
			}
		})(),
		(func() testCase {
			tmpdir := t.TempDir()

			return testCase{
				name: "absolute path to existing directory outside base",
				base: t.TempDir(),
				path: tmpdir,
				want: tmpdir,
			}
		})(),
		(func() testCase {
			tmpdir := filepath.Join(t.TempDir(), "elsewhere")

			return testCase{
				name:        "absolute path to non-existent directory outside base",
				base:        t.TempDir(),
				path:        tmpdir,
				want:        tmpdir,
				wantMissing: true,
			}
		})(),
		(func() testCase {
			tmpdir := t.TempDir()

			return testCase{
				name:        "path is relative to outside base",
				base:        testutil.MustMkdir(t, filepath.Join(tmpdir, "base")),
				path:        "../foo",
				want:        filepath.Join(tmpdir, "foo"),
				wantMissing: true,
			}
		})(),
		{
			name:    "base missing",
			base:    filepath.Join(t.TempDir(), "base", "missing"),
			path:    "foo/bar",
			wantErr: os.ErrNotExist,
		},
		(func() testCase {
			tmpdir := t.TempDir()

			return testCase{
				name:    "not a directory",
				base:    tmpdir,
				path:    filepath.Join(testutil.MustWriteFile(t, filepath.Join(tmpdir, "file"), ""), "dir"),
				wantErr: syscall.ENOTDIR,
			}
		})(),
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EnsureRelDir(tc.base, tc.path, os.ModePerm)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("EnsureRelDir(%q, %q) error diff (-want +got):\n%s", tc.base, tc.path, diff)
			}

			if diff := cmp.Diff(filepath.Clean(tc.want), filepath.Clean(got)); diff != "" {
				t.Errorf("EnsureRelDir(%q, %q) result diff (-want +got):\n%s", tc.base, tc.path, diff)
			}

			if tc.wantMissing {
				testutil.MustNotExist(t, tc.want)
			} else if err == nil {
				if st := testutil.MustLstat(t, got); !st.IsDir() {
					t.Errorf("Not a directory: %s", got)
				}
			}
		})
	}
}
