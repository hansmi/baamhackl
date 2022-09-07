package move

import (
	"context"
	"errors"
	"flag"
	"path/filepath"

	"github.com/google/subcommands"
	"github.com/hansmi/baamhackl/internal/cmdutil"
	"github.com/hansmi/baamhackl/internal/uniquename"
	"github.com/hansmi/baamhackl/internal/waryio"
	"go.uber.org/zap"
)

var errRenameNotSupported = errors.New("prefered destination names are only supported with a single source file")

// IntoCommand implements the "move-into" subcommand.
type IntoCommand struct {
	rename string
}

func (*IntoCommand) Name() string {
	return "move-into"
}

func (*IntoCommand) Synopsis() string {
	return `Move file(s) to a directory without overwriting.`
}

func (c *IntoCommand) Usage() string {
	return cmdutil.Usage(c, "<target_dir> <source...>", `Source files are moved into the target directory. File name conflicts with existing files are resolved by finding another, available name derived from the original name.`)
}

func (c *IntoCommand) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.rename, "rename", "", "Preferred destination name. When set only a single source file can be used.")
}

func (c *IntoCommand) execute(targetDir string, sourceFiles []string) error {
	logger := zap.L()

	opts := uniquename.DefaultOptions
	opts.TimePrefixEnabled = false

	for _, oldpath := range sourceFiles {
		newname := filepath.Base(oldpath)
		if c.rename != "" {
			if len(sourceFiles) > 1 {
				return errRenameNotSupported
			}
			newname = c.rename
		}
		newpath := filepath.Join(targetDir, newname)

		g, err := uniquename.New(newpath, opts)
		if err != nil {
			return err
		}

		actual, err := waryio.RenameToAvailableName(oldpath, g)
		if err != nil {
			return err
		}

		logger.Info("File moved successfully",
			zap.String("src", oldpath), zap.String("dest", actual))
	}

	return nil
}

func (c *IntoCommand) Execute(ctx context.Context, fs *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if fs.NArg() < 2 {
		fs.Usage()

		return subcommands.ExitUsageError
	}

	return cmdutil.ExecuteStatus(c.execute(fs.Arg(0), fs.Args()[1:]))
}
