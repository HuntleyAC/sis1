package service

import (
	"errors"
	"fmt"

	"sis-textanalyzer/internal/analyzer"
	"sis-textanalyzer/internal/task"
)

var ErrNilDependency = errors.New("service: nil dependency")

type Processor interface {
	Validate(input string) error
	Process(input string) (analyzer.Result, error)
}

type Repository interface {
	Save(t task.Task) error
	Get(id task.ID) (task.Task, error)
	List() []task.Task
}

type Service struct {
	repo   Repository
	proc   Processor
	nextID int
}

func New(repo Repository, proc Processor) (*Service, error) {
	if repo == nil || proc == nil {
		return nil, ErrNilDependency
	}
	return &Service{repo: repo, proc: proc}, nil
}

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

func (s *Service) Get(id task.ID) (task.Task, error) {
	t, err := s.repo.Get(id)
	if err != nil {
		return task.Task{}, fmt.Errorf("get task: %w", err)
	}
	return t, nil
}

func (s *Service) List() []task.Task {
	return s.repo.List()
}

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
