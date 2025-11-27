package pkg

import (
	"log/slog"
	"os"
)

func CustomSlog(service string) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		// AddSource: true,
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// time -> timestamp
			if a.Key == slog.TimeKey {
				return slog.String("timestamp", a.Value.Time().Format("2006-01-02T15:04:05Z07:00"))
			}

			// msg -> message
			if a.Key == slog.MessageKey {
				return slog.String("message", a.Value.String())
			}

			// level -> level (оставляем как есть)
			return a
		},
	})
	logger := slog.New(handler)
	host, err := os.Hostname()
	if err != nil {
		logger.Error("cant get host name", "error", err)
		os.Exit(1)
	}
	return logger.With("host", host, "service", service)
}
