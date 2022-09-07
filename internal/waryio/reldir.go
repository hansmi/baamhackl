package waryio

import (
	"os"
	"path/filepath"

	"github.com/hansmi/baamhackl/internal/relpath"
	"golang.org/x/sys/unix"
)

// EnsureRelDir creates a directory if and only if it is a subdirectory of
// base. The base directory must exist. Path may be an absolute path.
//
// On success the cleaned path to the created directory is returned.
//
// If path does not resolve to a subdirectory of base the directory is not
// created and the function merely returns the path.
func EnsureRelDir(base, path string, perm os.FileMode) (string, error) {
	r, err := relpath.Resolve(base, path)
	if err != nil {
		return "", err
	}

	if r.Contained() {
		for idx := range r.RelativeElems {
			partpath := filepath.Join(append([]string{r.Base}, r.RelativeElems[:idx+1]...)...)

			if st, err := os.Stat(partpath); err == nil {
				if st.IsDir() {
					continue
				}

				return "", &os.PathError{Op: "mkdir", Path: partpath, Err: unix.ENOTDIR}
			}

			if err := os.Mkdir(partpath, perm); err != nil {
				ignore := false

				if os.IsExist(err) {
					// The directory may have been created concurrently.
					if st, statErr := os.Stat(partpath); !(statErr == nil && st.IsDir()) {
						ignore = true
					}
				}

				if !ignore {
					return "", err
				}
			}
		}
	}

	return r.Path, nil
}
