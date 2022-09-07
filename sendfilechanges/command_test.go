package sendfilechanges

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/service"
	"github.com/hansmi/baamhackl/internal/watchman"
)

type fakeCaller struct {
	calls []service.FileChangedRequest
}

func (c *fakeCaller) Call(method string, args any, reply any) error {
	c.calls = append(c.calls, args.(service.FileChangedRequest))
	return nil
}

func TestExecute(t *testing.T) {
	tmpdir := t.TempDir()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() failed: %v", err)
	}

	reltmpdir, err := filepath.Rel(wd, tmpdir)
	if err != nil {
		t.Fatalf("Rel() failed: %v", err)
	}

	for _, tc := range []struct {
		name    string
		cmd     *Command
		want    []service.FileChangedRequest
		wantErr error
	}{
		{
			name: "empty",
			cmd: &Command{
				input:       strings.NewReader(""),
				handlerName: "missing",
				root:        t.TempDir(),
			},
			wantErr: io.EOF,
		},
		{
			name: "requests",
			cmd: &Command{
				input: strings.NewReader(`[
					{"name": "bbb.txt", "cclock": "abcd"},
					{"name": "sub/dir/aaa.txt", "mtime_us": 1650224366000000}
					]`),
				handlerName: "ocr",
				root:        tmpdir,
			},
			want: []service.FileChangedRequest{
				{
					HandlerName: "ocr",
					RootDir:     filepath.Clean(tmpdir),
					Change: watchman.FileChange{
						Name:   "bbb.txt",
						CClock: "abcd",
					},
				},
				{
					HandlerName: "ocr",
					RootDir:     filepath.Clean(tmpdir),
					Change: watchman.FileChange{
						Name:  "sub/dir/aaa.txt",
						MTime: time.Date(2022, time.April, 17, 19, 39, 26, 0, time.UTC),
					},
				},
			},
		},
		{
			name: "relative root",
			cmd: &Command{
				input:       strings.NewReader(`[{"name": "../file.txt"} ]`),
				handlerName: "relroot",
				root:        reltmpdir,
			},
			want: []service.FileChangedRequest{
				{
					HandlerName: "relroot",
					RootDir:     filepath.Clean(tmpdir),
					Change: watchman.FileChange{
						Name: "../file.txt",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var svc fakeCaller

			tc.cmd.service = &svc

			err := tc.cmd.execute(t.TempDir())

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if err == nil {
				if diff := cmp.Diff(tc.want, svc.calls, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("Request diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}
