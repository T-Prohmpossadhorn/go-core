# kafka Package

The `kafka` package provides a lightweight, in-memory message queue with an API inspired by Kafka. It mirrors the style of the `rabbitmq` package, integrating with `config` and `logger` for easy configuration and structured logging. This implementation does **not** connect to an actual Kafka broker; it is a simple, thread-safe queue useful for local testing or demonstration purposes.

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
- **In-Memory Topics**: Send and receive messages without an external broker.
- **Config Integration**: Uses the `config` package to load settings such as `kafka_brokers`, `kafka_topic`, and `otel_enabled`.
- **Structured Logging**: Leverages the `logger` package for contextual logs.
- **Optional Tracing**: Placeholder field for OpenTelemetry support.
- **Graceful Shutdown**: Close all topics with `Close()`.

## Installation
```bash
go get github.com/T-Prohmpossadhorn/go-core/kafka
```

## Usage
### Publishing Messages
```go
cfg, _ := config.New(config.WithDefault(map[string]interface{}{
    "kafka_brokers": "localhost:9092",
}))
k, _ := kafka.New(cfg)
ctx := context.Background()
if err := k.Publish(ctx, "tasks", []byte("hello")); err != nil {
    panic(err)
}
```

### Consuming Messages
```go
ctx := context.Background()
msgs, _ := k.Consume(ctx, "tasks")
for msg := range msgs {
    fmt.Println(string(msg))
}
```

## Configuration
| Key            | Type   | Default          |
|----------------|-------|------------------|
| `kafka_brokers`| string | `localhost:9092` |
| `kafka_topic`  | string | `default`        |
| `otel_enabled` | bool   | `false`          |

## Examples
Run the producer and consumer examples:
```bash
go run ./kafka/examples/producer
go run ./kafka/examples/consumer
```

## License
MIT License. See `LICENSE` file.
