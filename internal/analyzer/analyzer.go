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
	DefaultTopN   = 5
	DefaultMaxLen = 10_000
)

type WordFreq struct {
	Word  string
	Count int
}

type Result struct {
	Characters    int
	Words         int
	UniqueWords   int
	Sentences     int
	AvgWordLength float64
	TopWords      []WordFreq
}

func (r Result) Clone() Result {
	r.TopWords = slices.Clone(r.TopWords)
	return r
}

func (r Result) Equal(o Result) bool {
	return r.Characters == o.Characters &&
		r.Words == o.Words &&
		r.UniqueWords == o.UniqueWords &&
		r.Sentences == o.Sentences &&
		r.AvgWordLength == o.AvgWordLength &&
		slices.Equal(r.TopWords, o.TopWords)
}

type TextAnalyzer struct {
	TopN   int
	MaxLen int
}

func New() TextAnalyzer {
	return TextAnalyzer{TopN: DefaultTopN, MaxLen: DefaultMaxLen}
}

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
			r = '\''
		}
		b.WriteRune(r)
	}
	flush()
	return words
}

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
		count++
	}
	return count
}

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
