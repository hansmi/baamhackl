package watch

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/subcommands"
	"github.com/hansmi/baamhackl/internal/cleanupgroup"
	"github.com/hansmi/baamhackl/internal/cmdutil"
	"github.com/hansmi/baamhackl/internal/config"
	"github.com/hansmi/baamhackl/internal/service"
	"github.com/hansmi/baamhackl/internal/signalwait"
	"github.com/hansmi/baamhackl/internal/watchman"
	"github.com/hansmi/baamhackl/internal/watchmantrigger"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

func logConfig(cfg *config.Root, logger *zap.Logger) error {
	var buf bytes.Buffer

	if err := cfg.Marshal(&buf); err != nil {
		return err
	}

	logger.Debug("Configuration:\n" + buf.String())

	return nil
}

func createTempDir(parent string) (string, func() error, error) {
	tmpdir, err := os.MkdirTemp(parent, "")
	if err != nil {
		return "", nil, err
	}

	return tmpdir, func() error {
		if err := os.RemoveAll(tmpdir); !(err == nil || os.IsNotExist(err)) {
			return fmt.Errorf("removing temporary directory failed: %w", err)
		}

		return nil
	}, nil
}

// Command implements the "watch" subcommand.
type Command struct {
	wmFlags          watchman.Flags
	runtimeParentDir string
	slotCount        uint
	pruneInterval    time.Duration
	shutdownTimeout  time.Duration
	configFlag       config.Flag
}

func (*Command) Name() string {
	return "watch"
}

func (*Command) Synopsis() string {
	return "Observe one or multiple directories and act on file changes."
}

func (c *Command) Usage() string {
	return cmdutil.Usage(c, "", "")
}

func (c *Command) SetFlags(fs *flag.FlagSet) {
	c.wmFlags.SetFlags(fs)

	fs.StringVar(&c.runtimeParentDir, "state_dir", os.TempDir(),
		"Parent directory for runtime state directory.")
	fs.UintVar(&c.slotCount, "slots", 1,
		"Maximum number of handler commands to run simultaneously.")
	fs.DurationVar(&c.shutdownTimeout, "shutdown_timeout", time.Minute,
		"Amount of time to wait for running handlers.")
	fs.DurationVar(&c.pruneInterval, "prune_interval", time.Hour,
		"How often to delete old journal entries.")
	c.configFlag.SetFlags(fs)
}

func (c *Command) ExecuteWithClient(ctx context.Context, client watchman.Client) (err error) {
	logger := zap.L()

	cfg, err := c.configFlag.Load()
	if err != nil {
		return err
	}

	if err := logConfig(cfg, logger); err != nil {
		return err
	}

	if err := watchman.WaitForReady(ctx, client); err != nil {
		return err
	}

	waitForSignal, stopSignalWait := signalwait.Setup(os.Interrupt, syscall.SIGTERM)
	defer stopSignalWait()

	tmpdir, removeTempDir, err := createTempDir(c.runtimeParentDir)
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&err, multierr.Invoke(removeTempDir))

	var cleanup cleanupgroup.CleanupGroup
	defer func() {
		multierr.AppendInto(&err, cleanup.CallWithTimeout(c.shutdownTimeout))
	}()

	r := newRouter(cfg.Handlers)
	r.start(int(c.slotCount))
	cleanup.Append(r.stop)

	socketPath := filepath.Join(tmpdir, "server.socket")

	triggerGroup := watchmantrigger.Group{
		Client:     client,
		SocketPath: socketPath,
	}
	cleanup.Append(triggerGroup.DeleteAll)

	srv, err := service.ListenAndServe(socketPath, r)
	if err != nil {
		return err
	}

	defer multierr.AppendInvoke(&err, multierr.Close(srv))

	logger.Info("Socket is ready", zap.String("path", socketPath))

	if err := triggerGroup.SetAll(ctx, cfg.Handlers); err != nil {
		return err
	}

	if err := triggerGroup.RecrawlAll(ctx); err != nil {
		return err
	}

	r.startPruning(c.pruneInterval)

	if err := waitForSignal(ctx); err != nil {
		logger.Info(fmt.Sprintf("Shutting down gracefully: %v", err))
	}

	return nil
}

func (c *Command) Execute(ctx context.Context, fs *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if fs.NArg() > 0 {
		fs.Usage()
		return subcommands.ExitUsageError
	}

	return cmdutil.ExecuteStatus(c.ExecuteWithClient(ctx, c.wmFlags.NewClient()))
}
