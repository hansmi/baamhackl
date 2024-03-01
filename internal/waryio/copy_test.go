package waryio

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type fileWrapper struct {
	*os.File

	readHandler func(*os.File, []byte) (int, error)
}

func (r *fileWrapper) Read(p []byte) (int, error) {
	if r.readHandler != nil {
		return r.readHandler(r.File, p)
	}

	return r.File.Read(p)
}

func TestCopyInner(t *testing.T) {
	for _, tc := range []struct {
		name        string
		readHandler func(*os.File, []byte) (int, error)
		wantErr     *regexp.Regexp
		wantPerm    os.FileMode
	}{
		{
			name:     "normal",
			wantPerm: 0o640,
		},
		{
			name: "unepxected eof",
			readHandler: func(*os.File, []byte) (int, error) {
				return 0, io.ErrUnexpectedEOF
			},
			wantErr:  regexp.MustCompile(`unexpected EOF`),
			wantPerm: 0o755,
		},
		{
			name: "modify source",
			readHandler: func(f *os.File, p []byte) (int, error) {
				f2, err := os.OpenFile(f.Name(), os.O_WRONLY, 0644)
				if err != nil {
					return 0, err
				}

				defer f2.Close()

				if err := f2.Truncate(999); err != nil {
					return 0, err
				}

				return f.Read(p)
			},
			wantErr:  regexp.MustCompile(`^source was modified:.*\bsize changed\b.*`),
			wantPerm: 0o755,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			srcPath := filepath.Join(tmpdir, "src")

			content := strings.Repeat("Test content\n", 32*1024)

			if err := os.WriteFile(srcPath, []byte(content), 0644); err != nil {
				t.Errorf("WriteFile() failed: %v", err)
			}

			if err := os.Chmod(srcPath, tc.wantPerm); err != nil {
				t.Errorf("Chmod() failed: %v", err)
			}

			src, err := os.Open(srcPath)
			if err != nil {
				t.Fatalf("Open() failed: %v", err)
			}

			defer src.Close()

			var dst strings.Builder

			if _, err := copyInner(&fileWrapper{
				File:        src,
				readHandler: tc.readHandler,
			}, &dst); err == nil {
				if tc.wantErr != nil {
					t.Errorf("copyInner() failed with %q, Want match for %q", err, tc.wantErr)
				}

				st, err := os.Stat(srcPath)
				if err != nil {
					t.Errorf("Stat() failed: %v", err)
				}

				if got := st.Mode() & os.ModePerm; got != tc.wantPerm {
					t.Errorf("Got source permission %04o, want %04o", got, tc.wantPerm)
				}

				if diff := cmp.Diff(content, dst.String()); diff != "" {
					t.Errorf("Content diff (-want +got):\n%s", diff)
				}
			} else if tc.wantErr == nil || !tc.wantErr.MatchString(err.Error()) {
				t.Errorf("Want error matching %q, got %v", tc.wantErr, err)
			}
		})
	}
}
