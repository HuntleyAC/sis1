package service

import (
	"errors"
	"strings"
	"testing"

	"sis-textanalyzer/internal/analyzer"
	"sis-textanalyzer/internal/store"
	"sis-textanalyzer/internal/task"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	svc, err := New(store.NewMemoryStore(), analyzer.New())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return svc
}

// failingProcessor accepts any input but always fails to process it.
// It lets the tests reach the "failed" state through the Processor interface.
type failingProcessor struct{ err error }

func (p failingProcessor) Validate(string) error { return nil }

func (p failingProcessor) Process(string) (analyzer.Result, error) {
	return analyzer.Result{}, p.err
}

func TestNew_NilDependencies(t *testing.T) {
	if _, err := New(nil, analyzer.New()); !errors.Is(err, ErrNilDependency) {
		t.Errorf("nil repo: error = %v, want ErrNilDependency", err)
	}
	if _, err := New(store.NewMemoryStore(), nil); !errors.Is(err, ErrNilDependency) {
		t.Errorf("nil processor: error = %v, want ErrNilDependency", err)
	}
}

// Requirement 1: successful creation.
func TestCreate_Success(t *testing.T) {
	svc := newTestService(t)

	tk, err := svc.Create("Hello world.")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if tk.ID == "" {
		t.Fatal("created task has empty ID")
	}
	if tk.Input != "Hello world." {
		t.Fatalf("Input = %q", tk.Input)
	}
	if tk.State != task.StatePending {
		t.Fatalf("State = %s, want pending", tk.State)
	}
	if tk.Result != nil {
		t.Fatalf("Result = %+v, want nil before processing", tk.Result)
	}
}

// Requirement 2: invalid input is rejected (table-driven).
func TestCreate_InvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"empty", "", analyzer.ErrEmptyInput},
		{"whitespace only", "   \n\t", analyzer.ErrEmptyInput},
		{"punctuation only", "?!... ---", analyzer.ErrNoWords},
		{"invalid utf-8", "\xff\xfe", analyzer.ErrInvalidUTF8},
		{"too long", strings.Repeat("a", analyzer.DefaultMaxLen+1), analyzer.ErrInputTooLong},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(t)
			_, err := svc.Create(tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if n := len(svc.List()); n != 0 {
				t.Fatalf("invalid input was stored: %d tasks in store", n)
			}
		})
	}
}

// Requirement 3: successful retrieval.
func TestGet_Success(t *testing.T) {
	svc := newTestService(t)
	created, err := svc.Create("Some text")
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// Task is comparable with ==: all its fields are comparable (Result is a
	// pointer). Both tasks are pending, so both Result pointers are nil.
	if got != created {
		t.Fatalf("Get = %+v, want %+v", got, created)
	}
}

// Requirement 4: retrieving a task that does not exist.
func TestGet_NotFound(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Get("task-404")
	if !errors.Is(err, task.ErrNotFound) {
		t.Fatalf("error = %v, want task.ErrNotFound", err)
	}
}

func TestProcess_NotFound(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.Process("task-404")
	if !errors.Is(err, task.ErrNotFound) {
		t.Fatalf("error = %v, want task.ErrNotFound", err)
	}
}

// Requirements 5, 6, 7: successful processing, correct result, correct state.
func TestProcess_Success(t *testing.T) {
	svc := newTestService(t)
	created, err := svc.Create("Hello world. Hello Go!")
	if err != nil {
		t.Fatal(err)
	}

	processed, err := svc.Process(created.ID)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	want := analyzer.Result{
		Characters:    22,
		Words:         4,
		UniqueWords:   3,
		Sentences:     2,
		AvgWordLength: 4.25,
		TopWords:      []analyzer.WordFreq{{Word: "hello", Count: 2}, {Word: "go", Count: 1}, {Word: "world", Count: 1}},
	}

	if processed.State != task.StateCompleted {
		t.Fatalf("State = %s, want completed", processed.State)
	}
	if processed.Result == nil || !processed.Result.Equal(want) {
		t.Fatalf("Result = %+v, want %+v", processed.Result, want)
	}

	// The stored task must also be completed and hold the same result.
	stored, err := svc.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State != task.StateCompleted {
		t.Fatalf("stored State = %s, want completed", stored.State)
	}
	if stored.Result == nil || !stored.Result.Equal(want) {
		t.Fatalf("stored Result = %+v, want %+v", stored.Result, want)
	}
}

// Requirement 8: invalid operation, processing a task that is already completed.
func TestProcess_AlreadyCompleted(t *testing.T) {
	svc := newTestService(t)
	created, _ := svc.Create("Process me once.")
	first, err := svc.Process(created.ID)
	if err != nil {
		t.Fatalf("first Process: %v", err)
	}

	_, err = svc.Process(created.ID)
	if !errors.Is(err, task.ErrInvalidTransition) {
		t.Fatalf("second Process: error = %v, want ErrInvalidTransition", err)
	}
	var te *task.TransitionError
	if !errors.As(err, &te) || te.From != task.StateCompleted || te.To != task.StateProcessing {
		t.Fatalf("error = %v, want completed -> processing transition error", err)
	}

	// The rejected operation must leave the task untouched.
	stored, _ := svc.Get(created.ID)
	// stored != first would compare Result pointers (different copies), so
	// compare the state and the result contents instead.
	if stored.State != first.State || stored.Result == nil || !stored.Result.Equal(*first.Result) {
		t.Fatalf("task changed after rejected reprocess:\n got  %+v\n want %+v", stored, first)
	}
}

func TestProcess_ProcessorError_SetsFailed(t *testing.T) {
	procErr := errors.New("analysis engine unavailable")
	svc, err := New(store.NewMemoryStore(), failingProcessor{err: procErr})
	if err != nil {
		t.Fatal(err)
	}
	created, _ := svc.Create("some text")

	failed, err := svc.Process(created.ID)
	if !errors.Is(err, procErr) {
		t.Fatalf("error = %v, want %v", err, procErr)
	}
	if failed.State != task.StateFailed || failed.Err != procErr.Error() || failed.Result != nil {
		t.Fatalf("got %+v, want failed task with error message and no result", failed)
	}

	// A failed task is terminal: processing it again is an invalid transition.
	if _, err := svc.Process(created.ID); !errors.Is(err, task.ErrInvalidTransition) {
		t.Fatalf("reprocess failed task: error = %v, want ErrInvalidTransition", err)
	}
}

// Requirement 9: multiple tasks are independent of each other.
func TestMultipleTasks_Independent(t *testing.T) {
	svc := newTestService(t)
	a, err := svc.Create("one two three")
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create("Alpha. Beta. Alpha!")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID {
		t.Fatalf("tasks share ID %s", a.ID)
	}

	// Processing a must not touch b.
	if _, err := svc.Process(a.ID); err != nil {
		t.Fatal(err)
	}
	gotB, _ := svc.Get(b.ID)
	if gotB.State != task.StatePending || gotB.Result != nil {
		t.Fatalf("task b changed when a was processed: %+v", gotB)
	}

	if _, err := svc.Process(b.ID); err != nil {
		t.Fatal(err)
	}
	gotA, _ := svc.Get(a.ID)
	gotB, _ = svc.Get(b.ID)

	if gotA.Result.Words != 3 || gotA.Result.Sentences != 1 {
		t.Errorf("task a result = %+v, want 3 words, 1 sentence", *gotA.Result)
	}
	if gotB.Result.Words != 3 || gotB.Result.Sentences != 3 || gotB.Result.TopWords[0] != (analyzer.WordFreq{Word: "alpha", Count: 2}) {
		t.Errorf("task b result = %+v, want 3 words, 3 sentences, top word alpha x2", *gotB.Result)
	}

	list := svc.List()
	if len(list) != 2 || list[0].ID != a.ID || list[1].ID != b.ID {
		t.Fatalf("List = %+v, want [%s %s]", list, a.ID, b.ID)
	}
}
