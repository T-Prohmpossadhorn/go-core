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

otelTracer := otel.NewTracer("my-service")
ctx, span := otelTracer.Start(context.Background(), "my-operation")
// ... perform work ...
span.End()
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
