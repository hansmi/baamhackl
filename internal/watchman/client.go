package watchman

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os/exec"
	"path/filepath"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapio"
)

var ErrClientError = errors.New("watchman client error")

// Client is an abstract interface to a Watchman daemon.
type Client interface {
	Ping(ctx context.Context) error
	WatchSet(ctx context.Context, root string) error
	Recrawl(ctx context.Context, root string) error
	TriggerSet(ctx context.Context, root string, args any) error
	TriggerDel(ctx context.Context, root, name string) error
	ShutdownServer(ctx context.Context) error
}

// defaultClientCommand returns the recommended command for accessing
// a Watchman daemon.
func defaultClientCommand(base []string) []string {
	return append(append([]string(nil), base...),
		"--logfile=/dev/stderr",
		"--log-level=1",
		"--no-local",
		"--no-spawn",
		"--no-pretty",
		"--no-save-state",
	)
}

func runClient(ctx context.Context, args []string, input any) (err error) {
	logger := zap.L().Named(fmt.Sprintf("watchman %x", rand.Int31()))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger.Debug("Running Watchman client",
		zap.Strings("command", args),
		zap.Any("input", input),
	)

	commandLogger := &zapio.Writer{
		Log:   logger,
		Level: zap.DebugLevel,
	}
	defer multierr.AppendInvoke(&err, multierr.Close(commandLogger))

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if input == nil {
		cmd.Stdin = nil
	} else {
		buf, err := json.Marshal(input)
		if err != nil {
			return err
		}

		cmd.Stdin = bytes.NewReader(buf)
	}
	cmd.Stderr = commandLogger

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var response map[string]any

	decodeErr := json.NewDecoder(stdout).Decode(&response)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("client failed: %w", err)
	}

	if decodeErr != nil {
		return fmt.Errorf("decoding %q output failed: %w", cmd.Args, decodeErr)
	}

	logger.Debug("Watchman client output", zap.Any("response", response))

	if errmsg, ok := response["error"]; errmsg != nil && ok {
		return fmt.Errorf("%w: %v", ErrClientError, errmsg)
	}

	return nil
}

// CommandClient implements access to a Watchman daemon through the "watchman"
// command line program.
type CommandClient struct {
	Args []string
}

var _ Client = (*CommandClient)(nil)

func NewCommandClient(base []string) *CommandClient {
	return &CommandClient{
		Args: defaultClientCommand(base),
	}
}

func (c *CommandClient) run(ctx context.Context, args []string, input any) error {
	return runClient(ctx, append(append([]string(nil), c.Args...), args...), input)
}

func (c *CommandClient) Ping(ctx context.Context) error {
	if err := c.run(ctx, []string{"version"}, nil); err != nil {
		return fmt.Errorf("getting watchman daemon version failed: %w", err)
	}

	return nil
}

func (c *CommandClient) WatchSet(ctx context.Context, root string) error {
	return c.run(ctx, []string{"watch", filepath.Clean(root)}, nil)
}

func (c *CommandClient) Recrawl(ctx context.Context, root string) error {
	return c.run(ctx, []string{"debug-recrawl", filepath.Clean(root)}, nil)
}

func (c *CommandClient) TriggerSet(ctx context.Context, root string, descriptor any) error {
	if err := c.run(ctx, []string{"trigger", "--json-command"}, []any{
		"trigger",
		filepath.Clean(root),
		descriptor,
	}); err != nil {
		return fmt.Errorf("setting trigger on %q failed: %w", root, err)
	}

	return nil
}

func (c *CommandClient) TriggerDel(ctx context.Context, root, name string) error {
	if err := c.run(ctx, []string{"trigger-del", filepath.Clean(root), name}, nil); err != nil {
		return fmt.Errorf("deleting trigger %q on %q failed: %w", name, root, err)
	}

	return nil
}

func (c *CommandClient) ShutdownServer(ctx context.Context) error {
	return c.run(ctx, []string{"shutdown-server"}, nil)
}
