package cmdutil

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/subcommands"
	"github.com/hansmi/baamhackl/internal/exepath"
	"github.com/mitchellh/go-wordwrap"
)

// Usage generates a description of a subcommand.
func Usage(cmd subcommands.Command, args, description string) string {
	var buf strings.Builder

	foundFlags := false

	fs := flag.NewFlagSet(cmd.Name(), flag.PanicOnError)
	cmd.SetFlags(fs)
	fs.VisitAll(func(*flag.Flag) { foundFlags = true })

	fmt.Fprintf(&buf, "Usage: %s %s", filepath.Base(exepath.MustGet()), cmd.Name())

	if foundFlags {
		buf.WriteString(" [flags]")
	}

	if args != "" {
		fmt.Fprintf(&buf, " %s", args)
	}

	buf.WriteString("\n")

	for _, i := range []string{
		cmd.Synopsis(),
		description,
	} {
		if text := strings.TrimSpace(i); text != "" {
			fmt.Fprintf(&buf, "\n%s\n", wordwrap.WrapString(text, 78))
		}
	}

	if foundFlags {
		fmt.Fprintf(&buf, "\nFlags:\n")
	}

	return buf.String()
}
