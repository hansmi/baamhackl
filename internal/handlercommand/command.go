package handlercommand

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hansmi/baamhackl/internal/exepath"
	"github.com/hansmi/baamhackl/internal/waryio"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var ErrMissing = errors.New("missing command")

func createDirectories(paths []string) error {
	var result error

	for _, dir := range paths {
		if err := os.Mkdir(dir, os.ModePerm); !(err == nil || os.IsExist(err)) {
			multierr.AppendInto(&result, err)
		}
	}

	return result
}

func copyInputFile(source, dest string) error {
	var opts waryio.CopyOptions = waryio.DefaultCopyOptions

	opts.SourcePath = source
	opts.SourceFlags |= syscall.O_NOFOLLOW

	opts.DestPath = dest
	opts.DestFlags |= os.O_EXCL | syscall.O_NOFOLLOW

	return waryio.Copy(opts)
}

func runCommand(ctx context.Context, logger *zap.Logger, m MetricsReporter, cmd *exec.Cmd) error {
	start := time.Now()

	err := cmd.Run()

	wallTime := time.Since(start)

	logValues := []zapcore.Field{
		zap.Error(err),
		zap.Duration("wall_time", wallTime),
	}

	// ProcessState may be nil if starting the command failed, e.g. due to the
	// executable not being found.
	if ps := cmd.ProcessState; ps != nil {
		logValues = append(logValues,
			zap.Duration("system_time", ps.SystemTime()),
			zap.Duration("user_time", ps.UserTime()),
			zap.Int("exit_code", ps.ExitCode()),
		)

		if m != nil {
			m.ReportProcessState(ps.ExitCode(), wallTime, ps.UserTime(), ps.SystemTime())
		}
	}

	logger.Info("Command exited", logValues...)

	if err != nil {
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %s", ctx.Err(), err)
		default:
		}

		err = fmt.Errorf("command failed: %w", err)
	}

	return err
}

type MetricsReporter interface {
	ReportProcessState(exitCode int, wallTime, userTime, systemTime time.Duration)
}

type Options struct {
	Logger *zap.Logger

	// Path to the changed file.
	SourceFile string

	// Directory for storing execution-related files.
	BaseDir string

	// Command arguments.
	Command []string

	// Interface for reporting command-specific metrics.
	Metrics MetricsReporter
}

type Command struct {
	opts Options

	inputDir   string
	inputFile  string
	workDir    string
	outputFile string

	environ []string
}

// New instantiates a command for a changed file.
func New(opts Options) (*Command, error) {
	exe, err := exepath.Get()
	if err != nil {
		return nil, err
	}

	if len(opts.Command) < 1 {
		return nil, ErrMissing
	}

	c := &Command{
		opts:       opts,
		inputDir:   filepath.Join(opts.BaseDir, "input"),
		workDir:    filepath.Join(opts.BaseDir, "work"),
		outputFile: filepath.Join(opts.BaseDir, "command_output.txt"),
	}

	c.inputFile = filepath.Join(c.inputDir, filepath.Base(c.opts.SourceFile))
	c.environ = []string{
		"BAAMHACKL_PROGRAM=" + exe,
		"BAAMHACKL_ORIGINAL=" + c.opts.SourceFile,
		"BAAMHACKL_WORKDIR=" + c.workDir,
		"BAAMHACKL_INPUT=" + c.inputFile,
	}

	return c, nil
}

func (c *Command) prepare() error {
	if err := createDirectories([]string{
		c.inputDir,
		c.workDir,
	}); err != nil {
		return fmt.Errorf("creating directories failed: %w", err)
	}

	if err := copyInputFile(c.opts.SourceFile, c.inputFile); err != nil {
		return fmt.Errorf("copying changed file failed: %w", err)
	}

	return nil
}

func (c *Command) Run(ctx context.Context) (err error) {
	logger := c.opts.Logger

	if err := c.prepare(); err != nil {
		return err
	}

	outputHandle, err := os.OpenFile(c.outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_EXCL, 0o666)
	if err != nil {
		return fmt.Errorf("opening output file failed: %w", err)
	}

	defer multierr.AppendInvoke(&err, multierr.Close(outputHandle))

	cmd := exec.CommandContext(ctx, c.opts.Command[0], c.opts.Command[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = outputHandle
	cmd.Stderr = outputHandle
	cmd.Dir = c.workDir
	cmd.Env = append(append([]string(nil), os.Environ()...), c.environ...)

	logValues := []zapcore.Field{
		zap.String("dir", cmd.Dir),
		zap.Strings("env", c.environ),
		zap.Strings("args", cmd.Args),
	}

	if deadline, ok := ctx.Deadline(); ok {
		logValues = append(logValues,
			zap.Time("deadline", deadline),
			zap.Duration("deadline_remaining", time.Until(deadline)),
		)
	}

	logger.Info("Run handler command", logValues...)

	return runCommand(ctx, logger, c.opts.Metrics, cmd)
}
