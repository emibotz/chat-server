package logging

import "log/slog"

func Error(message string, err error) {
	slog.Error(
		message,
		slog.String("error", err.Error()),
	)
}
