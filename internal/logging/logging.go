package logging

import (
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	initLogger sync.Once
	baseLogger *slog.Logger
)

// New returns a shared logger scoped to a component name.
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
