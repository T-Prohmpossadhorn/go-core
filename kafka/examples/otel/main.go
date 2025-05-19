package main

import (
	"context"
	"os"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/kafka"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/T-Prohmpossadhorn/go-core/otel"
)

func main() {
	logger.Init()
	defer logger.Sync()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{
		"otel_enabled": true,
	}))

	// Use mock exporter for demonstration
	os.Setenv("OTEL_TEST_MOCK_EXPORTER", "true")
	defer os.Unsetenv("OTEL_TEST_MOCK_EXPORTER")
	otel.Init(cfg)
	defer otel.Shutdown(context.Background())

	k, _ := kafka.New(cfg)
	defer k.Close()

	ctx, span := otel.StartSpan(context.Background(), "kafka-example", "publish")
	_ = k.Publish(ctx, "tasks", []byte("hello"))
	span.End()

	msgs, _ := k.Consume(context.Background(), "tasks")
	msg := <-msgs
	logger.InfoContext(ctx, "consumed", logger.String("msg", string(msg)))
}
