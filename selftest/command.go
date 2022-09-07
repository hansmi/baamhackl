package selftest

import (
	"context"
	"flag"
	"time"

	"github.com/google/subcommands"
	"github.com/hansmi/baamhackl/internal/cmdutil"
	"github.com/hansmi/baamhackl/internal/watchman"
	"go.uber.org/zap"
)

// Command implements the "selftest" subcommand.
type Command struct {
	wmFlags     watchman.Flags
	timeout     time.Duration
	keepWorkDir bool
}

func (*Command) Name() string {
	return "selftest"
}

func (*Command) Synopsis() string {
	return "Execute tests to verify the system configuration."
}

func (c *Command) Usage() string {
	return cmdutil.Usage(c, "", `
Start a temporary Watchman server instance before executing a number of tests.
Verifies that file change notifications and command execution are working.
`)
}

func (c *Command) SetFlags(fs *flag.FlagSet) {
	c.wmFlags.SetFlags(fs)
	fs.DurationVar(&c.timeout, "timeout", time.Minute, "Maximum duration for running all tests.")
	fs.BoolVar(&c.keepWorkDir, "keep", false, "Leave the temporary directory behind to aid in debugging.")
}

func (c *Command) execute(ctx context.Context) error {
	logger := zap.L()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	err := withTempDir(c.keepWorkDir, func(tmpdir string) error {
		r, err := newRunner(tmpdir)
		if err != nil {
			return err
		}

		return withServer(ctx, logger.Named("watchman server"), c.wmFlags, tmpdir, r.run)
	})

	if err == nil {
		logger.Info("Self-test successful.")
	}

	return err
}

func (c *Command) Execute(ctx context.Context, fs *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if fs.NArg() > 0 {
		fs.Usage()
		return subcommands.ExitUsageError
	}

	return cmdutil.ExecuteStatus(c.execute(ctx))
}
