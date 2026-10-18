package task

import (
	"fmt"
	"slices"

	"sis-textanalyzer/internal/analyzer"
)

type ID string

type State int

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

var transitions = map[State][]State{
	StatePending:    {StateProcessing},
	StateProcessing: {StateCompleted, StateFailed},
}

func (s State) String() string {
	if name, ok := stateNames[s]; ok {
		return name
	}
	return fmt.Sprintf("State(%d)", int(s))
}

func (s State) CanTransitionTo(next State) bool {
	return slices.Contains(transitions[s], next)
}

func (s State) IsTerminal() bool {
	return len(transitions[s]) == 0
}

type Task struct {
	ID     ID
	Input  string
	State  State
	Result *analyzer.Result
	Err    string
}

func New(id ID, input string) Task {
	return Task{ID: id, Input: input, State: StatePending}
}

func (t *Task) TransitionTo(next State) error {
	if !t.State.CanTransitionTo(next) {
		return &TransitionError{From: t.State, To: next}
	}
	t.State = next
	return nil
}

func (t *Task) Complete(r analyzer.Result) error {
	if err := t.TransitionTo(StateCompleted); err != nil {
		return err
	}
	t.Result = &r
	return nil
}

func (t *Task) Fail(cause error) error {
	if err := t.TransitionTo(StateFailed); err != nil {
		return err
	}
	t.Err = cause.Error()
	return nil
}

func (t *Task) Clone() Task {
	c := *t
	if t.Result != nil {
		r := t.Result.Clone()
		c.Result = &r
	}
	return c
}
