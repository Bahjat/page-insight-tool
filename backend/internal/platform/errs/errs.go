package errs

import "fmt"

// Kind categorizes application errors for HTTP status mapping.
type Kind int

const (
	Unknown       Kind = iota
	InvalidInput       // maps to 400
	Unreachable        // maps to 502
	ParsingFailed      // maps to 500
)

// AppError carries a category, user message, and original cause.
type AppError struct {
	Kind    Kind
	Code    int // HTTP status code returned by the target domain
	Message string
	Cause   error
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
