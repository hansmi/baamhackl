package watch

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

func TestHandlerOnce(t *testing.T) {
	type test struct {
		name               string
		h                  handlerOnce
		wantErr            error
		wantPermanent      bool
		changedFileRemains bool
	}

	tests := []test{
		{
			name: "success on non-final attempt",
			h: handlerOnce{
				cfg: &config.Handler{},
			},
		},
		{
			name: "success on final attempt",
			h: handlerOnce{
				final: true,
				cfg:   &config.Handler{},
			},
		},
		{
			name: "missing source file",
			h: handlerOnce{
				cfg:         &config.Handler{},
				changedFile: filepath.Join(t.TempDir(), "missing", "file"),
			},
			wantErr:       os.ErrNotExist,
			wantPermanent: true,
		},
		{
			name: "command error",
			h: handlerOnce{
				cfg: func() *config.Handler {
					o := config.HandlerDefaults
					o.RetryCount = 0
					return &o
				}(),
				run: func(ctx context.Context) error {
					return errCommand
				},
			},
			wantErr: errCommand,
		},
		{
			name: "command error with retries",
			h: handlerOnce{
				cfg: func() *config.Handler {
					o := config.HandlerDefaults
					o.RetryCount = 2
					return &o
				}(),
				run: func(ctx context.Context) error {
					return errCommand
				},
			},
			wantErr: errCommand,
		},
	}

	sourceForRemove := testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "src"), "remove1")
	tests = append(tests, test{
		name: "command removes input",
		h: handlerOnce{
			cfg: func() *config.Handler {
				o := config.HandlerDefaults
				o.RetryCount = 2
				return &o
			}(),
			changedFile: sourceForRemove,
			run: func(ctx context.Context) error {
				testutil.MustRemove(t, sourceForRemove)
				return nil
			},
		},
		wantPermanent: true,
	})

	sourceForRemoveAfterFailure := testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "src"), "remove2")
	tests = append(tests, test{
		name: "command removes input and fails",
		h: handlerOnce{
			cfg: func() *config.Handler {
				o := config.HandlerDefaults
				o.RetryCount = 2
				return &o
			}(),
			changedFile: sourceForRemoveAfterFailure,
			run: func(ctx context.Context) error {
				testutil.MustRemove(t, sourceForRemoveAfterFailure)
				return errCommand
			},
		},
		wantErr:            errCommand,
		wantPermanent:      true,
		changedFileRemains: true,
	})

	sourceForModification := testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "src"), "modify1")
	tests = append(tests, test{
		name: "command modifies input",
		h: handlerOnce{
			cfg: func() *config.Handler {
				o := config.HandlerDefaults
				o.RetryCount = 2
				return &o
			}(),
			changedFile: sourceForModification,
			run: func(ctx context.Context) error {
				testutil.MustWriteFile(t, sourceForModification, "modified")
				return nil
			},
		},
		wantErr:            waryio.ErrFileChanged,
		changedFileRemains: true,
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

			if tc.h.cfg.Path == "" {
				tc.h.cfg.Path = t.TempDir()
			}
			if tc.h.journal == nil {
				tc.h.journal = journal.New(tc.h.cfg)
			}
			if tc.h.changedFile == "" {
				tc.h.changedFile = testutil.MustWriteFile(t, filepath.Join(tc.h.cfg.Path, "test.txt"), "content")
			}
			if tc.h.baseDir == "" {
				tc.h.baseDir = t.TempDir()
			}
			if tc.h.logger == nil {
				tc.h.logger = zaptest.NewLogger(t)
			}
			if tc.h.run == nil {
				tc.h.run = func(ctx context.Context) error {
					return nil
				}
			}

			changedFileExistedBeforeRun := false

			if _, err := os.Lstat(tc.h.changedFile); err == nil {
				changedFileExistedBeforeRun = true
			} else if !os.IsNotExist(err) {
				t.Errorf("Lstat(%q) failed: %v", tc.h.changedFile, err)
			}

			permanent, err := tc.h.do(ctx)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tc.wantPermanent, permanent); diff != "" {
				t.Errorf("Permanent error diff (-want +got):\n%s", diff)
			}

			if changedFileExistedBeforeRun {
				statAfter, err := os.Lstat(tc.h.changedFile)

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
					t.Errorf("Lstat(%q) failed: %v", tc.h.changedFile, err)
				}
			}
		})
	}
}

func TestHandlerOnceNoCommand(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	h := handlerOnce{
		cfg:         &config.HandlerDefaults,
		changedFile: t.TempDir(),
		baseDir:     t.TempDir(),
		logger:      zap.NewNop(),
	}

	permanent, err := h.do(ctx)

	if diff := cmp.Diff(handlercommand.ErrMissing, err, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("Error diff (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(true, permanent); diff != "" {
		t.Errorf("Permanent error diff (-want +got):\n%s", diff)
	}
}
