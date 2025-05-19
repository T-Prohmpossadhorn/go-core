package main

import (
	"context"
	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/kafka"
	"github.com/T-Prohmpossadhorn/go-core/logger"
)

func main() {
	logger.Init()
	defer logger.Sync()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{
		"kafka_brokers": "localhost:9092",
	}))
	k, _ := kafka.New(cfg)
	defer k.Close()

	k.Publish(context.Background(), "tasks", []byte("hello"))
}
