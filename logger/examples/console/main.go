package main

import (
	"context"

	"github.com/T-Prohmpossadhorn/go-core/logger"
)

func main() {
	// Initialize logger with file output
	cfg := logger.LoggerConfig{
		Level:      "debug",
		Output:     "console",
		FilePath:   "",
		JSONFormat: false,
	}
	if err := logger.InitWithConfig(cfg); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	ctx := context.Background()
	logger.InfoContext(ctx, "Application started",
		logger.String("app", "example"),
		logger.Int("version", 1),
	)

	logger.DebugContext(ctx, "Debug message", logger.String("debug_field", "debug"))
	logger.WarnContext(ctx, "Warn message", logger.String("warn_field", "warn"))
	logger.ErrorContext(ctx, "Error message", logger.String("error_field", "error"))
	logger.FatalContext(ctx, "Fatal message", logger.String("fatal_field", "fatal"))
}
