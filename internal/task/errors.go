package task

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound is returned by repositories when no task has the given ID.
	ErrNotFound = errors.New("task not found")
	// ErrInvalidTransition is the sentinel wrapped by every *TransitionError.
	ErrInvalidTransition = errors.New("invalid state transition")
)

// TransitionError describes a rejected state change. It unwraps to
// ErrInvalidTransition, so callers can use either errors.Is or errors.As.
type TransitionError struct {
	From State
	To   State
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("%s: %s -> %s", ErrInvalidTransition, e.From, e.To)
}

func (e *TransitionError) Unwrap() error {
	return ErrInvalidTransition
}
