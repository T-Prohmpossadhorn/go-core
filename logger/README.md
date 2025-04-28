# Logger

A flexible, extensible logging package for Go applications with OpenTelemetry integration.

## Features

- Simple, consistent API for logging
- Support for [zap](https://github.com/uber-go/zap) and [logrus](https://github.com/sirupsen/logrus) logging backends
- OpenTelemetry integration via context
- Structured logging with strongly typed fields
- Multiple log levels (Debug, Info, Warn, Error, Fatal)
- Easy configuration
- Support for global logger instance
- Easily extendable to support other logging libraries

## Installation

```bash
go get github.com/T-Prohmpossadhorn/logger
```

## Basic Usage

```go
package main

import (
    "context"
    
    "github.com/T-Prohmpossadhorn/logger"
)

func main() {
    // Create a logger with default configuration (uses zap)
    log := logger.New(logger.DefaultConfig())
    
    ctx := context.Background()
    
    // Log messages at different levels
    log.Debug(ctx, "This is a debug message")
    log.Info(ctx, "This is an info message")
    log.Warn(ctx, "This is a warning message")
    log.Error(ctx, "This is an error message")
    log.Fatal(ctx, "This is a fatal message") // This will exit the program
}
```

## Configuration

You can configure the logger using the `Config` struct:

```go
config := logger.Config{
    Type:        logger.ZapLogger, // or logger.LogrusLogger
    Level:       logger.Info,
    Output:      os.Stdout,
    ServiceName: "my-service",
}

log := logger.New(config)
```

## Structured Logging

This package supports structured logging with strongly typed fields:

```go
log.Info(ctx, "User logged in",
    logger.String("user_id", "12345"),
    logger.Int("login_count", 5),
    logger.Bool("is_admin", false),
    logger.Error(err), // For error fields
)
```

## OpenTelemetry Integration

The logger will automatically extract trace and span IDs from the context if available:

```go
// In a function where ctx contains OpenTelemetry span
log.Info(ctx, "Processing request")

// Output will include trace_id and span_id fields
```

### Complete OpenTelemetry Example

Here's a full example of integrating the logger with OpenTelemetry:

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/T-Prohmpossadhorn/logger"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
    "go.opentelemetry.io/otel/trace"
)

func main() {
    // Initialize OpenTelemetry
    tp, err := initTracer()
    if err != nil {
        log.Fatal(err)
    }
    defer func() {
        if err := tp.Shutdown(context.Background()); err != nil {
            log.Printf("Error shutting down tracer provider: %v", err)
        }
    }()

    // Initialize logger
    logConfig := logger.DefaultConfig()
    logConfig.ServiceName = "example-service"
    log := logger.New(logConfig)

    // Create a span
    tracer := otel.Tracer("example-tracer")
    ctx, span := tracer.Start(context.Background(), "example-operation")
    defer span.End()

    // Log with the span context
    // The logger will automatically extract and add trace_id and span_id fields
    log.Info(ctx, "Operation started")

    // Add more span events
    span.AddEvent("Processing item")

    // Log within the span context
    processItem(ctx, log)

    log.Info(ctx, "Operation completed")
}

func processItem(ctx context.Context, log logger.Logger) {
    // Create a child span
    tracer := otel.Tracer("example-tracer")
    ctx, span := tracer.Start(ctx, "process-item")
    defer span.End()

    // The log will include the child span's ID
    log.Debug(ctx, "Processing item details", 
        logger.String("item_id", "item-123"),
        logger.Int("priority", 1))

    // Simulate error
    err := simulateWork()
    if err != nil {
        // Record the error in the span
        span.RecordError(err)
        // Log the error with the same context
        log.Error(ctx, "Failed to process item", logger.Error(err))
    }
}

func simulateWork() error {
    // Simulate work
    return nil // or return an error
}

func initTracer() (*sdktrace.TracerProvider, error) {
    // Create stdout exporter
    exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
    if err != nil {
        return nil, err
    }

    // Create trace provider
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithSampler(sdktrace.AlwaysSample()),
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("example-service"),
        )),
    )
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

This example demonstrates:
- Initializing OpenTelemetry with a tracer provider
- Creating spans and child spans
- Using the logger within span contexts
- How trace and span IDs are automatically included in log entries
- Recording errors in both logs and spans

## Creating Loggers with Preset Fields

You can create loggers with preset fields that will be included in all log entries:

```go
// Create a logger with user context
userLogger := log.WithFields(
    logger.String("user_id", "12345"),
    logger.String("username", "johndoe"),
)

// All these logs will include the user_id and username fields
userLogger.Info(ctx, "User viewed dashboard")
userLogger.Warn(ctx, "Failed login attempt")
```

## Global Logger

For convenience, you can set and use a global logger instance:

```go
// Set the global logger
logger.SetLogger(logger.New(config))

// Use global functions
logger.Info(ctx, "Using global logger")
logger.Error(ctx, "Something went wrong", logger.Error(err))

// Or get the global logger instance
log := logger.GetLogger()
log.Info(ctx, "Another way to use the global logger")
```

## Testing

The package includes a mock logger for testing:

```go
mockLogger := logger.NewMockLogger()

// Use the mock logger in your code
mockLogger.Info(ctx, "Test message")

// Check what was logged
logs := mockLogger.GetInfoLogs()
assert.Len(t, logs, 1)
assert.Equal(t, "Test message", logs[0].Msg)
```

## Switching Logging Libraries

You can easily switch between logging backends:

```go
// Start with zap
zapConfig := logger.DefaultConfig()
zapConfig.Type = logger.ZapLogger
log := logger.New(zapConfig)

// Switch to logrus
logrusConfig := logger.DefaultConfig()
logrusConfig.Type = logger.LogrusLogger
log = logger.New(logrusConfig)
```

## Adding Your Own Logging Backend

To add support for another logging library:

1. Create a new file for your implementation
2. Create a struct that implements the `Logger` interface
3. Add a new constant to `LoggerType` in `main.go`
4. Update the `New` function to support your logger type

## License

MIT

## Author

[T-Prohmpossadhorn](https://github.com/T-Prohmpossadhorn)