package logger

import (
	"log/slog"
	"os"
)

// New returns a structured JSON logger with source location enabled.
// Level should be a valid slog level string: DEBUG, INFO, WARN, ERROR.
// Unrecognized values default to ERROR.
func New(level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelError
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     lvl,
	}))
}
