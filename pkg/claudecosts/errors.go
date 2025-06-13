package claudecosts

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrNoClaudeDir   = errors.New("claude directory not found")
	ErrNoJSONLFiles  = errors.New("no JSONL files found")
	ErrInvalidConfig = errors.New("invalid configuration")
	ErrParsingFailed = errors.New("failed to parse JSONL files")
)

// ParseError represents an error during file parsing
type ParseError struct {
	File string
	Err  error
}

func (e ParseError) Error() string {
	return fmt.Sprintf("failed to parse %s: %v", e.File, e.Err)
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}
