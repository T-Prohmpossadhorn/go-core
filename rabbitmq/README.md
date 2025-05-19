# rabbitmq Package

The `rabbitmq` package is a thin wrapper around the official [amqp091-go](https://github.com/rabbitmq/amqp091-go) client. It exposes simple APIs for publishing and consuming messages while integrating with the `config`, `logger`, and `otel` packages from this monorepo.

## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
  - [Basic Publishing](#basic-publishing)
  - [Basic Consuming](#basic-consuming)
  - [Tracing with OpenTelemetry](#tracing-with-opentelemetry)
- [Configuration](#configuration)
- [Examples](#examples)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Features
- **RabbitMQ Client**: Publishes and consumes messages using the real RabbitMQ protocol.
- **Config Integration**: Uses the `config` package to load settings such as `rabbitmq_url` and `otel_enabled`.
- **Structured Logging**: Leverages the `logger` package for contextual logs that include trace information.
- **OpenTelemetry Support**: When `otel_enabled` is `true`, operations create spans using the `otel` package.
- **Graceful Shutdown**: Close the connection with `Close()` to clean up resources.

## Installation
Install the package using `go get`:

```bash
go get github.com/T-Prohmpossadhorn/go-core/rabbitmq
```

### Dependencies
- `github.com/T-Prohmpossadhorn/go-core/config`
- `github.com/T-Prohmpossadhorn/go-core/logger`
- `github.com/T-Prohmpossadhorn/go-core/otel`

Add them to your `go.mod` if they are not already present:

```bash
go get github.com/T-Prohmpossadhorn/go-core/config
go get github.com/T-Prohmpossadhorn/go-core/logger
go get github.com/T-Prohmpossadhorn/go-core/otel
```

## Usage
The package exposes `New`, `Publish`, `Consume`, and `Close` functions. Below are common scenarios.

### Basic Publishing
Create a queue and publish a message:

```go
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

    _ = rmq.Publish(context.Background(), "tasks", []byte("hello"))
}
```

### Basic Consuming
Consume messages from a queue:

```go
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

    cfg, _ := config.New()
    rmq, _ := rabbitmq.New(cfg)
    defer rmq.Close()

    msgs, _ := rmq.Consume(context.Background(), "tasks")
    for msg := range msgs {
        fmt.Println(string(msg))
    }
}
```

### Tracing with OpenTelemetry
Enable tracing by setting `otel_enabled` to `true` and initializing the `otel` package:

```go
cfg, _ := config.New(config.WithDefault(map[string]interface{}{
    "otel_enabled": true,
}))

// Use a mock exporter for local testing
os.Setenv("OTEL_TEST_MOCK_EXPORTER", "true")
defer os.Unsetenv("OTEL_TEST_MOCK_EXPORTER")
_ = otel.Init(cfg)
defer otel.Shutdown(context.Background())

rmq, _ := rabbitmq.New(cfg)
ctx := context.Background()
_ = rmq.Publish(ctx, "tasks", []byte("traced message"))
```

Logs produced by `Publish` and `Consume` will include `trace_id` and `span_id` fields when tracing is enabled.

## Configuration
| Key            | Type   | Default                                       |
| -------------- | ------ | --------------------------------------------- |
| `rabbitmq_url` | string | `amqp://guest:guest@localhost:5672/`           |
| `otel_enabled` | bool   | `false`                                       |

Configuration can be supplied via a YAML/JSON file or environment variables using the `config` package. Example environment variables:

```bash
export CONFIG_RABBITMQ_URL=amqp://guest:guest@localhost:5672/
export CONFIG_OTEL_ENABLED=true
```

## Examples
Run the publisher and consumer examples located in `rabbitmq/examples`:

```bash
go run ./rabbitmq/examples/publisher
go run ./rabbitmq/examples/consumer
```

## Testing
Execute unit tests with coverage:

```bash
cd rabbitmq
go test -v -cover
```

Tests verify publishing, consuming, and tracing behavior using the mock OpenTelemetry exporter. The package has small, fast-running tests that avoid network access.

## Troubleshooting
- **No Traces in Logs**: Ensure `otel_enabled` is set to `true` and `otel.Init` has been called.
- **Context Cancellation**: Publishing or consuming operations return an error if the provided context is canceled.
- **Queue Not Found**: Queues are created on demand when publishing or consuming; no additional setup is required.

## Contributing
Contributions are welcome! Please open issues or pull requests on GitHub. Run `go test ./...` and `gofmt` before submitting changes.

## License
MIT License. See the `LICENSE` file for details.
