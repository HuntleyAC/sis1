package analyzer

import "errors"

// Validation errors returned by TextAnalyzer.Validate. Callers should check
// them with errors.Is, because some of them are wrapped with extra context.
var (
	ErrInvalidUTF8  = errors.New("input is not valid UTF-8 text")
	ErrEmptyInput   = errors.New("input is empty")
	ErrInputTooLong = errors.New("input is too long")
	ErrNoWords      = errors.New("input contains no words to analyze")
)
