package task

import (
	"errors"
	"testing"

	"sis-textanalyzer/internal/analyzer"
)

func TestNew_StartsPending(t *testing.T) {
	tk := New("task-1", "some text")
	if tk.State != StatePending {
		t.Fatalf("State = %s, want %s", tk.State, StatePending)
	}
	if tk.Result != nil {
		t.Fatalf("Result = %+v, want nil", tk.Result)
	}
}

func TestTransitionTo(t *testing.T) {
	all := []State{StatePending, StateProcessing, StateCompleted, StateFailed}
	allowed := map[[2]State]bool{
		{StatePending, StateProcessing}:   true,
		{StateProcessing, StateCompleted}: true,
		{StateProcessing, StateFailed}:    true,
	}

	// Every pair of states is checked: only the three transitions above are allowed.
	for _, from := range all {
		for _, to := range all {
			wantOK := allowed[[2]State{from, to}]
			t.Run(from.String()+"->"+to.String(), func(t *testing.T) {
				tk := Task{ID: "task-1", State: from}
				err := tk.TransitionTo(to)

				if wantOK {
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if tk.State != to {
						t.Fatalf("State = %s, want %s", tk.State, to)
					}
					return
				}

				if !errors.Is(err, ErrInvalidTransition) {
					t.Fatalf("error = %v, want ErrInvalidTransition", err)
				}
				var te *TransitionError
				if !errors.As(err, &te) || te.From != from || te.To != to {
					t.Fatalf("error = %#v, want *TransitionError{%s, %s}", err, from, to)
				}
				if tk.State != from {
					t.Fatalf("state changed on rejected transition: %s, want %s", tk.State, from)
				}
			})
		}
	}
}

func TestIsTerminal(t *testing.T) {
	tests := []struct {
		state State
		want  bool
	}{
		{StatePending, false},
		{StateProcessing, false},
		{StateCompleted, true},
		{StateFailed, true},
	}
	for _, tc := range tests {
		if got := tc.state.IsTerminal(); got != tc.want {
			t.Errorf("%s.IsTerminal() = %v, want %v", tc.state, got, tc.want)
		}
	}
}

func TestComplete_RequiresProcessing(t *testing.T) {
	tk := New("task-1", "text")
	if err := tk.Complete(analyzer.Result{Words: 1}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Complete on pending task: error = %v, want ErrInvalidTransition", err)
	}
	if tk.Result != nil {
		t.Fatal("result was attached even though the transition was rejected")
	}
}

func TestFail_RecordsCause(t *testing.T) {
	tk := Task{ID: "task-1", State: StateProcessing}
	if err := tk.Fail(errors.New("boom")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tk.State != StateFailed || tk.Err != "boom" {
		t.Fatalf("got state %s, err %q; want failed, \"boom\"", tk.State, tk.Err)
	}
}

func TestClone_IsDeep(t *testing.T) {
	orig := Task{ID: "task-1", Result: &analyzer.Result{
		Words:    1,
		TopWords: []analyzer.WordFreq{{Word: "go", Count: 1}},
	}}
	c := orig.Clone()
	c.Result.Words = 99
	c.Result.TopWords[0].Word = "changed"

	if orig.Result.Words != 1 || orig.Result.TopWords[0].Word != "go" {
		t.Fatalf("modifying the clone changed the original: %+v", orig.Result)
	}
}

func TestStateString(t *testing.T) {
	if got := StateCompleted.String(); got != "completed" {
		t.Errorf("StateCompleted.String() = %q", got)
	}
	if got := State(42).String(); got != "State(42)" {
		t.Errorf("State(42).String() = %q", got)
	}
}
