package teelog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/testutil"
	"github.com/spf13/afero"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"golang.org/x/sys/unix"
)

var errSync = errors.New("sync error")
var errClose = errors.New("close error")

type fakeFs struct {
	afero.Fs
	syncErr  error
	closeErr error
}

func (fs *fakeFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	fh, err := fs.Fs.OpenFile(name, flag, perm)

	return &fakeFsFile{fs: fs, File: fh}, err
}

type fakeFsFile struct {
	fs *fakeFs
	afero.File
}

func (f *fakeFsFile) Sync() error {
	return f.fs.syncErr
}

func (f *fakeFsFile) Close() error {
	err := f.File.Close()

	if f.fs.closeErr != nil {
		err = f.fs.closeErr
	}

	return err
}

func TestFile(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fs      afero.Fs
		wantErr error
	}{
		{
			name: "memfs",
			fs:   afero.NewMemMapFs(),
		},
		{
			name: "sync not implemented",
			fs: &fakeFs{
				Fs:      afero.NewMemMapFs(),
				syncErr: unix.EINVAL,
			},
		},
		{
			name: "sync fails",
			fs: &fakeFs{
				Fs:      afero.NewMemMapFs(),
				syncErr: errSync,
			},
			wantErr: errSync,
		},
		{
			name: "close fails",
			fs: &fakeFs{
				Fs:       afero.NewMemMapFs(),
				closeErr: errClose,
			},
			wantErr: errClose,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent, observed := observer.New(zapcore.DebugLevel)
			f := &File{
				Parent: zap.New(parent),
				Path:   testutil.MustTempFile(t, tc.fs).Name(),
				fs:     tc.fs,
			}

			for i := 0; i < 3; i++ {
				msg := fmt.Sprintf("test %d", i)

				err := f.Wrap(func(inner *zap.Logger) error {
					inner.Info(msg)
					return nil
				})

				if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
					t.Errorf("Wrap() error diff (-want +got):\n%s", diff)
				}

				if diff := cmp.Diff(msg, observed.All()[i].Message); diff != "" {
					t.Errorf("Message in parent (-want +got):\n%s", diff)
				}

				if err == nil {
					if content, err := afero.ReadFile(tc.fs, f.Path); err != nil {
						t.Errorf("ReadFile() failed: %v", err)
					} else {
						var msg struct {
							Time    string `json:"ts"`
							Level   string `json:"level"`
							Message string `json:"msg"`
						}

						if err := json.NewDecoder(bytes.NewReader(content)).Decode(&msg); err != nil {
							t.Errorf("Unmarshal() failed: %v", err)
						} else if diff := cmp.Diff("test 0", msg.Message); diff != "" {
							t.Errorf("Log message diff (-want +got):\n%s", diff)
						}
					}
				}
			}
		})
	}
}
