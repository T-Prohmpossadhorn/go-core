package main

import (
	"context"

	"github.com/T-Prohmpossadhorn/go-core/logger"
)

func main() {
	cfg := logger.LoggerConfig{
		Level:      "info",
		Output:     "file",
		FilePath:   "app.log",
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
}
