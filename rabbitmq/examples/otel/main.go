package main

import (
	"context"
	"os"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/T-Prohmpossadhorn/go-core/otel"
	"github.com/T-Prohmpossadhorn/go-core/rabbitmq"
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

	rmq, _ := rabbitmq.New(cfg)
	defer rmq.Close()

	ctx, span := otel.StartSpan(context.Background(), "rabbitmq-example", "publish")
	_ = rmq.Publish(ctx, "tasks", []byte("hello"))
	span.End()

	msgs, _ := rmq.Consume(context.Background(), "tasks")
	msg := <-msgs
	logger.InfoContext(ctx, "consumed", logger.String("msg", string(msg)))
}
