package selftest

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hansmi/baamhackl/internal/watchman"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapio"
	"golang.org/x/sys/unix"
)

var errDeadlineMissing = errors.New("context must have a deadline")

type serverReadyFunc func(context.Context, watchman.Client) error

// withServer starts a watchman server with a pristine configuration. All
// output is logged.
func withServer(ctx context.Context, logger *zap.Logger, fv watchman.Flags, dir string, ready serverReadyFunc) (err error) {
	commandLogger := &zapio.Writer{
		Log:   logger,
		Level: zap.DebugLevel,
	}
	defer multierr.AppendInvoke(&err, multierr.Close(commandLogger))

	sockname := filepath.Join(dir, "sock")

	var args []string

	if deadline, ok := ctx.Deadline(); !ok {
		return errDeadlineMissing
	} else {
		seconds := time.Until(deadline).Seconds()

		// Wrap the server command with another ensuring that the server will
		// eventually be cleaned up.
		args = append(args,
			"/usr/bin/timeout",
			"--verbose",
			fmt.Sprintf("--kill-after=%.0fs", math.Ceil(seconds*1.2)),
			fmt.Sprintf("%.0fs", math.Ceil(seconds*1.1)),
		)
	}

	args = append(args, fv.Args()...)
	args = append(args,
		"--foreground",
		"--logfile=/dev/stderr",
		"--pidfile", filepath.Join(dir, "pid"),
		"--statefile", filepath.Join(dir, "state"),
		"--sockname", sockname,
	)

	logger.Debug("Starting watchman server", zap.Strings("args", args))

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdout = commandLogger
	cmd.Stderr = commandLogger
	cmd.Env = append([]string(nil), os.Environ()...)
	cmd.Env = append(cmd.Env, "WATCHMAN_CONFIG_FILE="+os.DevNull)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Request a separate process group.
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	clientCtx, clientCancel := context.WithCancel(ctx)
	defer clientCancel()

	go func() {
		defer clientCancel()

		if waitErr := cmd.Wait(); waitErr != nil {
			logger.Error("Watchman server failed", zap.Error(waitErr))
		}
	}()

	defer func() {
		if cmd.ProcessState == nil {
			logger.Debug("Killing watchman server")

			// Terminate whole process group, not just the process itself.
			multierr.AppendInto(&err, unix.Kill(-cmd.Process.Pid, unix.SIGKILL))

			// Wait for process to terminate
			<-clientCtx.Done()
		}
	}()

	client := fv.NewClient()
	client.Args = append(client.Args, "--sockname", sockname)

	if err := watchman.WaitForReady(clientCtx, client); err != nil {
		return err
	}

	err = ready(clientCtx, client)

	select {
	case <-clientCtx.Done():
	default:
		// Best-effort attempt at shutting down cleanly
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		shutdownErr := client.ShutdownServer(shutdownCtx)

		select {
		case <-clientCtx.Done():
		case <-shutdownCtx.Done():
			multierr.AppendInto(&shutdownErr, shutdownCtx.Err())
		}

		if !(shutdownErr == nil || errors.Is(shutdownErr, context.Canceled)) {
			logger.Debug("Server shutdown failed", zap.Error(shutdownErr))
		}
	}

	return err
}
