package app

import (
	"log/slog"

	"github.com/treykane/cli-notes/internal/logging"
)

var appLog = logging.New("app")

func (m *Model) setStatusError(status string, err error, attrs ...any) {
	m.status = status
	fields := make([]any, 0, len(attrs)+2)
	fields = append(fields, slog.Any("error", err))
	fields = append(fields, attrs...)
	appLog.Error(status, fields...)
}
