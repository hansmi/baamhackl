// Package cmdemu provides a way to emulate an external command from test code.
package cmdemu

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/hansmi/baamhackl/internal/exepath"
)

const ExitSuccess = 0
const ExitFailure = 1

// The command was used incorrectly, e.g., with the wrong number of
// arguments, a bad flag, a bad syntax in a parameter, or whatever.
const ExitUsage = 64

// Something does not work, but the exact reason is not known.
const ExitUnavailable = 69

type ExitCodeError int

var _ error = (*ExitCodeError)(nil)

func (e ExitCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e)
}

func (e ExitCodeError) Code() int {
	return int(e)
}

type TestingM interface {
	Run() int
}

type Command struct {
	Name    string
	Execute func(args []string) error
}

func (c *Command) MakeArgs(args ...string) []string {
	return append([]string{exepath.MustGet(), "-emulate", c.Name, "--"}, args...)
}

type Wrapper struct {
	selected string
	commands map[string]Command
}

func New(fs *flag.FlagSet) *Wrapper {
	w := &Wrapper{
		commands: map[string]Command{},
	}

	fs.StringVar(&w.selected, "emulate", "", "")

	return w
}

func (w *Wrapper) Register(cmd Command) {
	w.commands[cmd.Name] = cmd
}

func (w *Wrapper) run(stderr io.Writer, name string, args []string) int {
	cmd, ok := w.commands[name]
	if !ok {
		fmt.Fprintf(stderr, "Unknown command %q\n", name)
		return ExitUsage
	}

	err := cmd.Execute(args)
	if err == nil {
		return ExitSuccess
	}

	var exitCodeErr ExitCodeError

	if errors.As(err, &exitCodeErr) {
		return exitCodeErr.Code()
	}

	fmt.Fprintf(stderr, "Error: %v\n", err)

	return ExitUnavailable
}

func (w *Wrapper) Main(m TestingM) int {
	flag.Parse()

	if w.selected != "" {
		return w.run(os.Stderr, w.selected, flag.Args())
	}

	return m.Run()
}
