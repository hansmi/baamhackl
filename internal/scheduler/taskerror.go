package scheduler

import (
	"errors"
	"fmt"
	"time"
)

const Stop time.Duration = -1

type TaskError struct {
	// Underlying error
	Err error

	// Delay before re-running the task. Use a negative value to make the error
	// permanent (i.e. don't schedule a retry).
	RetryDelay time.Duration
}

var _ error = (*TaskError)(nil)

func AsTaskError(err error) *TaskError {
	if err == nil {
		return nil
	}

	var te *TaskError

	if errors.As(err, &te) {
		return te
	}

	return &TaskError{Err: err, RetryDelay: Stop}
}

func (e *TaskError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

func (e *TaskError) Permanent() bool {
	if e == nil {
		return true
	}

	return e.RetryDelay < 0
}

func (e *TaskError) Error() string {
	var err error

	if e != nil {
		err = e.Err
	}

	return fmt.Sprint(err)
}
