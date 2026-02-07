package app

import (
	"log/slog"

	"github.com/treykane/cli-notes/internal/logging"
)

// appLog is the package-level structured logger for the app package.
//
// It is pre-configured with the component tag "app" so that all log entries
// produced by this package are easily identifiable in log output. This logger
// is used throughout the app package for operational logging — warnings about
// non-critical failures (e.g. draft save errors, git status errors), errors
// during rendering or search indexing, and informational messages about
// significant state changes.
//
// The log level is controlled by the CLI_NOTES_LOG_LEVEL environment variable
// (see the logging package for details). All output is written to stderr so
// it does not interfere with the Bubble Tea terminal UI on stdout.
var appLog = logging.New("app")

// setStatusError updates the status bar with a user-facing error message and
// simultaneously logs a structured error entry with full context.
//
// This is the standard way to handle errors that should be both visible to the
// user (via the footer status bar) and recorded in logs (for debugging). The
// status parameter is displayed verbatim in the UI, while the err and any
// additional key-value attrs are included only in the log entry.
//
// Usage:
//
//	m.setStatusError("Error saving note", err, "path", notePath)
//	m.setStatusError("Clipboard copy failed", err)
//
// The variadic attrs parameter accepts slog-style key-value pairs that provide
// additional context in the log entry (e.g. file paths, operation names,
// sequence numbers). The error itself is always logged under the "error" key.
func (m *Model) setStatusError(status string, err error, attrs ...any) {
	m.status = status
	fields := make([]any, 0, len(attrs)+2)
	fields = append(fields, slog.Any("error", err))
	fields = append(fields, attrs...)
	appLog.Error(status, fields...)
}
