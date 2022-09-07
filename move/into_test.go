package move

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestExecute(t *testing.T) {
	for _, tc := range []struct {
		name        string
		cmd         IntoCommand
		sourceFiles []string
		wantErr     error
	}{
		{name: "empty"},
		{
			name: "success",
			sourceFiles: []string{
				t.TempDir(),
			},
		},
		{
			name: "success with multiple",
			sourceFiles: []string{
				t.TempDir(),
				t.TempDir(),
				t.TempDir(),
			},
		},
		{
			name: "rename",
			cmd: IntoCommand{
				rename: "another name",
			},
			sourceFiles: []string{t.TempDir()},
		},
		{
			name: "rename with multiple",
			cmd: IntoCommand{
				rename: "another name",
			},
			sourceFiles: []string{
				t.TempDir(),
				t.TempDir(),
				t.TempDir(),
			},
			wantErr: errRenameNotSupported,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cmd.execute(t.TempDir(), tc.sourceFiles)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}
		})
	}
}
