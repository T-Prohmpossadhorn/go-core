# rabbitmq Package

The `rabbitmq` package provides a lightweight, in-memory message queue with an API inspired by RabbitMQ. It mirrors the style of the `httpc` package, integrating with `config` and `logger` for easy configuration and structured logging. This implementation does **not** connect to an actual RabbitMQ server; it is a simple, thread-safe queue useful for local testing or demonstration purposes.

## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
  - [Publishing Messages](#publishing-messages)
  - [Consuming Messages](#consuming-messages)
- [Configuration](#configuration)
- [Examples](#examples)
- [License](#license)

## Features
- **In-Memory Queues**: Send and receive messages without an external broker.
- **Config Integration**: Uses the `config` package to load settings such as `rabbitmq_url` and `otel_enabled`.
- **Structured Logging**: Leverages the `logger` package for contextual logs.
- **Optional Tracing**: Placeholder field for OpenTelemetry support.
- **Graceful Shutdown**: Close all queues with `Close()`.

## Installation
```bash
go get github.com/T-Prohmpossadhorn/go-core/rabbitmq
```

## Usage
### Publishing Messages
```go
cfg, _ := config.New(config.WithDefault(map[string]interface{}{
    "rabbitmq_url": "amqp://guest:guest@localhost:5672/",
}))
rmq, _ := rabbitmq.New(cfg)
ctx := context.Background()
if err := rmq.Publish(ctx, "tasks", []byte("hello")); err != nil {
    panic(err)
}
```

### Consuming Messages
```go
ctx := context.Background()
msgs, _ := rmq.Consume(ctx, "tasks")
for msg := range msgs {
    fmt.Println(string(msg))
}
```

## Configuration
| Key            | Type   | Default                                       |
|----------------|-------|-----------------------------------------------|
| `rabbitmq_url` | string | `amqp://guest:guest@localhost:5672/`           |
| `otel_enabled` | bool   | `false`                                       |

## Examples
Run the publisher and consumer examples:
```bash
go run ./rabbitmq/examples/publisher
go run ./rabbitmq/examples/consumer
```

## License
MIT License. See `LICENSE` file.
