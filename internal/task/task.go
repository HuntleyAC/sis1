// Package task defines the Task model and its lifecycle (state machine).
package task

import (
	"fmt"
	"slices"

	"sis-textanalyzer/internal/analyzer"
)

// ID uniquely identifies a task.
type ID string

// State is a step in the task lifecycle.
type State int

// The zero value of State is StatePending, so a new Task starts as pending.
const (
	StatePending State = iota
	StateProcessing
	StateCompleted
	StateFailed
)

var stateNames = map[State]string{
	StatePending:    "pending",
	StateProcessing: "processing",
	StateCompleted:  "completed",
	StateFailed:     "failed",
}

// transitions lists, for each state, the states it may move to.
// States with no outgoing transitions (completed, failed) are terminal.
var transitions = map[State][]State{
	StatePending:    {StateProcessing},
	StateProcessing: {StateCompleted, StateFailed},
}

// String implements fmt.Stringer.
func (s State) String() string {
	if name, ok := stateNames[s]; ok {
		return name
	}
	return fmt.Sprintf("State(%d)", int(s))
}

// CanTransitionTo reports whether moving from s to next is allowed.
func (s State) CanTransitionTo(next State) bool {
	return slices.Contains(transitions[s], next)
}

// IsTerminal reports whether no further transitions are possible from s.
func (s State) IsTerminal() bool {
	return len(transitions[s]) == 0
}

// Task is a single unit of work: a text to analyze, its lifecycle state and,
// once completed, the analysis result.
type Task struct {
	ID     ID
	Input  string
	State  State
	Result *analyzer.Result // nil until the task is completed
	Err    string           // failure reason, set only in StateFailed
}

// New returns a pending task with the given ID and input.
func New(id ID, input string) Task {
	return Task{ID: id, Input: input, State: StatePending}
}

// TransitionTo moves the task to next, or returns a *TransitionError if the
// lifecycle does not allow that change. The task is not modified on error.
func (t *Task) TransitionTo(next State) error {
	if !t.State.CanTransitionTo(next) {
		return &TransitionError{From: t.State, To: next}
	}
	t.State = next
	return nil
}

// Complete moves the task to StateCompleted and attaches the result.
func (t *Task) Complete(r analyzer.Result) error {
	if err := t.TransitionTo(StateCompleted); err != nil {
		return err
	}
	t.Result = &r
	return nil
}

// Fail moves the task to StateFailed and records the cause.
func (t *Task) Fail(cause error) error {
	if err := t.TransitionTo(StateFailed); err != nil {
		return err
	}
	t.Err = cause.Error()
	return nil
}

// Clone returns a deep copy of t that shares no memory with the original.
//
// It has a pointer receiver like the other Task methods (one receiver kind
// per type). A plain copy (c := *t) is not enough, because Result is a
// pointer and Result.TopWords is a slice: both would still be shared.
func (t *Task) Clone() Task {
	c := *t
	if t.Result != nil {
		r := t.Result.Clone()
		c.Result = &r
	}
	return c
}
