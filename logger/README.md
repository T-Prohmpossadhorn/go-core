# Logger Package

The `logger` package is a lightweight, high-performance, and thread-safe logging solution for Go applications, built on top of the Zap logging library (v1.26.0). It provides structured logging with support for console and file output, JSON and Zap console formats, a variety of field types, and integration with OpenTelemetry (v1.29.0) for trace-aware logging. Designed for simplicity and efficiency, the package is ideal for applications requiring robust logging with minimal configuration.

## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
  - [Basic Logging (Console, JSON)](#basic-logging-console-json)
  - [File Output with Zap Console Format](#file-output-with-zap-console-format)
  - [Context-Aware Logging with OpenTelemetry](#context-aware-logging-with-opentelemetry)
  - [Advanced Configuration](#advanced-configuration)
- [Configuration](#configuration)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Features
- **High-Performance Logging**: Built on Zap v1.26.0, leveraging its efficient logging pipeline for minimal overhead.
- **Log Levels**: Supports `debug`, `info`, `warn`, `error`, and `fatal` (fatal exits the program).
- **Output Options**: Logs to console or file, with JSON or Zap console formats.
- **Structured Logging**: Supports field types: `string`, `int`, `float`, `bool`, `error`, and `any` (for arbitrary data like slices or structs).
- **Context Support**: Offers both context-aware (`InfoContext`) and non-context-aware (`Info`) logging functions.
- **OpenTelemetry Integration**: Automatically includes `trace_id` and `span_id` from the context for trace-aware logging using OpenTelemetry v1.29.0.
- **Thread-Safety**: Ensures safe concurrent access using `sync.RWMutex`.
- **Performance Optimizations**: Minimizes allocations and contention with Zap’s encoders and efficient buffer management.
- **Comprehensive Testing**: Includes tests for all log levels, field types, and output combinations.

## Installation
To install the `logger` package, use `go get`:

```bash
go get github.com/your-org/logger
```

### Dependencies
The package requires the following dependencies, which will be installed automatically with `go get`:

- `go.uber.org/zap@v1.26.0`: Core logging library.
- `go.opentelemetry.io/otel@v1.29.0`: OpenTelemetry for trace integration.
- `github.com/stretchr/testify@v1.8.4`: Testing framework (for tests only).

Add them to your `.LINE_ORDERgo.mod` explicitly if needed:

```bash
go get go.uber.org/zap@v1.26.0
go get go.opentelemetry.io/otel@v1.29.0
go get github.com/stretchr/testify@v1.8.4
```

### Go Version
The package is compatible with Go 1.24.2 or later. Ensure your Go version meets this requirement:

```bash
go version
```

## Usage
The `logger` package is designed for ease of use, with a simple API for initializing, configuring, and logging messages. Below are examples demonstrating various use cases.

### Basic Logging (Console, JSON)
Initialize the logger with default settings (info level, console output, JSON format) and log a message with structured fields:

```go
package main

import (
    "github.com/your-org/logger"
)

func main() {
    if err := logger.Init(); err != nil {
        panic("Failed to initialize logger: " + err.Error())
    }
    defer logger.Sync()

    logger.Info("Application started",
        logger.String("app", "example"),
        logger.Int("version", 1),
        logger.Float("uptime", 3.14),
        logger.Bool("active", true),
        logger.ErrField(errors.New("initialization error")),
        logger.Any("metadata", []string{"x", "y"}),
    )
}
```

**Output (JSON)**:
```json
{"level":"info","ts":"2025-05-01T12:00:00.000Z","caller":"main.go:12","msg":"Application started","app":"example","version":1,"uptime":3.14,"active":true,"error":"initialization error","metadata":["x","y"]}
```

### File Output with Zap Console Format
Configure the logger to write to a file in Zap’s console format (human-readable, tab-separated):

```go
package main

import (
    "context"
    "github.com/your-org/logger"
)

func main() {
    cfg := logger.LoggerConfig{
        Level:      "info",
        Output:     "file",
        FilePath:   "app.log",
        JSONFormat: false,
    }
    if err := logger.InitWithConfig(cfg); err != nil {
        panic("Failed to initialize logger: " + err.Error())
    }
    defer logger.Sync()

    ctx := context.Background()
    logger.InfoContext(ctx, "Application started",
        logger.String("app", "example"),
        logger.Int("version", 1),
    )
}
```

**Output (app.log, Zap console format)**:
```
2025-05-01T12:00:00.000Z	info	main.go:15	Application started	{"app": "example", "version": 1}
```

### Context-Aware Logging with OpenTelemetry
Use context-aware logging to include OpenTelemetry trace fields (`trace_id`, `span_id`):

```go
package main

import (
    "context"
    "github.com/your-org/logger"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    if err := logger.Init(); err != nil {
        panic("Failed to initialize logger: " + err.Error())
    }
    defer logger.Sync()

    // Mock OpenTelemetry tracer
    tracer := trace.NewNoopTracerProvider().Tracer("example")
    ctx, span := tracer.Start(context.Background(), "example-span")
    defer span.End()

    logger.InfoContext(ctx, "Processing request",
        logger.String("request_id", "abc123"),
        logger.Any("params", map[string]string{"key": "value"}),
    )
}
```

**Output (JSON)**:
```json
{"level":"info","ts":"2025-05-01T12:00:00.000Z","caller":"main.go:20","msg":"Processing request","request_id":"abc123","params":{"key":"value"},"trace_id":"00000000000000000000000000000000","span_id":"0000000000000000"}
```

### Advanced Configuration
Customize the logger with specific log levels, output destinations, and formats:

```go
package main

import (
    "github.com/your-org/logger"
)

func main() {
    cfg := logger.LoggerConfig{
        Level:      "debug", // Enable debug and higher levels
        Output:     "console",
        JSONFormat: true,
    }
    if err := logger.InitWithConfig(cfg); err != nil {
        panic("Failed to initialize logger: " + err.Error())
    }
    defer logger.Sync()

    logger.Debug("Debugging application",
        logger.String("component", "server"),
        logger.Int("port", 8080),
    )
}
```

**Output (JSON)**:
```json
{"level":"debug","ts":"2025-05-01T12:00:00.000Z","caller":"main.go:15","msg":"Debugging application","component":"server","port":8080}
```

## Configuration
The `logger` package is configured via the `LoggerConfig` struct:

```go
type LoggerConfig struct {
    Level      string // Log level: "debug", "info", "warn", "error", "fatal"
    Output     string // Output destination: "console" or "file"
    FilePath   string // File path for file output (required if Output="file")
    JSONFormat bool   // Output format: true for JSON, false for Zap console format
}
```

### Configuration Options
- **Level**:
  - `debug`: Logs all messages (debug and above).
  - `info`: Logs info, warn, error, and fatal messages.
  - `warn`: Logs warn, error, and fatal messages.
  - `error`: Logs error and fatal messages.
  - `fatal`: Logs only fatal messages (exits program).
  - Default: `info`.

- **Output**:
  - `console`: Logs to standard output (`os.Stdout`).
  - `file`: Logs to a specified file (requires `FilePath`).
  - Default: `console`.

- **FilePath**:
  - Path to the log file (e.g., `app.log`).
  - Required if `Output="file"`.
  - Ensure write permissions (e.g., `chmod 666 app.log`).

- **JSONFormat**:
  - `true`: Outputs logs in JSON format (e.g., `{"level":"info","msg":"Test"}`).
  - `false`: Outputs logs in Zap’s console format (e.g., `2025-05-01T12:00:00.000Z INFO Test message {"key": "value"}`).
  - Default: `true`.

## Testing
The package includes comprehensive tests to validate all log levels, field types, output destinations, and formats.

### Running Tests
Run tests with verbose output to see log output:

```bash
go test -v ./logger
```

### Expected Output
- **JSON Format** (console/file):
  ```json
  {"level":"info","ts":"2025-05-01T12:00:00.000Z","caller":"logger_test.go:123","msg":"Test message","string_field":"value","int_field":42,"float_field":3.14,"bool_field":true,"error":"test error","any_field":["x","y"],"trace_id":"...","span_id":"..."}
  ```
- **Zap Console Format** (console/file):
  ```
  2025-05-01T12:00:00.000Z	info	logger_test.go:123	Test message	{"string_field": "value", "int_field": 42, "float_field": 3.14, "bool_field": true, "error": "test error", "any_field": ["x", "y"], "trace_id": "...", "span_id": "..."}
  ```

Tests cover:
- All log levels (`debug`, `info`, `warn`, `error`).
- All field types (`string`, `int`, `float`, `bool`, `error`, `any`).
- Console and file output.
- JSON and Zap console formats.
- OpenTelemetry trace fields.

## Troubleshooting
### Compilation Errors
If you encounter compilation errors (e.g., related to `go.uber.org/zap` or `go.opentelemetry.io/otel`):
- **Verify Dependencies**:
  ```bash
  go list -m go.uber.org/zap
  go list -m go.opentelemetry.io/otel
  ```
  Ensure `go.uber.org/zap@v1.26.0` and `go.opentelemetry.io/otel@v1.29.0` are used.
- **Check Dependency Conflicts**:
  ```bash
  go mod graph | grep zap
  go mod graph | grep opentelemetry
  ```
  Look for conflicting versions.
- **Clear Module Cache**:
  ```bash
  go clean -modcache
  go mod tidy
  ```
- **Share Error Output**: Provide the exact error message to diagnose the issue.

### Logger Not Initialized
If you see `logger not initialized` errors:
- Ensure `logger.Init()` or `logger.InitWithConfig()` is called before logging.
- Example:
  ```go
  if err := logger.Init(); err != nil {
      panic(err)
  }
  ```

### File Output Issues
If file output fails:
- Verify the `FilePath` is valid and writable (e.g., `chmod 666 app.log`).
- Check for errors during initialization:
  ```go
  cfg := logger.LoggerConfig{Output: "file", FilePath: "app.log"}
  if err := logger.InitWithConfig(cfg); err != nil {
      fmt.Println("Error:", err)
  }
  ```

### OpenTelemetry Fields Missing
If `trace_id` or `span_id` are not included:
- Ensure the context contains a valid OpenTelemetry span:
  ```go
  ctx, span := tracer.Start(context.Background(), "example")
  logger.InfoContext(ctx, "Message")
  span.End()
  ```

## Contributing
Contributions are welcome! To contribute:
1. **Fork the Repository**: Create a fork at `github.com/your-org/logger`.
2. **Create a Branch**: Use a descriptive branch name (e.g., `feature/add-custom-field`).
3. **Make Changes**: Implement your feature or fix, ensuring tests pass.
4. **Run Tests**: Verify with `go test -v ./logger`.
5. **Submit a Pull Request**: Include a clear description of changes and reference any issues.

### Development Setup
Clone the repository and install dependencies:

```bash
git clone https://github.com/your-org/logger.git
cd logger
go mod tidy
```

### Code Style
- Follow Go conventions (use `gofmt`, `golint`).
- Write clear, concise code with comments for complex logic.
- Ensure tests cover new functionality.

## License
This project is licensed under the MIT License. See the `LICENSE` file for details.