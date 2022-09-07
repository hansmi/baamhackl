package sendfilechanges

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/rpc"
	"os"
	"path/filepath"

	"github.com/google/subcommands"
	"github.com/hansmi/baamhackl/internal/cmdutil"
	"github.com/hansmi/baamhackl/internal/service"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type serviceCaller interface {
	Call(string, any, any) error
}

// Command implements the "send-file-changes" subcommand.
type Command struct {
	input       io.Reader
	service     serviceCaller
	handlerName string
	root        string
}

func (*Command) Name() string {
	return "send-file-changes"
}

func (*Command) Synopsis() string {
	return `Send information about changed files to running server.`
}

func (c *Command) Usage() string {
	return cmdutil.Usage(c, "<socket>", `JSON-formatted file change descriptions are read from standard input before being sent to a running server via the given Unix socket.`)
}

func (c *Command) SetFlags(fs *flag.FlagSet) {
	const envVarTrigger = "WATCHMAN_TRIGGER"
	const envVarRoot = "WATCHMAN_ROOT"

	defaultHandlerName := os.Getenv(envVarTrigger)

	defaultRoot := os.Getenv(envVarRoot)
	if defaultRoot == "" {
		defaultRoot = "."
	}

	fs.StringVar(&c.handlerName, "name", defaultHandlerName,
		"Handler name. Defaults to "+envVarTrigger+" from environment.")
	fs.StringVar(&c.root, "root", defaultRoot,
		"Path to the watched root directory. Change events are relative to this path. Defaults to "+envVarRoot+" from environment if set.")
}

func (c *Command) execute(socketPath string) (err error) {
	logger := zap.L()

	req := service.FileChangedRequest{
		HandlerName: c.handlerName,
	}

	if req.RootDir, err = filepath.Abs(c.root); err != nil {
		return err
	}

	if req.HandlerName == "" {
		return errors.New("missing handler name")
	}

	input := c.input

	if input == nil {
		input = os.Stdin
	}

	changes, err := readInput(input)
	if err != nil {
		return fmt.Errorf("reading input failed: %w", err)
	}

	if len(changes) == 0 {
		return nil
	}

	svc := c.service

	if svc == nil {
		client, err := rpc.Dial("unix", socketPath)
		if err != nil {
			return err
		}

		defer multierr.AppendInvoke(&err, multierr.Close(client))

		svc = client
	}

	// Events are sent one at a time. Watchman kills the trigger command if
	// another change event arrives while the command is still running.
	for _, req.Change = range changes {
		logger.Info("Sending request", zap.Reflect("request", req))

		if err := svc.Call("Service.FileChanged", req, &service.FileChangedResponse{}); err != nil {
			return err
		}
	}

	return nil
}

func (c *Command) Execute(ctx context.Context, fs *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if fs.NArg() != 1 {
		fs.Usage()
		return subcommands.ExitUsageError
	}

	return cmdutil.ExecuteStatus(c.execute(fs.Arg(0)))
}
