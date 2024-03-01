package waryio

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

type sourceReader interface {
	io.Reader
	Stat() (os.FileInfo, error)
}

func copyInner(src sourceReader, dest io.Writer) (os.FileMode, error) {
	srcStatBefore, err := src.Stat()
	if err != nil {
		return 0, err
	}

	copiedBytes, err := io.Copy(dest, src)
	if err != nil {
		return 0, err
	}

	if srcStatAfter, err := src.Stat(); err != nil {
		return 0, err
	} else if err := DescribeChanges(srcStatBefore, srcStatAfter).Err(); err != nil {
		return 0, fmt.Errorf("source was modified: %w", err)
	}

	if copiedBytes != srcStatBefore.Size() {
		return 0, fmt.Errorf("copied %d bytes while source has %d bytes", copiedBytes, srcStatBefore.Size())
	}

	return srcStatBefore.Mode() & os.ModePerm, nil
}

type CopyOptions struct {
	SourcePath         string
	SourceFlags        int
	SourceMode         os.FileMode
	SourcePermPreserve bool

	DestPath  string
	DestFlags int
	DestMode  os.FileMode
	DestSync  bool
}

var DefaultCopyOptions = CopyOptions{
	SourceFlags:        os.O_RDONLY | syscall.O_NOFOLLOW,
	SourcePermPreserve: true,

	DestFlags: os.O_WRONLY | os.O_CREATE | os.O_TRUNC | syscall.O_NOFOLLOW,
	DestMode:  0o666,
}

// Copy creates an exact file copy. The operation fails if the source file is
// modified concurrently.
func Copy(opts CopyOptions) (err error) {
	var src, dest *os.File

	defer func() {
		for _, fh := range []*os.File{src, dest} {
			if fh == nil {
				continue
			}

			if closeErr := fh.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}
	}()

	src, err = os.OpenFile(opts.SourcePath, opts.SourceFlags, opts.SourceMode)
	if err != nil {
		return err
	}

	dest, err = os.OpenFile(opts.DestPath, opts.DestFlags, opts.DestMode)
	if err != nil {
		return err
	}

	if sourcePerm, err := copyInner(src, dest); err != nil {
		return err
	} else if opts.SourcePermPreserve {
		if err := dest.Chmod(sourcePerm); err != nil {
			return err
		}
	}

	if opts.DestSync {
		if err := dest.Sync(); err != nil {
			return err
		}
	}

	return nil
}
