package agentsdk

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SDKError is the common interface for all SDK error types.
type SDKError interface {
	error
	sdkError() // marker method
}

// CLINotFoundError is returned when the claude CLI binary cannot be found.
type CLINotFoundError struct {
	SearchedPaths []string
}

func (e *CLINotFoundError) Error() string {
	return fmt.Sprintf("claude CLI not found (searched: %s)", strings.Join(e.SearchedPaths, ", "))
}

func (*CLINotFoundError) sdkError() {}

// CLIConnectionError is returned when the SDK cannot connect to the claude process.
type CLIConnectionError struct {
	Reason string
}

func (e *CLIConnectionError) Error() string {
	return fmt.Sprintf("claude CLI connection error: %s", e.Reason)
}

func (*CLIConnectionError) sdkError() {}

// ProcessError is returned when the claude subprocess exits with a non-zero code.
type ProcessError struct {
	ExitCode int
	Stderr   string
}

func (e *ProcessError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("claude process exited with code %d: %s", e.ExitCode, e.Stderr)
	}
	return fmt.Sprintf("claude process exited with code %d", e.ExitCode)
}

func (*ProcessError) sdkError() {}

// JSONDecodeError is returned when a stdout line contains invalid JSON.
type JSONDecodeError struct {
	Line          string
	OriginalError error
}

func (e *JSONDecodeError) Error() string {
	return fmt.Sprintf("invalid JSON on stdout: %s (line: %.100s)", e.OriginalError, e.Line)
}

func (e *JSONDecodeError) Unwrap() error {
	return e.OriginalError
}

func (*JSONDecodeError) sdkError() {}

// MessageParseError is returned when a message is missing required fields.
type MessageParseError struct {
	Data json.RawMessage
	Err  error
}

func (e *MessageParseError) Error() string {
	return fmt.Sprintf("failed to parse message: %s (data: %.200s)", e.Err, string(e.Data))
}

func (e *MessageParseError) Unwrap() error {
	return e.Err
}

func (*MessageParseError) sdkError() {}
