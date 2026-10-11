// Package analyzer contains the domain logic of the application: validating
// a piece of text and computing statistics about it.
//
// Everything here is pure and deterministic: the same input always produces
// the same Result, and nothing is stored or shared between calls.
package analyzer

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// DefaultTopN is the number of most frequent words reported by default.
	DefaultTopN = 5
	// DefaultMaxLen is the default maximum input length, in characters (runes).
	DefaultMaxLen = 10_000
)

// WordFreq is a word together with the number of times it occurs.
type WordFreq struct {
	Word  string
	Count int
}

// Result holds the statistics computed for one text.
type Result struct {
	Characters    int        // Unicode characters (runes), including spaces and punctuation
	Words         int        // total number of words
	UniqueWords   int        // number of distinct words (case-insensitive)
	Sentences     int        // number of sentences
	AvgWordLength float64    // average word length in characters, rounded to 2 decimals
	TopWords      []WordFreq // most frequent words: count desc, then alphabetically
}

// Clone returns a deep copy of r. Result contains a slice, so a plain
// assignment would share the TopWords backing array with the original.
func (r Result) Clone() Result {
	r.TopWords = slices.Clone(r.TopWords)
	return r
}

// Equal reports whether r and o hold the same statistics. Result cannot be
// compared with == because it contains a slice; WordFreq can, so the slices
// are compared element by element with slices.Equal.
func (r Result) Equal(o Result) bool {
	return r.Characters == o.Characters &&
		r.Words == o.Words &&
		r.UniqueWords == o.UniqueWords &&
		r.Sentences == o.Sentences &&
		r.AvgWordLength == o.AvgWordLength &&
		slices.Equal(r.TopWords, o.TopWords)
}

// TextAnalyzer validates and analyzes text. The zero value is ready to use
// and falls back to DefaultTopN and DefaultMaxLen.
type TextAnalyzer struct {
	TopN   int // how many frequent words to report
	MaxLen int // maximum accepted input length in characters
}

// New returns a TextAnalyzer with the default settings.
func New() TextAnalyzer {
	return TextAnalyzer{TopN: DefaultTopN, MaxLen: DefaultMaxLen}
}

// Validate checks that input can be analyzed. It returns nil for valid input
// or one of the Err* validation errors otherwise.
func (a TextAnalyzer) Validate(input string) error {
	if !utf8.ValidString(input) {
		return ErrInvalidUTF8
	}
	if strings.TrimSpace(input) == "" {
		return ErrEmptyInput
	}
	if n := utf8.RuneCountInString(input); n > a.maxLen() {
		return fmt.Errorf("%w: %d characters, limit is %d", ErrInputTooLong, n, a.maxLen())
	}
	if len(tokenize(input)) == 0 {
		return ErrNoWords
	}
	return nil
}

// Process validates input and computes its statistics.
func (a TextAnalyzer) Process(input string) (Result, error) {
	if err := a.Validate(input); err != nil {
		return Result{}, err
	}

	words := tokenize(input)
	freq := make(map[string]int, len(words))
	totalLen := 0
	for _, w := range words {
		freq[w]++
		totalLen += utf8.RuneCountInString(w)
	}

	return Result{
		Characters:    utf8.RuneCountInString(input),
		Words:         len(words),
		UniqueWords:   len(freq),
		Sentences:     countSentences(input),
		AvgWordLength: round2(float64(totalLen) / float64(len(words))),
		TopWords:      topWords(freq, a.topN()),
	}, nil
}

func (a TextAnalyzer) topN() int {
	if a.TopN <= 0 {
		return DefaultTopN
	}
	return a.TopN
}

func (a TextAnalyzer) maxLen() int {
	if a.MaxLen <= 0 {
		return DefaultMaxLen
	}
	return a.MaxLen
}

// tokenize splits text into lowercase words. A word is a run of letters and
// digits; apostrophes inside a word ("don't") and decimal separators inside a
// number ("3.14", "1,000") are kept as part of the word.
func tokenize(text string) []string {
	rs := []rune(text)
	var words []string
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			words = append(words, strings.ToLower(b.String()))
			b.Reset()
		}
	}
	for i, r := range rs {
		if !isWordPart(rs, i) {
			flush()
			continue
		}
		if isApostrophe(r) {
			r = '\'' // treat "don’t" and "don't" as the same word
		}
		b.WriteRune(r)
	}
	flush()
	return words
}

// countSentences counts sentences: runs of text that contain at least one
// word and end with '.', '!', '?' or '…' (or with the end of the text).
// Repeated terminators such as "?!" or "..." end only one sentence.
func countSentences(text string) int {
	rs := []rune(text)
	count := 0
	inSentence := false
	for i, r := range rs {
		switch {
		case isWordPart(rs, i):
			inSentence = true
		case isSentenceEnd(r) && inSentence:
			count++
			inSentence = false
		}
	}
	if inSentence {
		count++ // last sentence has no terminator
	}
	return count
}

// topWords returns the n most frequent words, ordered by count (descending)
// and then alphabetically, so that ties are always resolved the same way.
func topWords(freq map[string]int, n int) []WordFreq {
	list := make([]WordFreq, 0, len(freq))
	for w, c := range freq {
		list = append(list, WordFreq{Word: w, Count: c})
	}
	slices.SortFunc(list, func(x, y WordFreq) int {
		if c := cmp.Compare(y.Count, x.Count); c != 0 {
			return c
		}
		return cmp.Compare(x.Word, y.Word)
	})
	if len(list) > n {
		list = list[:n]
	}
	return list
}

// isWordPart reports whether rs[i] belongs to a word, looking at its
// neighbours for apostrophes and decimal separators.
func isWordPart(rs []rune, i int) bool {
	r := rs[i]
	if isLetterOrDigit(r) {
		return true
	}
	if i == 0 || i == len(rs)-1 {
		return false
	}
	prev, next := rs[i-1], rs[i+1]
	switch {
	case isApostrophe(r):
		return isLetterOrDigit(prev) && isLetterOrDigit(next)
	case r == '.' || r == ',':
		return unicode.IsDigit(prev) && unicode.IsDigit(next)
	}
	return false
}

func isLetterOrDigit(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isApostrophe(r rune) bool {
	return r == '\'' || r == '’'
}

func isSentenceEnd(r rune) bool {
	return r == '.' || r == '!' || r == '?' || r == '…'
}

func round2(x float64) float64 {
	return math.Round(x*100) / 100
}
