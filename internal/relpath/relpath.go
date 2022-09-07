package relpath

import (
	"os"
	"path/filepath"
	"strings"
)

type Resolved struct {
	// Path to file or directory.
	Path string

	// Path in which relative paths are contained.
	Base string

	// Base-relative path to file or directory. May contain ".." elements.
	Relative string

	// Elements of base-relative path. May contain "..".
	RelativeElems []string
}

// Contained determines whether the path is fully contained within the base
// directory.
func (r Resolved) Contained() bool {
	for _, i := range r.RelativeElems {
		if i == ".." {
			return false
		}
	}

	return len(r.RelativeElems) > 0
}

// Resolve the path relative to base. Path may be absolute.
func Resolve(base, path string) (Resolved, error) {
	var err error

	r := Resolved{
		Base: filepath.Clean(base),
	}

	if filepath.IsAbs(path) {
		r.Path = filepath.Clean(path)
	} else {
		r.Path = filepath.Join(r.Base, path)
	}

	if r.Relative, err = filepath.Rel(r.Base, r.Path); err != nil {
		return Resolved{}, err
	}

	if r.RelativeElems = strings.Split(r.Relative, string(filepath.Separator)); len(r.RelativeElems) == 0 {
		return Resolved{}, os.ErrInvalid
	}

	return r, nil
}
