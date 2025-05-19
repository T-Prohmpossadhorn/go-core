# go-core

`go-core` is a modular Go library that provides reusable components for configuration management, structured logging, and OpenTelemetry integration. It is designed for developers who want to build scalable and maintainable Go applications with minimal boilerplate.

## Modules

### config

A flexible configuration loader with support for YAML, JSON, environment variables, and default values.

**Example:**

```go
import "github.com/T-Prohmpossadhorn/go-core/config"

cfg := config.Load("config.yaml")
fmt.Println(cfg.GetString("server.port"))
```

---

### logger

A structured logger with configurable log levels, contextual logging, and support for console and file outputs.

**Example:**

```go
import "github.com/T-Prohmpossadhorn/go-core/logger"

if err := logger.Init(); err != nil {
    panic(err)
}
defer logger.Sync()

logger.Info("Application started")
```

---

### otel

Helpers for integrating OpenTelemetry into your application with support for tracing and context propagation.

**Example:**

```go
import "github.com/T-Prohmpossadhorn/go-core/otel"

ctx, span := otel.StartSpan(context.Background(), "my-service", "my-operation")
defer span.End()
// ... perform work ...
```

### rabbitmq

An in-memory queue with a RabbitMQ-style API.

**Example:**

```go
import (
    "context"
    "github.com/T-Prohmpossadhorn/go-core/config"
    "github.com/T-Prohmpossadhorn/go-core/rabbitmq"
)

cfg, _ := config.New()
rmq, _ := rabbitmq.New(cfg)
defer rmq.Close()
rmq.Publish(context.Background(), "q", []byte("hello"))
```

---

## Testing

Each module includes unit tests. To run all tests:

```bash
go test ./...
```

---

## Examples

Each module includes example programs under its respective `examples/` folder.

To run an example:

```bash
go run ./logger/examples/console/main.go
```

---

## Project Structure

```
go-core/
├── config/         # Configuration loader
├── logger/         # Structured logging
├── otel/           # OpenTelemetry support
├── httpc/          # HTTP server and client
├── rabbitmq/       # In-memory message queue
├── go.mod
├── go.sum
└── README.md
```

---

## Requirements

- Go 1.24.2 or higher

---

## License

MIT License — see `LICENSE` file for details.

## Author

T-Prohmpossadhorn


---

## Module Documentation

- [config/README.md](https://github.com/T-Prohmpossadhorn/go-core/blob/main/config/README.md)
- [logger/README.md](https://github.com/T-Prohmpossadhorn/go-core/blob/main/logger/README.md)
- [otel/README.md](https://github.com/T-Prohmpossadhorn/go-core/blob/main/otel/README.md)
- [httpc/README.md](https://github.com/T-Prohmpossadhorn/go-core/blob/main/httpc/README.md)
- [rabbitmq/README.md](https://github.com/T-Prohmpossadhorn/go-core/blob/main/rabbitmq/README.md)
