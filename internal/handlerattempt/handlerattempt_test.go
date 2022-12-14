package handlerattempt

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/handlercommand"
	"github.com/hansmi/baamhackl/internal/journal"
	"github.com/hansmi/baamhackl/internal/testutil"
	"github.com/hansmi/baamhackl/internal/waryio"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var errCommand = errors.New("command error")

func TestValidateChangedFile(t *testing.T) {
	for _, tc := range []struct {
		name     string
		path     string
		wantErr  error
		wantSize int64
	}{
		{
			name:    "dir",
			path:    t.TempDir(),
			wantErr: os.ErrInvalid,
		},
		{
			name:     "regular file",
			path:     testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "test.txt"), "test"),
			wantSize: 4,
		},
		{
			name:    "missing",
			path:    filepath.Join(t.TempDir(), "test.txt"),
			wantErr: os.ErrNotExist,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateChangedFile(tc.path)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if err == nil {
				if diff := cmp.Diff(tc.wantSize, got.Size()); diff != "" {
					t.Errorf("File size diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestAttempt(t *testing.T) {
	type test struct {
		name               string
		opts               Options
		run                func(context.Context) error
		wantErr            error
		wantPermanent      bool
		changedFileRemains bool
	}

	tests := []test{
		{
			name: "success on non-final attempt",
			opts: Options{
				Config: &config.Handler{},
			},
		},
		{
			name: "success on final attempt",
			opts: Options{
				Config: &config.Handler{},
				Final:  true,
			},
		},
		{
			name: "missing source file",
			opts: Options{
				Config:      &config.Handler{},
				ChangedFile: filepath.Join(t.TempDir(), "missing", "file"),
			},
			wantErr:       os.ErrNotExist,
			wantPermanent: true,
		},
		{
			name: "command error",
			opts: Options{
				Config: func() *config.Handler {
					o := config.HandlerDefaults
					o.RetryCount = 0
					return &o
				}(),
			},
			run: func(ctx context.Context) error {
				return errCommand
			},
			wantErr: errCommand,
		},
		{
			name: "command error with retries",
			opts: Options{
				Config: func() *config.Handler {
					o := config.HandlerDefaults
					o.RetryCount = 2
					return &o
				}(),
			},
			run: func(ctx context.Context) error {
				return errCommand
			},
			wantErr: errCommand,
		},
	}

	sourceForRemove := testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "src"), "remove1")
	tests = append(tests, test{
		name: "command removes input",
		opts: Options{
			Config: func() *config.Handler {
				o := config.HandlerDefaults
				o.RetryCount = 2
				return &o
			}(),
			ChangedFile: sourceForRemove,
		},
		run: func(ctx context.Context) error {
			testutil.MustRemove(t, sourceForRemove)
			return nil
		},
		wantPermanent: true,
	})

	sourceForRemoveAfterFailure := testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "src"), "remove2")
	tests = append(tests, test{
		name: "command removes input and fails",
		opts: Options{
			Config: func() *config.Handler {
				o := config.HandlerDefaults
				o.RetryCount = 2
				return &o
			}(),
			ChangedFile: sourceForRemoveAfterFailure,
		},
		run: func(ctx context.Context) error {
			testutil.MustRemove(t, sourceForRemoveAfterFailure)
			return errCommand
		},
		wantErr:            errCommand,
		wantPermanent:      true,
		changedFileRemains: true,
	})

	sourceForModification := testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "src"), "modify1")
	tests = append(tests, test{
		name: "command modifies input",
		opts: Options{
			Config: func() *config.Handler {
				o := config.HandlerDefaults
				o.RetryCount = 2
				return &o
			}(),
			ChangedFile: sourceForModification,
		},
		run: func(ctx context.Context) error {
			testutil.MustWriteFile(t, sourceForModification, "modified")
			return nil
		},
		wantErr:            waryio.ErrFileChanged,
		changedFileRemains: true,
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			if tc.opts.Config.Path == "" {
				tc.opts.Config.Path = t.TempDir()
			}
			if len(tc.opts.Config.Command) == 0 {
				tc.opts.Config.Command = []string{"placeholder"}
			}
			if tc.opts.Journal == nil {
				tc.opts.Journal = journal.New(tc.opts.Config)
			}
			if tc.opts.ChangedFile == "" {
				tc.opts.ChangedFile = testutil.MustWriteFile(t, filepath.Join(tc.opts.Config.Path, "test.txt"), "content")
			}
			if tc.opts.BaseDir == "" {
				tc.opts.BaseDir = t.TempDir()
			}
			if tc.opts.Logger == nil {
				tc.opts.Logger = zaptest.NewLogger(t)
			}

			h, err := New(tc.opts)
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}

			if tc.run == nil {
				h.run = func(ctx context.Context) error {
					return nil
				}
			} else {
				h.run = tc.run
			}

			changedFileExistedBeforeRun := false

			if _, err := os.Lstat(tc.opts.ChangedFile); err == nil {
				changedFileExistedBeforeRun = true
			} else if !os.IsNotExist(err) {
				t.Errorf("Lstat(%q) failed: %v", tc.opts.ChangedFile, err)
			}

			permanent, err := h.Run(ctx)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantPermanent, permanent); diff != "" {
				t.Errorf("Permanent error diff (-want +got):\n%s", diff)
			}

			if changedFileExistedBeforeRun {
				statAfter, err := os.Lstat(tc.opts.ChangedFile)

				if !tc.changedFileRemains {
					if permanent && err == nil {
						t.Errorf("Input file still exists after permanent failure: %+v", statAfter)
					}

					if os.IsNotExist(err) {
						err = nil
					}
				} else if os.IsNotExist(err) {
					err = nil
				}

				if err != nil {
					t.Errorf("Lstat(%q) failed: %v", tc.opts.ChangedFile, err)
				}
			}
		})
	}
}

func TestNewNoCommand(t *testing.T) {
	_, err := New(Options{
		Config:      &config.HandlerDefaults,
		ChangedFile: t.TempDir(),
		BaseDir:     t.TempDir(),
		Logger:      zap.NewNop(),
	})

	if diff := cmp.Diff(handlercommand.ErrMissing, err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("Error diff (-want +got):\n%s", diff)
	}
}
