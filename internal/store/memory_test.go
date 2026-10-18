package store

import (
	"errors"
	"testing"

	"sis-textanalyzer/internal/analyzer"
	"sis-textanalyzer/internal/task"
)

func TestSaveAndGet(t *testing.T) {
	s := NewMemoryStore()
	want := task.New("task-1", "hello")
	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Get("task-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != want.ID || got.Input != want.Input || got.State != want.State {
		t.Fatalf("Get = %+v, want %+v", got, want)
	}
}

func TestGet_NotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.Get("missing")
	if !errors.Is(err, task.ErrNotFound) {
		t.Fatalf("error = %v, want task.ErrNotFound", err)
	}
}

func TestSave_EmptyID(t *testing.T) {
	s := NewMemoryStore()
	if err := s.Save(task.Task{Input: "x"}); !errors.Is(err, ErrEmptyID) {
		t.Fatalf("error = %v, want ErrEmptyID", err)
	}
}

func TestSave_ReplacesExisting(t *testing.T) {
	s := NewMemoryStore()
	tk := task.New("task-1", "hello")
	_ = s.Save(tk)
	tk.State = task.StateProcessing
	_ = s.Save(tk)

	got, _ := s.Get("task-1")
	if got.State != task.StateProcessing {
		t.Fatalf("State = %s, want processing", got.State)
	}
	if n := len(s.List()); n != 1 {
		t.Fatalf("List has %d tasks, want 1", n)
	}
}

func TestStore_ReturnsCopies(t *testing.T) {
	s := NewMemoryStore()
	tk := task.Task{ID: "task-1", State: task.StateCompleted, Result: &analyzer.Result{
		TopWords: []analyzer.WordFreq{{Word: "go", Count: 1}},
	}}
	_ = s.Save(tk)

	tk.Result.TopWords[0].Word = "changed-after-save"

	got, _ := s.Get("task-1")
	got.State = task.StatePending
	got.Result.TopWords[0].Word = "changed-after-get"

	again, _ := s.Get("task-1")
	if again.State != task.StateCompleted {
		t.Fatalf("State = %s, want completed", again.State)
	}
	if w := again.Result.TopWords[0].Word; w != "go" {
		t.Fatalf("TopWords[0].Word = %q, want %q", w, "go")
	}
}

func TestList_PreservesInsertionOrder(t *testing.T) {
	s := NewMemoryStore()
	ids := []task.ID{"task-1", "task-2", "task-10", "task-3"}
	for _, id := range ids {
		_ = s.Save(task.New(id, "text"))
	}
	list := s.List()
	if len(list) != len(ids) {
		t.Fatalf("List has %d tasks, want %d", len(list), len(ids))
	}
	for i, tk := range list {
		if tk.ID != ids[i] {
			t.Fatalf("List[%d].ID = %s, want %s", i, tk.ID, ids[i])
		}
	}
}
