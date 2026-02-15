package errs

import "fmt"

// Kind categorizes application errors for HTTP status mapping.
type Kind int

const (
	// Unknown represents an unclassified error.
	Unknown Kind = iota
	// InvalidInput indicates the request was malformed (HTTP 400).
	InvalidInput
	// Unreachable indicates the target URL could not be reached (HTTP 502).
	Unreachable
	// Timeout indicates the target took too long to respond (HTTP 504).
	Timeout
	// ParsingFailed indicates the response could not be parsed (HTTP 500).
	ParsingFailed
)

// AppError carries a category, user message, and original cause.
type AppError struct {
	Kind           Kind
	UpstreamStatus int // HTTP status code returned by the target domain
	Message        string
	Cause          error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}
