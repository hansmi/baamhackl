package exepath

import (
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/multierr"
)

type executableFunc func() (string, error)

type data struct {
	path string
	err  error
}

func resolve(fn executableFunc) data {
	if fn == nil {
		fn = os.Executable
	}

	var d data

	if d.path, d.err = fn(); d.err == nil {
		if path, err := filepath.Abs(d.path); err == nil {
			d.path = path
		} else {
			multierr.AppendInto(&d.err, err)
		}
	}

	return d
}

var global data
var globalOnce sync.Once

// Get returns the path name for the executable of the current process.
func Get() (string, error) {
	globalOnce.Do(func() {
		global = resolve(nil)
	})

	return global.path, global.err
}

// MustGet is like Get but panics if the executable is unknown.
func MustGet() string {
	path, err := Get()
	if err != nil {
		panic(err)
	}

	return path
}
