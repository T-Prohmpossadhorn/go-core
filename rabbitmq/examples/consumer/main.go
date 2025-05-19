package main

import (
	"context"
	"fmt"
	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/T-Prohmpossadhorn/go-core/rabbitmq"
)

func main() {
	logger.Init()
	defer logger.Sync()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	rmq, _ := rabbitmq.New(cfg)
	defer rmq.Close()

	msgs, _ := rmq.Consume(context.Background(), "tasks")
	for msg := range msgs {
		fmt.Println(string(msg))
	}
}
