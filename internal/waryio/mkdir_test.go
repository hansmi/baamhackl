package waryio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/testutil"
)

func TestMakeAvailableDir(t *testing.T) {
	tmpdir := t.TempDir()

	for _, tc := range []struct {
		name    string
		paths   StringIter
		want    string
		wantErr error
	}{
		{
			name:    "empty",
			paths:   &iterSlice{},
			wantErr: ErrIterExhausted,
		},
		{
			name: "success",
			paths: &iterSlice{
				filepath.Join(tmpdir, "dir1"),
			},
			want: filepath.Join(tmpdir, "dir1"),
		},
		{
			name: "multiple tries",
			paths: &iterSlice{
				t.TempDir(),
				t.TempDir(),
				t.TempDir(),
				filepath.Join(tmpdir, "dir2"),
				t.TempDir(),
			},
			want: filepath.Join(tmpdir, "dir2"),
		},
		{
			name: "parent does not exist",
			paths: &iterSlice{
				filepath.Join(t.TempDir(), "missing", "dir", "test"),
			},
			wantErr: os.ErrNotExist,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := MakeAvailableDir(tc.paths)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if err == nil {
				if st := testutil.MustLstat(t, got); st == nil || !st.IsDir() {
					t.Errorf("Created path is not a directory: %#v", st)
				}

				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("Generated path diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}
