// Package store provides an in-memory task repository.
package store

import (
	"errors"
	"fmt"

	"sis-textanalyzer/internal/task"
)

// ErrEmptyID is returned when saving a task without an ID.
var ErrEmptyID = errors.New("task ID is empty")

// MemoryStore keeps tasks in memory.
//
// Tasks are stored and returned by value (as deep copies), so code outside
// the store cannot change a stored task without calling Save again.
//
// MemoryStore is not safe for concurrent use; the application is synchronous
// in Checkpoint 1.
type MemoryStore struct {
	tasks map[task.ID]task.Task
	order []task.ID // insertion order, used by List
}

// NewMemoryStore returns an empty store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{tasks: make(map[task.ID]task.Task)}
}

// Save inserts a new task or replaces an existing task with the same ID.
func (s *MemoryStore) Save(t task.Task) error {
	if t.ID == "" {
		return ErrEmptyID
	}
	if _, exists := s.tasks[t.ID]; !exists {
		s.order = append(s.order, t.ID)
	}
	s.tasks[t.ID] = t.Clone()
	return nil
}

// Get returns a copy of the task with the given ID, or an error wrapping
// task.ErrNotFound.
func (s *MemoryStore) Get(id task.ID) (task.Task, error) {
	t, ok := s.tasks[id]
	if !ok {
		return task.Task{}, fmt.Errorf("%w: %q", task.ErrNotFound, id)
	}
	return t.Clone(), nil
}

// List returns copies of all tasks in the order they were first saved.
func (s *MemoryStore) List() []task.Task {
	out := make([]task.Task, 0, len(s.order))
	for _, id := range s.order {
		// Clone has a pointer receiver and map elements are not addressable,
		// so s.tasks[id].Clone() would not compile: copy into a variable first.
		t := s.tasks[id]
		out = append(out, t.Clone())
	}
	return out
}
