package waryio

import (
	"os"

	"golang.org/x/sys/unix"
)

// renameWithoutReplace renames oldpath to newpath without replacing a file
// which may already exist at newpath.
func renameWithoutReplace(oldpath, newpath string) error {
	return unix.Renameat2(unix.AT_FDCWD, oldpath, unix.AT_FDCWD, newpath, unix.RENAME_NOREPLACE)
}

// RenameToAvailableName attempts to rename oldpath to a path produced by g.
// The used destination path is returned.
func RenameToAvailableName(oldpath string, g StringIter) (string, error) {
	for path, ok := g.Next(); ok; {
		err := renameWithoutReplace(oldpath, path)
		if err == nil {
			return path, nil
		}

		path, ok = g.Next()

		if !(ok && os.IsExist(err)) {
			return "", err
		}
	}

	return "", ErrIterExhausted
}
