package cmdutil

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/subcommands"
	"github.com/hansmi/baamhackl/internal/exepath"
)

type fakeCommand struct {
	name     string
	synopsis string
	setFlags func(*flag.FlagSet)
}

func (c *fakeCommand) Name() string {
	return c.name
}

func (c *fakeCommand) Synopsis() string {
	return c.synopsis
}

func (c *fakeCommand) Usage() string {
	panic("not implemented")
}

func (c *fakeCommand) SetFlags(fs *flag.FlagSet) {
	if c.setFlags != nil {
		c.setFlags(fs)
	}
}

func (c *fakeCommand) Execute(context.Context, *flag.FlagSet, ...interface{}) subcommands.ExitStatus {
	panic("not implemented")
}

func TestUsage(t *testing.T) {
	program := filepath.Base(exepath.MustGet())

	for _, tc := range []struct {
		cmd  *fakeCommand
		args string
		desc string
		want string
	}{
		{
			cmd: &fakeCommand{
				name: "empty",
			},
			want: fmt.Sprintf("Usage: %s empty\n", program),
		},
		{
			cmd: &fakeCommand{
				name: "oneflag",
				setFlags: func(fs *flag.FlagSet) {
					fs.Bool("enable", false, "Description")
				},
			},
			want: fmt.Sprintf("Usage: %s oneflag [flags]\n\nFlags:\n", program),
		},
		{
			cmd: &fakeCommand{
				name: "withargsandflags",
				setFlags: func(fs *flag.FlagSet) {
					fs.Bool("check", true, "Description")
				},
			},
			args: "<first> [second]",
			want: fmt.Sprintf("Usage: %s withargsandflags [flags] <first> [second]\n\nFlags:\n", program),
		},
		{
			cmd: &fakeCommand{
				name: "desc",
			},
			args: "files...",
			desc: "First line.\nSecond line.",
			want: fmt.Sprintf("Usage: %s desc files...\n\nFirst line.\nSecond line.\n", program),
		},
	} {
		t.Run(tc.cmd.name, func(t *testing.T) {
			got := Usage(tc.cmd, tc.args, tc.desc)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Usage() diff (-want +got):\n%s", diff)
			}
		})
	}
}
