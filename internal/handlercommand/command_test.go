package handlercommand

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/baamhackl/internal/cmdemu"
	"github.com/hansmi/baamhackl/internal/exepath"
	"github.com/hansmi/baamhackl/internal/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var fakeCommand = cmdemu.Command{
	Name: "handler",
	Execute: func(args []string) error {
		fs := flag.NewFlagSet("", flag.PanicOnError)
		_ = fs.Parse(args)

		if fs.NArg() == 1 {
			switch fs.Arg(0) {
			case "success":
				fmt.Println("success")
				return nil

			case "exit-99":
				return cmdemu.ExitCodeError(99)
			}
		}

		return errors.New("incorrect usage")
	},
}

func TestMain(m *testing.M) {
	w := cmdemu.New(flag.CommandLine)
	w.Register(fakeCommand)
	os.Exit(w.Main(m))
}

func TestCreateDirectories(t *testing.T) {
	for _, tc := range []struct {
		paths   []string
		wantErr error
	}{
		{},
		{
			paths: []string{
				filepath.Join(t.TempDir(), "bar"),
				filepath.Join(t.TempDir(), "something"),
			},
		},
		{
			paths: []string{
				filepath.Join(t.TempDir(), "a"),
				t.TempDir(),
				filepath.Join(t.TempDir(), "a", "b"),
				t.TempDir(),
			},
			wantErr: os.ErrNotExist,
		},
	} {
		err := createDirectories(tc.paths)

		if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
			t.Errorf("Error diff (-want +got):\n%s", diff)
		}
	}
}

func TestRunCommand(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	for _, tc := range []struct {
		name     string
		args     []string
		wantErr  error
		wantCode int
	}{
		{
			name: "success",
			args: fakeCommand.MakeArgs("success"),
		},
		{
			name:     "failure",
			args:     fakeCommand.MakeArgs("exit-99"),
			wantCode: 99,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			loggerCore, observed := observer.New(zapcore.DebugLevel)
			logger := zap.New(loggerCore)

			cmd := exec.CommandContext(ctx, tc.args[0], tc.args[1:]...)
			cmd.Stdin = nil
			cmd.Stdout = nil
			cmd.Stderr = nil
			cmd.Dir = t.TempDir()

			err := runCommand(ctx, logger, cmd)

			var exitErr *exec.ExitError

			if errors.As(err, &exitErr) && exitErr.ExitCode() == tc.wantCode {
			} else if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if all := observed.All(); len(all) != 1 {
				t.Errorf("Expected exactly one log message: %+v", all)
			} else {
				fields := all[0].ContextMap()

				if diff := cmp.Diff(int64(tc.wantCode), fields["exit_code"]); diff != "" {
					t.Errorf("Logged exit code diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestNew(t *testing.T) {
	for _, tc := range []struct {
		name       string
		opts       Options
		sourceName string
		wantErr    error
	}{
		{
			name:    "empty",
			wantErr: ErrMissing,
		},
		{
			name: "success",
			opts: Options{
				SourceFile: testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "src"), ""),
				Logger:     zap.NewNop(),
				BaseDir:    t.TempDir(),
				Command:    fakeCommand.MakeArgs("success"),
			},
			sourceName: "src",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := New(tc.opts)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}

			if err == nil {
				wantEnv := []string{
					"BAAMHACKL_PROGRAM=" + exepath.MustGet(),
					"BAAMHACKL_ORIGINAL=" + tc.opts.SourceFile,
					"BAAMHACKL_INPUT=" + filepath.Join(tc.opts.BaseDir, "input", tc.sourceName),
					"BAAMHACKL_WORKDIR=" + filepath.Join(tc.opts.BaseDir, "work"),
				}

				if diff := cmp.Diff(wantEnv, c.environ, cmpopts.SortSlices(func(a, b string) bool {
					return a < b
				})); diff != "" {
					t.Errorf("Env variable diff (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestRun(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ctx     context.Context
		opts    Options
		wantErr error
	}{
		{
			name: "success",
			opts: Options{
				SourceFile: testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "src"), "foobar"),
				BaseDir:    t.TempDir(),
				Command:    fakeCommand.MakeArgs("success"),
			},
		},
		{
			name: "base dir missing",
			opts: Options{
				SourceFile: testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "src"), ""),
				BaseDir:    filepath.Join(t.TempDir(), "not", "found"),
				Command:    []string{""},
			},
			wantErr: os.ErrNotExist,
		},
		{
			name: "source file missing",
			opts: Options{
				SourceFile: filepath.Join(t.TempDir(), "not", "found"),
				BaseDir:    t.TempDir(),
				Command:    []string{""},
			},
			wantErr: os.ErrNotExist,
		},
		{
			name: "log file exists already",
			opts: Options{
				SourceFile: testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "src"), ""),
				BaseDir: func() string {
					tmpdir := t.TempDir()

					testutil.MustWriteFile(t, filepath.Join(tmpdir, "command_output.txt"), "")

					return tmpdir
				}(),
				Command: []string{""},
			},
			wantErr: os.ErrExist,
		},
		{
			name: "command error",
			opts: Options{
				SourceFile: testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "src"), "foobar"),
				BaseDir:    t.TempDir(),
				Command:    fakeCommand.MakeArgs("exit-99"),
			},
			wantErr: cmpopts.AnyError,
		},
		{
			name: "cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			opts: Options{
				SourceFile: testutil.MustWriteFile(t, filepath.Join(t.TempDir(), "src"), "foobar"),
				BaseDir:    t.TempDir(),
				Command:    fakeCommand.MakeArgs("success"),
			},
			wantErr: context.Canceled,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.opts.Logger == nil {
				tc.opts.Logger = zap.NewNop()
			}

			ctx := tc.ctx

			if ctx == nil {
				ctx = context.Background()
			}

			ctx, cancel := context.WithCancel(ctx)
			t.Cleanup(cancel)

			c, err := New(tc.opts)
			if err != nil {
				t.Fatalf("New(%+v) failed: %v", tc.opts, err)
			}

			err = c.Run(ctx)

			if diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Error diff (-want +got):\n%s", diff)
			}
		})
	}
}
