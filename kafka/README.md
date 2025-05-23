# kafka Package

The `kafka` package wraps [kafka-go](https://github.com/segmentio/kafka-go) to communicate with a real Kafka broker. It integrates with the `config`, `logger`, and `otel` packages in this monorepo.

## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
  - [Producing Messages](#producing-messages)
  - [Consuming Messages](#consuming-messages)
  - [Tracing with OpenTelemetry](#tracing-with-opentelemetry)
- [Configuration](#configuration)
- [Examples](#examples)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Features
- **Kafka Client**: Sends and receives messages using real Kafka brokers.
- **Config Integration**: Load `kafka_brokers`, `kafka_topic`, and `otel_enabled` using the `config` package.
- **Structured Logging**: `logger` provides context-aware logs.
- **OpenTelemetry Support**: When enabled, operations create spans with the `otel` package.
- **Graceful Shutdown**: Close all writers and readers with `Close()`.

## Installation
Install the package:

```bash
go get github.com/T-Prohmpossadhorn/go-core/kafka
```

### Dependencies
- `github.com/T-Prohmpossadhorn/go-core/config`
- `github.com/T-Prohmpossadhorn/go-core/logger`
- `github.com/T-Prohmpossadhorn/go-core/otel`

Add them to your project as needed:

```bash
go get github.com/T-Prohmpossadhorn/go-core/config
go get github.com/T-Prohmpossadhorn/go-core/logger
go get github.com/T-Prohmpossadhorn/go-core/otel
```

## Usage
Create a new instance with `New`. You can publish messages with either `Publish`/`Consume` using raw bytes or use `PublishJSON`/`ConsumeJSON` to work with structs directly.

### Producing Messages

```go
package main

import (
    "context"

    "github.com/T-Prohmpossadhorn/go-core/config"
    "github.com/T-Prohmpossadhorn/go-core/logger"
    "github.com/T-Prohmpossadhorn/go-core/kafka"
)

func main() {
    logger.Init()
    defer logger.Sync()

    cfg, _ := config.New(config.WithDefault(map[string]interface{}{
        "kafka_brokers": "localhost:9092",
    }))
    k, _ := kafka.New(cfg)
    defer k.Close()

    type Task struct {
        Name string `json:"name"`
    }

    _ = kafka.PublishJSON(context.Background(), k, "tasks", Task{Name: "hello"})
}
```

### Consuming Messages

```go
package main

import (
    "context"
    "fmt"

    "github.com/T-Prohmpossadhorn/go-core/config"
    "github.com/T-Prohmpossadhorn/go-core/logger"
    "github.com/T-Prohmpossadhorn/go-core/kafka"
)

func main() {
    logger.Init()
    defer logger.Sync()

    cfg, _ := config.New()
    k, _ := kafka.New(cfg)
    defer k.Close()

    type Task struct {
        Name string `json:"name"`
    }

    msgs, _ := kafka.ConsumeJSON[Task](context.Background(), k, "tasks")
    for msg := range msgs {
        fmt.Println(msg.Name)
    }
}
```

### Tracing with OpenTelemetry
Enable tracing by setting `otel_enabled` to `true` and initializing the `otel` package:

```go
cfg, _ := config.New(config.WithDefault(map[string]interface{}{
    "otel_enabled": true,
}))

os.Setenv("OTEL_TEST_MOCK_EXPORTER", "true")
defer os.Unsetenv("OTEL_TEST_MOCK_EXPORTER")
_ = otel.Init(cfg)
defer otel.Shutdown(context.Background())

k, _ := kafka.New(cfg)
ctx := context.Background()
type Task struct {
    Name string `json:"name"`
}

_ = kafka.PublishJSON(ctx, k, "tasks", Task{Name: "traced"})

msgs, _ := kafka.ConsumeJSON[Task](context.Background(), k, "tasks")
msg := <-msgs
fmt.Println(msg.Name)
```

With tracing enabled, log entries include `trace_id` and `span_id` so you can correlate events across services.

## Configuration
| Key              | Type   | Default          |
| ---------------- | ------ | ---------------- |
| `kafka_brokers`  | string | `localhost:9092` |
| `kafka_topic`    | string | `default`        |
| `otel_enabled`   | bool   | `false`          |
| `kafka_enable_tls` | bool | `false`          |
| `kafka_username`   | string | ``              |
| `kafka_password`   | string | ``              |

Configuration can be loaded from files or environment variables. Example environment usage:

```bash
export CONFIG_KAFKA_BROKERS=localhost:9092
export CONFIG_OTEL_ENABLED=true
```

## Examples
Run the producer and consumer examples found in `kafka/examples`:

```bash
go run ./kafka/examples/producer
go run ./kafka/examples/consumer
```

## Testing
Run the unit tests:

```bash
cd kafka
go test -v -cover
```

Tests cover publishing, consuming, and tracing using the mock exporter to avoid external dependencies.

## Troubleshooting
- **Tracing Disabled**: Confirm `otel_enabled` is true and `otel.Init` completed successfully.
- **Context Errors**: Operations fail when the provided context is canceled.
- **Topic Not Found**: Topics are created on demand when publishing or consuming.

## Contributing
Feedback and contributions are encouraged! Open an issue or pull request on GitHub and ensure `go test ./...` passes before submission.

## License
MIT License. See the `LICENSE` file for full details.
