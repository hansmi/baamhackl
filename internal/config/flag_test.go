package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/testutil"
)

func TestFlag(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		setenv  string
		want    *Root
		wantErr error
	}{
		{
			name:    "empty argument",
			wantErr: ErrMissingFile,
		},
		{
			name:    "nonexistent file",
			args:    []string{"--config", filepath.Join(t.TempDir(), "missing")},
			wantErr: os.ErrNotExist,
		},
		{
			name: "empty config",
			args: []string{"--config", os.DevNull},
			want: &Root{},
		},
		{
			name: "minimal",
			args: []string{"--config", testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "cfg"), `
---
handlers:
- name: from flag
  path: /test/dir
  command: ["path", "to", "command"]
`)},
			want: &Root{
				Handlers: []*Handler{
					(func() *Handler {
						o := HandlerDefaults
						o.Name = "from flag"
						o.Path = "/test/dir"
						o.Command = []string{"path", "to", "command"}
						return &o
					})(),
				},
			},
		},
		{
			name: "env",
			setenv: testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "env"), `
handlers:
- name: from env
  path: /tmp
  command: ["test"]
`),
			want: &Root{
				Handlers: []*Handler{
					(func() *Handler {
						o := HandlerDefaults
						o.Name = "from env"
						o.Path = "/tmp"
						o.Command = []string{"test"}
						return &o
					})(),
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testutil.MustUnsetenv(t, PathEnvVar)

			if tc.setenv != "" {
				testutil.MustSetenv(t, PathEnvVar, tc.setenv)
			}

			var f Flag

			fs := flag.NewFlagSet(tc.name, flag.PanicOnError)
			f.SetFlags(fs)

			if err := fs.Parse(tc.args); err != nil {
				t.Fatal(err)
			}

			got, err := f.Load()

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if err == nil {
				if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("Config diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}
