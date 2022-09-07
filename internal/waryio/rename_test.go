package waryio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/testutil"
	"github.com/hansmi/baamhackl/internal/uniquename"
)

func TestRenameToAvailableName(t *testing.T) {
	timePrefixDisabledOpts := uniquename.DefaultOptions
	timePrefixDisabledOpts.TimePrefixEnabled = false

	type testCase struct {
		name     string
		oldpath  string
		newpaths StringIter
		want     string
		wantErr  error
	}

	for _, tc := range []testCase{
		{
			name:     "empty",
			oldpath:  testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "old"), ""),
			newpaths: &iterSlice{},
			wantErr:  ErrIterExhausted,
		},
		(func() testCase {
			tmpdir := t.TempDir()

			g, err := uniquename.New(filepath.Join(tmpdir, "new"), timePrefixDisabledOpts)
			if err != nil {
				t.Fatal(err)
			}

			return testCase{
				name:     "success",
				oldpath:  testutil.MustWriteFile(t, filepath.Join(tmpdir, "old"), ""),
				newpaths: g,
				want:     filepath.Join(tmpdir, "new"),
			}
		})(),
		(func() testCase {
			tmpdir := t.TempDir()
			return testCase{
				name:    "dest exists",
				oldpath: testutil.MustWriteFile(t, filepath.Join(tmpdir, "old.txt"), ""),
				newpaths: &iterSlice{
					testutil.MustWriteFile(t, filepath.Join(tmpdir, "first.txt"), ""),
					testutil.MustWriteFile(t, filepath.Join(tmpdir, "second.txt"), ""),
					filepath.Join(tmpdir, "third.txt"),
				},
				want: filepath.Join(tmpdir, "third.txt"),
			}
		})(),
		{
			name:    "dest directory does not exist",
			oldpath: testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "old"), ""),
			newpaths: &iterSlice{
				filepath.Join(t.TempDir(), "missing", "dir", "file.txt"),
			},
			wantErr: os.ErrNotExist,
		},
		(func() testCase {
			tmpdir := t.TempDir()
			path := filepath.Join(tmpdir, "file.txt")
			return testCase{
				name:     "rename to same",
				oldpath:  testutil.MustWriteFile(t, path, ""),
				newpaths: &iterSlice{path, path, filepath.Join(tmpdir, "target"), path},
				want:     filepath.Join(tmpdir, "target"),
			}
		})(),
		(func() testCase {
			tmpdir := t.TempDir()
			path := filepath.Join(tmpdir, "file.txt")
			return testCase{
				name:     "no name available",
				oldpath:  testutil.MustWriteFile(t, path, ""),
				newpaths: &iterSlice{path, path, path},
				wantErr:  os.ErrExist,
			}
		})(),
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenameToAvailableName(tc.oldpath, tc.newpaths)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if err == nil {
				testutil.MustNotExist(t, tc.oldpath)
				testutil.MustLstat(t, got)

				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("Generated path diff (-want +got):\n%s", diff)
				}
			} else {
				testutil.MustLstat(t, tc.oldpath)
			}
		})
	}
}
