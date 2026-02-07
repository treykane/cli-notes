// Package logging provides a shared, structured logger for the cli-notes application.
//
// It wraps the standard library's [log/slog] package and provides a single
// initialization point so all components share the same output handler and
// log level. The log level can be controlled at startup via the
// CLI_NOTES_LOG_LEVEL environment variable (debug, info, warn, error).
// If unset, the default level is INFO.
//
// Usage:
//
//	log := logging.New("config")       // creates a logger tagged with component="config"
//	log.Info("loaded config", "path", p)
//	log.Error("failed to save", "error", err)
//
// All log output is written to stderr so it does not interfere with the
// terminal UI rendered on stdout.
package logging

import (
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	// initLogger ensures the base logger is created exactly once across all
	// goroutines, even if multiple components call New concurrently.
	initLogger sync.Once

	// baseLogger is the singleton logger instance shared by all components.
	// Component-specific loggers are derived from this via With().
	baseLogger *slog.Logger
)

// New returns a structured logger scoped to the given component name.
//
// The component name is added as a "component" attribute to every log entry
// produced by the returned logger, making it easy to filter logs by subsystem
// (e.g. "app", "config", "search").
//
// If component is empty, the base logger is returned without any additional
// attributes. The underlying base logger is lazily initialized on the first
// call and reused for all subsequent calls.
func New(component string) *slog.Logger {
	initLogger.Do(func() {
		baseLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: parseLevel(os.Getenv("CLI_NOTES_LOG_LEVEL")),
		}))
	})
	if component == "" {
		return baseLogger
	}
	return baseLogger.With("component", component)
}

// parseLevel converts a human-readable log level string to a [slog.Level].
//
// Recognized values (case-insensitive, whitespace-trimmed):
//   - "debug"           → slog.LevelDebug
//   - "warn", "warning" → slog.LevelWarn
//   - "error"           → slog.LevelError
//   - anything else     → slog.LevelInfo (the default)
func parseLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
