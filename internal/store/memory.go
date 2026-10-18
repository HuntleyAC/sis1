package store

import (
	"errors"
	"fmt"

	"sis-textanalyzer/internal/task"
)

var ErrEmptyID = errors.New("task ID is empty")

type MemoryStore struct {
	tasks map[task.ID]task.Task
	order []task.ID
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{tasks: make(map[task.ID]task.Task)}
}

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

func (s *MemoryStore) Get(id task.ID) (task.Task, error) {
	t, ok := s.tasks[id]
	if !ok {
		return task.Task{}, fmt.Errorf("%w: %q", task.ErrNotFound, id)
	}
	return t.Clone(), nil
}

func (s *MemoryStore) List() []task.Task {
	out := make([]task.Task, 0, len(s.order))
	for _, id := range s.order {
		t := s.tasks[id]
		out = append(out, t.Clone())
	}
	return out
}
