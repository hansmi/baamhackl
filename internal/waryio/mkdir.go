package waryio

import (
	"os"
)

func MakeAvailableDir(g StringIter) (string, error) {
	for path, ok := g.Next(); ok; {
		err := os.Mkdir(path, os.ModePerm)
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
