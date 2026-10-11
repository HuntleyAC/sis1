package analyzer

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		maxLen  int
		input   string
		wantErr error
	}{
		{name: "valid text", input: "Hello, world!", wantErr: nil},
		{name: "single word", input: "Go", wantErr: nil},
		{name: "empty", input: "", wantErr: ErrEmptyInput},
		{name: "only whitespace", input: " \n\t ", wantErr: ErrEmptyInput},
		{name: "only punctuation", input: "!!! ... ???", wantErr: ErrNoWords},
		{name: "invalid utf-8", input: "abc\xff", wantErr: ErrInvalidUTF8},
		{name: "too long", maxLen: 5, input: "abcdef", wantErr: ErrInputTooLong},
		{name: "exactly max length", maxLen: 6, input: "abcdef", wantErr: nil},
		{name: "length counted in characters, not bytes", maxLen: 6, input: "привет", wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := TextAnalyzer{MaxLen: tc.maxLen}
			err := a.Validate(tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Validate(%q) error = %v, want %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestProcess(t *testing.T) {
	tests := []struct {
		name  string
		topN  int
		input string
		want  Result
	}{
		{
			name:  "basic two sentences",
			input: "Hello world. Hello Go!",
			want: Result{
				Characters: 22, Words: 4, UniqueWords: 3, Sentences: 2, AvgWordLength: 4.25,
				TopWords: []WordFreq{{"hello", 2}, {"go", 1}, {"world", 1}},
			},
		},
		{
			name:  "case insensitive and no terminator",
			input: "Go go GO",
			want: Result{
				Characters: 8, Words: 3, UniqueWords: 1, Sentences: 1, AvgWordLength: 2,
				TopWords: []WordFreq{{"go", 3}},
			},
		},
		{
			name:  "cyrillic text",
			input: "Привет, мир! Привет.",
			want: Result{
				Characters: 20, Words: 3, UniqueWords: 2, Sentences: 2, AvgWordLength: 5,
				TopWords: []WordFreq{{"привет", 2}, {"мир", 1}},
			},
		},
		{
			name:  "apostrophes stay inside words",
			input: "Don't stop, don’t!",
			want: Result{
				Characters: 18, Words: 3, UniqueWords: 2, Sentences: 1, AvgWordLength: 4.67,
				TopWords: []WordFreq{{"don't", 2}, {"stop", 1}},
			},
		},
		{
			name:  "decimal numbers are one word and do not end a sentence",
			input: "Pi is 3.14. Really?",
			want: Result{
				Characters: 19, Words: 4, UniqueWords: 4, Sentences: 2, AvgWordLength: 3.5,
				TopWords: []WordFreq{{"3.14", 1}, {"is", 1}, {"pi", 1}, {"really", 1}},
			},
		},
		{
			name:  "repeated terminators end one sentence",
			input: "Wait... what?! Ok",
			want: Result{
				Characters: 17, Words: 3, UniqueWords: 3, Sentences: 3, AvgWordLength: 3.33,
				TopWords: []WordFreq{{"ok", 1}, {"wait", 1}, {"what", 1}},
			},
		},
		{
			name:  "ties broken alphabetically and limited to topN",
			topN:  2,
			input: "b a c b a",
			want: Result{
				Characters: 9, Words: 5, UniqueWords: 3, Sentences: 1, AvgWordLength: 1,
				TopWords: []WordFreq{{"a", 2}, {"b", 2}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := TextAnalyzer{TopN: tc.topN}
			got, err := a.Process(tc.input)
			if err != nil {
				t.Fatalf("Process(%q) unexpected error: %v", tc.input, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Process(%q)\n got  %+v\n want %+v", tc.input, got, tc.want)
			}
		})
	}
}

func TestProcess_RejectsInvalidInput(t *testing.T) {
	_, err := New().Process("   ")
	if !errors.Is(err, ErrEmptyInput) {
		t.Fatalf("Process error = %v, want %v", err, ErrEmptyInput)
	}
}

func TestProcess_IsDeterministic(t *testing.T) {
	input := strings.Repeat("the quick brown fox jumps over the lazy dog. ", 20)
	a := New()
	first, err := a.Process(input)
	if err != nil {
		t.Fatal(err)
	}
	for range 10 {
		got, err := a.Process(input)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, first) {
			t.Fatalf("results differ between runs:\n%+v\n%+v", first, got)
		}
	}
}

func TestResultClone_DoesNotShareTopWords(t *testing.T) {
	orig := Result{TopWords: []WordFreq{{"a", 1}}}
	c := orig.Clone()
	c.TopWords[0].Word = "changed"
	if orig.TopWords[0].Word != "a" {
		t.Fatalf("modifying the clone changed the original: %+v", orig.TopWords)
	}
}
