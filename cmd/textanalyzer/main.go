// Command textanalyzer analyzes one or more texts from the command line.
//
// Each argument becomes a separate task. With no arguments, the whole of
// standard input is analyzed as a single task.
//
//	go run ./cmd/textanalyzer "First text." "Second text!"
//	echo "Some text" | go run ./cmd/textanalyzer
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"sis-textanalyzer/internal/analyzer"
	"sis-textanalyzer/internal/service"
	"sis-textanalyzer/internal/store"
	"sis-textanalyzer/internal/task"
)

// Compile-time checks: the build fails right here if a type stops
// implementing an interface that the service depends on.
var (
	_ service.Processor  = analyzer.TextAnalyzer{}
	_ service.Repository = (*store.MemoryStore)(nil)
)

func main() {
	topN := flag.Int("top", analyzer.DefaultTopN, "number of most frequent words to show")
	reprocess := flag.Bool("reprocess", false, "try to process each completed task again (shows that the lifecycle rejects it)")
	flag.Parse() // on an unknown flag, prints usage and exits with code 2

	os.Exit(run(flag.Args(), *topN, *reprocess, os.Stdin, os.Stdout, os.Stderr))
}

// run does the actual work and returns the exit code. It receives
// io.Reader / io.Writer instead of using os.Stdin / os.Stdout directly.
func run(inputs []string, topN int, reprocess bool, stdin io.Reader, stdout, stderr io.Writer) int {
	if topN < 1 {
		fmt.Fprintln(stderr, "error: -top must be at least 1")
		return 2
	}

	if len(inputs) == 0 {
		data, err := io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintf(stderr, "error: read stdin: %v\n", err)
			return 1
		}
		// Some shells (e.g. Windows PowerShell) prepend a UTF-8 byte order
		// mark and append a newline when piping; neither is part of the text.
		const bom = "\xef\xbb\xbf"
		text := strings.TrimPrefix(string(data), bom)
		inputs = []string{strings.TrimSpace(text)}
	}

	proc := analyzer.TextAnalyzer{TopN: topN, MaxLen: analyzer.DefaultMaxLen}
	svc, err := service.New(store.NewMemoryStore(), proc)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	exitCode := 0
	for i, input := range inputs {
		t, err := svc.Create(input)
		if err != nil {
			fmt.Fprintf(stderr, "input #%d rejected: %v\n", i+1, err)
			exitCode = 1
			continue
		}
		if _, err := svc.Process(t.ID); err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", t.ID, err)
			exitCode = 1
		}
	}

	for _, t := range svc.List() {
		printTask(stdout, t)
		if reprocess && t.State == task.StateCompleted {
			if _, err := svc.Process(t.ID); err != nil {
				fmt.Fprintf(stdout, "  reprocess:    rejected (%v)\n", err)
			}
		}
		fmt.Fprintln(stdout)
	}
	return exitCode
}

func printTask(w io.Writer, t task.Task) {
	fmt.Fprintf(w, "%s [%s]\n", t.ID, t.State)
	fmt.Fprintf(w, "  input:        %q\n", preview(t.Input, 60))
	if t.Err != "" {
		fmt.Fprintf(w, "  error:        %s\n", t.Err)
	}
	if t.Result == nil {
		return
	}
	r := t.Result
	fmt.Fprintf(w, "  characters:   %d\n", r.Characters)
	fmt.Fprintf(w, "  words:        %d (unique: %d)\n", r.Words, r.UniqueWords)
	fmt.Fprintf(w, "  sentences:    %d\n", r.Sentences)
	fmt.Fprintf(w, "  avg word len: %.2f\n", r.AvgWordLength)
	top := make([]string, len(r.TopWords))
	for i, wf := range r.TopWords {
		top[i] = fmt.Sprintf("%s (%d)", wf.Word, wf.Count)
	}
	fmt.Fprintf(w, "  top words:    %s\n", strings.Join(top, ", "))
}

// preview shortens s to at most n characters for display.
func preview(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[:n-1]) + "…"
}
