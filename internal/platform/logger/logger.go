package logger

import (
	"log/slog"
	"os"
)

func New(env string) *slog.Logger {
	opts := &slog.HandlerOptions{
		ReplaceAttr: replaceTime,
	}

	var handler slog.Handler
	switch env {
	case "development", "dev":
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		opts.Level = slog.LevelInfo
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func replaceTime(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.String("time", a.Value.Time().UTC().Format("2006-01-02T15:04:05.000Z"))
	}
	return a
}
