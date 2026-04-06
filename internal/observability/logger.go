package observability

import (
	"log/slog"
	"os"
	"strings"

	"opencrab/internal/config"
)

func NewLogger(cfg config.Config) *slog.Logger {
	level := slog.LevelInfo
	if strings.EqualFold(cfg.App.Environment, "development") {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler).With(
		slog.String("app", cfg.App.Name),
		slog.String("env", cfg.App.Environment),
	)
}
