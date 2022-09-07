package waryio

import "errors"

var ErrIterExhausted = errors.New("iterator exhausted")

type StringIter interface {
	Next() (string, bool)
}
