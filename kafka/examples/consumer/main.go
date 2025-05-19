package main

import (
	"context"
	"fmt"
	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/kafka"
	"github.com/T-Prohmpossadhorn/go-core/logger"
)

func main() {
	logger.Init()
	defer logger.Sync()

	cfg, _ := config.New(config.WithDefault(map[string]interface{}{}))
	k, _ := kafka.New(cfg)
	defer k.Close()

	msgs, _ := k.Consume(context.Background(), "tasks")
	for msg := range msgs {
		fmt.Println(string(msg))
	}
}
