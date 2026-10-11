// Package service ties the pieces together: it creates tasks, stores them,
// runs them through a Processor and enforces the task lifecycle.
package service

import (
	"errors"
	"fmt"

	"sis-textanalyzer/internal/analyzer"
	"sis-textanalyzer/internal/task"
)

// ErrNilDependency is returned by New when a dependency is missing.
var ErrNilDependency = errors.New("service: nil dependency")

// Processor validates task input and computes the result.
// analyzer.TextAnalyzer implements it.
type Processor interface {
	Validate(input string) error
	Process(input string) (analyzer.Result, error)
}

// Repository stores tasks. Get must return an error wrapping
// task.ErrNotFound for unknown IDs. store.MemoryStore implements it.
type Repository interface {
	Save(t task.Task) error
	Get(id task.ID) (task.Task, error)
	List() []task.Task
}

// Service is the entry point for working with tasks.
type Service struct {
	repo   Repository
	proc   Processor
	nextID int
}

// New returns a Service that uses repo for storage and proc for processing.
func New(repo Repository, proc Processor) (*Service, error) {
	if repo == nil || proc == nil {
		return nil, ErrNilDependency
	}
	return &Service{repo: repo, proc: proc}, nil
}

// Create validates input and stores a new pending task.
func (s *Service) Create(input string) (task.Task, error) {
	if err := s.proc.Validate(input); err != nil {
		return task.Task{}, fmt.Errorf("create task: %w", err)
	}
	s.nextID++
	t := task.New(task.ID(fmt.Sprintf("task-%d", s.nextID)), input)
	if err := s.repo.Save(t); err != nil {
		return task.Task{}, fmt.Errorf("create task: %w", err)
	}
	return t, nil
}

// Get returns the task with the given ID.
func (s *Service) Get(id task.ID) (task.Task, error) {
	t, err := s.repo.Get(id)
	if err != nil {
		return task.Task{}, fmt.Errorf("get task: %w", err)
	}
	return t, nil
}

// List returns all tasks in creation order.
func (s *Service) List() []task.Task {
	return s.repo.List()
}

// Process runs a pending task: pending -> processing -> completed, or
// processing -> failed if the processor returns an error. Processing a task
// that is not pending (for example, one already completed) is rejected with
// an error wrapping task.ErrInvalidTransition, and the task is left unchanged.
//
// If processing fails, the failed task is returned together with the error.
func (s *Service) Process(id task.ID) (task.Task, error) {
	t, err := s.repo.Get(id)
	if err != nil {
		return task.Task{}, fmt.Errorf("process task: %w", err)
	}
	if err := t.TransitionTo(task.StateProcessing); err != nil {
		return task.Task{}, fmt.Errorf("process task %s: %w", id, err)
	}
	if err := s.repo.Save(t); err != nil {
		return task.Task{}, fmt.Errorf("process task %s: %w", id, err)
	}

	result, procErr := s.proc.Process(t.Input)
	if procErr != nil {
		if err := t.Fail(procErr); err != nil {
			return task.Task{}, fmt.Errorf("process task %s: %w", id, err)
		}
		if err := s.repo.Save(t); err != nil {
			return task.Task{}, fmt.Errorf("process task %s: %w", id, err)
		}
		return t, fmt.Errorf("process task %s: %w", id, procErr)
	}

	if err := t.Complete(result); err != nil {
		return task.Task{}, fmt.Errorf("process task %s: %w", id, err)
	}
	if err := s.repo.Save(t); err != nil {
		return task.Task{}, fmt.Errorf("process task %s: %w", id, err)
	}
	return t, nil
}
