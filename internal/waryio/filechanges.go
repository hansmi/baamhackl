package waryio

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

var ErrFileChanged = errors.New("file changed")

type FileChanges []string

func (c *FileChanges) append(text string) {
	*c = append(*c, text)
}

// DescribeChanges produces a list of changes detected between the two given
// FileInfo instances.
func DescribeChanges(a, b os.FileInfo) FileChanges {
	var result FileChanges

	if !os.SameFile(a, b) {
		result.append("moved or replaced (not the same file)")
	}

	if aType, bType := a.Mode().Type(), b.Mode().Type(); aType != bType {
		result.append(fmt.Sprintf("type changed (%s != %s)", aType, bType))
	}

	if a.Size() != b.Size() {
		result.append(fmt.Sprintf("size changed (%d != %d)", a.Size(), b.Size()))
	}

	if !a.ModTime().Equal(b.ModTime()) {
		result.append(fmt.Sprintf("modification time changed (%s != %s)", a.ModTime(), b.ModTime()))
	}

	return result
}

func (c FileChanges) Empty() bool {
	return len(c) == 0
}

func (c FileChanges) Err() error {
	if len(c) == 0 {
		return nil
	}

	return fmt.Errorf("%w: %s", ErrFileChanged, strings.Join([]string(c), ", "))
}
