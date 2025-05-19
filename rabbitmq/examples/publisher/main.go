package main

import (
	"context"
	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/T-Prohmpossadhorn/go-core/rabbitmq"
)

func main() {
	logger.Init()
	defer logger.Sync()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{
		"rabbitmq_url": "amqp://guest:guest@localhost:5672/",
	}))
	rmq, _ := rabbitmq.New(cfg)
	defer rmq.Close()

	rmq.Publish(context.Background(), "tasks", []byte("hello"))
}
