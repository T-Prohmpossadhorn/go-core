# httpc Package

The `httpc` package is a Gin-based HTTP server and client for Go applications, part of the `github.com/T-Prohmpossadhorn/go-core` monorepo. It provides a robust framework for building RESTful APIs with reflection-based service registration, supporting complex JSON payloads, input validation, optional OpenTelemetry tracing, and graceful shutdown. Designed for microservices, it integrates with `config` and `logger` for configuration and logging, achieving high test coverage and thread-safety.

## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
  - [Registering a Service](#registering-a-service)
  - [Sending HTTP Requests](#sending-http-requests)
  - [Healthcheck Endpoint](#healthcheck-endpoint)
  - [OpenAPI Documentation](#openapi-documentation)
  - [OpenTelemetry Integration](#opentelemetry-integration)
  - [Graceful Shutdown](#graceful-shutdown)
- [Configuration](#configuration)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Features
- **Gin-Based Server**: Uses `github.com/gin-gonic/gin@v1.10.0` for routing and middleware, supporting GET, POST, PUT, and DELETE methods with extensible endpoint registration via `ListenAndServe`.
- **HTTP Client**: Sends HTTP requests with configurable timeouts, retries, and backoff, supporting GET, POST, PUT, and DELETE with JSON payloads and string/struct responses.
- **Reflection-Based Service Registration**: Registers service methods as HTTP endpoints using `RegisterMethods`, supporting both pointer and non-pointer service types for flexibility.
- **Complex JSON Support**: Handles nested JSON payloads with strict validation using `github.com/go-playground/validator/v10@v10.26.0`, enforcing required fields, length constraints, and custom rules.
- **Healthcheck**: `/health` endpoint returning `200 OK` with `{"status":"healthy"}`.
- **Swagger Documentation**: Generates OpenAPI 3.0.3 JSON at `/api/docs/swagger.json` for registered endpoints, reflecting service methods and schemas.
- **Mandatory Integration**: Uses `config` for settings and `logger` for request logging with structured JSON output.
- **Optional Tracing**: Supports `go.opentelemetry.io/otel@v1.24.0` for request tracing when enabled, for both server and client.
- **Graceful Shutdown**: Supports graceful server shutdown via `Shutdown` method, handling active connections with a configurable timeout.
- **Thread-Safety**: Safe for concurrent requests with proper synchronization.
- **High Test Coverage**: Achieves 82.3% coverage (targeting ≥91.1%) with comprehensive unit tests covering server, client, and error cases.
- **Go 1.24.2**: Compatible with the latest Go version.

## Installation
Install the `httpc` package:

```bash
go get github.com/T-Prohmpossadhorn/go-core/httpc
```

### Dependencies
- `github.com/T-Prohmpossadhorn/go-core/config@latest`
- `github.com/T-Prohmpossadhorn/go-core/logger@latest`
- `github.com/T-Prohmpossadhorn/go-core/otel@latest` (optional)
- `github.com/cenkalti/backoff/v4@v4.3.0`
- `github.com/gin-gonic/gin@v1.10.0`
- `github.com/go-playground/validator/v10@v10.26.0`
- `github.com/google/uuid@v1.6.0`
- `go.opentelemetry.io/otel@v1.24.0`
- `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.24.0`
- `go.opentelemetry.io/otel/sdk@v1.24.0`
- `go.opentelemetry.io/otel/trace@v1.24.0`

Add to `go.mod`:

```bash
go get github.com/T-Prohmpossadhorn/go-core/config@latest
go get github.com/T-Prohmpossadhorn/go-core/logger@latest
go get github.com/T-Prohmpossadhorn/go-core/otel@latest
go get github.com/cenkalti/backoff/v4@v4.3.0
go get github.com/gin-gonic/gin@v1.10.0
go get github.com/go-playground/validator/v10@v10.26.0
go get github.com/google/uuid@v1.6.0
go get go.opentelemetry.io/otel@v1.24.0
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc@v1.24.0
go get go.opentelemetry.io/otel/sdk@v1.24.0
go get go.opentelemetry.io/otel/trace@v1.24.0
```

### Go Version
Requires Go 1.24.2 or later:

```bash
go version
```

## Usage
The `httpc` package enables building HTTP servers with reflection-based service registration and clients for sending HTTP requests. Server endpoints support JSON payloads with validation for POST/PUT/DELETE requests and query parameters for GET requests, returning JSON or string responses. The client handles JSON requests and responses with configurable retries and backoff. Both server and client support optional OpenTelemetry tracing and graceful shutdown for production readiness.

### Registering a Service
Define a service struct with a `RegisterMethods` method to specify HTTP endpoints:

```go
package main

import (
    "os"
    "os/signal"
    "reflect"
    "syscall"
    "time"

    "github.com/T-Prohmpossadhorn/go-core/config"
    "github.com/T-Prohmpossadhorn/go-core/httpc"
    "github.com/T-Prohmpossadhorn/go-core/logger"
    "github.com/T-Prohmpossadhorn/go-core/otel"
)

type User struct {
    Name    string `json:"name" validate:"required,min=1,max=50"`
    Address struct {
        City string `json:"city" validate:"required,min=1,max=50"`
    } `json:"address" validate:"required"`
}

type MyService struct{}

func (s *MyService) Hello(name string) (string, error) {
    return "Hello, " + name + "!", nil
}

func (s *MyService) Create(user User) (string, error) {
    return "Created user " + user.Name, nil
}

func (s *MyService) RegisterMethods() []httpc.MethodInfo {
    return []httpc.MethodInfo{
        {
            Name:       "Hello",
            HTTPMethod: "GET",
            InputType:  reflect.TypeOf(""),
            OutputType: reflect.TypeOf(""),
        },
        {
            Name:       "Create",
            HTTPMethod: "POST",
            InputType:  reflect.TypeOf(User{}),
            OutputType: reflect.TypeOf(""),
        },
    }
}

func main() {
    if err := logger.Init(); err != nil {
        panic("Failed to initialize logger: " + err.Error())
    }
    defer logger.Sync()

    cfg, err := config.New(config.WithDefault(map[string]interface{}{
        "otel_enabled": false,
        "port":         8080,
    }))
    if err != nil {
        panic("Failed to initialize config: " + err.Error())
    }

    if cfg.GetBool("otel_enabled") {
        if err := otel.Init(cfg); err != nil {
            panic("Failed to initialize otel: " + err.Error())
        }
        defer otel.Shutdown(context.Background())
    }

    server, err := httpc.NewServer(cfg)
    if err != nil {
        panic("Failed to initialize HTTP server: " + err.Error())
    }

    err = server.RegisterService(&MyService{}, httpc.WithPathPrefix("/api/v1"))
    if err != nil {
        panic("Failed to register service: " + err.Error())
    }

    // Start server in a goroutine
    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Error("Server failed to start", logger.ErrField(err))
            panic("Failed to start server: " + err.Error())
        }
    }()

    // Handle graceful shutdown
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
    <-sigs

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := server.Shutdown(ctx); err != nil {
        logger.Error("Server shutdown failed", logger.ErrField(err))
        panic("Server shutdown failed: " + err.Error())
    }
    logger.Info("Server shut down gracefully")
}
```

### Sending HTTP Requests
Create an `HTTPClient` to send HTTP requests:

```go
package main

import (
    "fmt"
    "github.com/T-Prohmpossadhorn/go-core/config"
    "github.com/T-Prohmpossadhorn/go-core/httpc"
    "github.com/T-Prohmpossadhorn/go-core/logger"
)

type User struct {
    Name    string `json:"name" validate:"required,min=1,max=50"`
    Address struct {
        City string `json:"city" validate:"required,min=1,max=50"`
    } `json:"address" validate:"required"`
}

func main() {
    if err := logger.Init(); err != nil {
        panic("Failed to initialize logger: " + err.Error())
    }
    defer logger.Sync()

    cfg, err := config.New(config.WithDefault(map[string]interface{}{
        "otel_enabled":           false,
        "http_client_timeout_ms": 1000,
        "http_client_max_retries": 2,
        "http_client_backoff_base_ms": 100,
        "http_client_backoff_max_ms": 1000,
        "http_client_backoff_factor": 2,
        "http_client_disable_backoff": false,
    }))
    if err != nil {
        panic("Failed to initialize config: " + err.Error())
    }

    client, err := httpc.NewHTTPClient(cfg)
    if err != nil {
        panic("Failed to initialize HTTP client: " + err.Error())
    }

    var greeting string
    err = client.Call("GET", "http://localhost:8080/api/v1/Hello?name=Alice", nil, &greeting)
    if err != nil {
        fmt.Printf("GET request failed: %v\n", err)
        return
    }
    fmt.Printf("GET response: %s\n", greeting) // Output: Hello, Alice!

    user := User{
        Name: "Bob",
        Address: struct {
            City string `json:"city" validate:"required,min=1,max=50"`
        }{
            City: "Metropolis",
        },
    }
    var createResult string
    err = client.Call("POST", "http://localhost:8080/api/v1/Create", user, &createResult)
    if err != nil {
        fmt.Printf("POST request failed: %v\n", err)
        return
    }
    fmt.Printf("POST response: %s\n", createResult) // Output: Created user Bob
}
```

Send requests using curl:

```bash
# GET
curl http://localhost:8080/api/v1/Hello?name=Alice
# Response: "Hello, Alice!"

# POST
curl -X POST http://localhost:8080/api/v1/Create -H "Content-Type: application/json" -d '{"name":"Bob","address":{"city":"Metropolis"}}'
# Response: "Created user Bob"

# Invalid POST
curl -X POST http://localhost:8080/api/v1/Create -H "Content-Type: application/json" -d '{"name":"","address":{"city":""}}'
# Response: {"error":"validation failed: Key: 'User.Name' Error:Field validation for 'Name' failed on the 'required' tag\nKey: 'User.Address.City' Error:Field validation for 'City' failed on the 'required' tag"}

# Error case (simulated server error)
curl http://localhost:8080/api/v1/GetMethod?name=error
# Response: {"error":"simulated server error"}
```

### Healthcheck Endpoint
Access the healthcheck endpoint:

```bash
curl http://localhost:8080/health
# Response: {"status":"healthy"}
```

### OpenAPI Documentation
Access the OpenAPI 3.0.3 JSON at `http://localhost:8080/api/docs/swagger.json` to explore the API. The dynamically generated documentation reflects service methods, schemas, and validation rules.

Example:
```bash
curl http://localhost:8080/api/docs/swagger.json
```

Response (abridged):
```json
{
    "openapi": "3.0.3",
    "info": {
        "title": "httpc API",
        "version": "1.0.0"
    },
    "paths": {
        "/api/v1/Hello": {
            "get": {
                "summary": "Hello",
                "operationId": "Hello",
                "parameters": [
                    {
                        "name": "name",
                        "in": "query",
                        "required": false,
                        "schema": {
                            "type": "string"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Successful response",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "400": {
                        "description": "Bad request",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "type": "object",
                                    "properties": {
                                        "error": {"type": "string"}
                                    }
                                }
                            }
                        }
                    },
                    "500": {
                        "description": "Internal server error",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "type": "object",
                                    "properties": {
                                        "error": {"type": "string"}
                                    }
                                }
                            }
                        }
                    }
                }
            }
        },
        "/api/v1/Create": {
            "post": {
                "summary": "Create",
                "operationId": "Create",
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": {
                                "type": "object",
                                "properties": {
                                    "name": {
                                        "type": "string",
                                        "minLength": 1,
                                        "maxLength": 50
                                    },
                                    "address": {
                                        "type": "object",
                                        "properties": {
                                            "city": {
                                                "type": "string",
                                                "minLength": 1,
                                                "maxLength": 50
                                            }
                                        },
                                        "required": ["city"]
                                    }
                                },
                                "required": ["name", "address"]
                            }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "Successful response",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "400": {
                        "description": "Bad request",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "type": "object",
                                    "properties": {
                                        "error": {"type": "string"}
                                    }
                                }
                            }
                        }
                    },
                    "500": {
                        "description": "Internal server error",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "type": "object",
                                    "properties": {
                                        "error": {"type": "string"}
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
```

### OpenTelemetry Integration
The `httpc` package supports OpenTelemetry tracing for both server and client when enabled via the `otel_enabled` configuration. Tracing captures request spans, including method calls, endpoints, and errors, which are exported to an OTLP collector (e.g., Jaeger, Zipkin) for distributed tracing.

#### Enabling Tracing
1. **Configure OTLP Collector**:
   Run an OTLP-compatible collector (e.g., OpenTelemetry Collector, Jaeger) to receive traces:
   ```bash
   docker run -p 4317:4317 otel/opentelemetry-collector
   ```
   Ensure the collector is accessible at `localhost:4317` or update the `otel_endpoint` configuration.

2. **Set Configuration**:
   Enable tracing in the server and client configuration:
   ```go
   cfg, err := config.New(config.WithDefault(map[string]interface{}{
       "otel_enabled":           true,
       "otel_endpoint":          "localhost:4317",
       "port":                   8080,
       "http_client_timeout_ms": 1000,
       "http_client_max_retries": 2,
   }))
   ```

3. **Initialize OpenTelemetry**:
   Before creating the server or client, initialize otel if tracing is enabled:
   ```go
   if cfg.GetBool("otel_enabled") {
       if err := otel.Init(cfg); err != nil {
           panic("Failed to initialize otel: " + err.Error())
       }
       defer otel.Shutdown(context.Background())
   }
   ```

4. **Verify Traces**:
   - Server logs include trace and span IDs for registered services and requests:
     ```
     {"level":"info","ts":"2025-05-04T13:38:12.182+0700","caller":"logger/logger.go:196","msg":"Starting RegisterService","trace_id":"5e466a7389a4862911e4d1a9136d62b7","span_id":"cf9ce1a5ce23dffd"}
     ```
   - Client logs include trace and span IDs for requests:
     ```
     {"level":"info","ts":"2025-05-04T13:38:12.183+0700","caller":"logger/logger.go:196","msg":"Sending request","trace_id":"3b68e43a942ee1c9386c3a9f209b5dc6","span_id":"06c13a501e3ffa04","method":"GET","url":"http://127.0.0.1:62814/api/v1/Hello?name=Test"}
     ```
   - View traces in the collector’s UI (e.g., Jaeger at `http://localhost:16686`).

#### Example
See the example files in the `examples/` directory (`server/main.go`, `client/main.go`) for a complete implementation with otel tracing and graceful shutdown.

#### Notes
- Tracing is optional and disabled by default (`otel_enabled: false`).
- Ensure the OTLP collector is running before enabling tracing, or tests may fail with connection errors.
- For testing, set `OTEL_TEST_MOCK_EXPORTER=true` to use a mock exporter:
  ```bash
  OTEL_TEST_MOCK_EXPORTER=true go test -v ./httpc
  ```

### Graceful Shutdown
The server supports graceful shutdown via the `Shutdown` method, allowing active connections to complete within a configurable timeout (default: 5 seconds). This ensures no requests are dropped during server termination, making it suitable for production environments.

Example:
```go
go func() {
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        logger.Error("Server failed to start", logger.ErrField(err))
        panic("Failed to start server: " + err.Error())
    }
}()

sigs := make(chan os.Signal, 1)
signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
<-sigs

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
if err := server.Shutdown(ctx); err != nil {
    logger.Error("Server shutdown failed", logger.ErrField(err))
    panic("Server shutdown failed: " + err.Error())
}
logger.Info("Server shut down gracefully")
```

Verify shutdown with logs:
```
{"level":"info","ts":"2025-05-04T13:38:12.183+0700","caller":"logger/logger.go:196","msg":"Shutting down server"}
{"level":"info","ts":"2025-05-04T13:38:12.184+0700","caller":"logger/logger.go:196","msg":"Server shut down gracefully"}
```

## Configuration
Configured via environment variables or a map, loaded by `config`:

```go
type ServerConfig struct {
    OtelEnabled bool `json:"otel_enabled" default:"false"`
    Port        int  `json:"port" default:"8080" required:"true" validate:"gt=0,lte=65535"`
}

type ClientConfig struct {
    OtelEnabled          bool  `json:"otel_enabled" default:"false"`
    TimeoutMs            int   `json:"http_client_timeout_ms" default:"1000" required:"true" validate:"gt=0"`
    MaxRetries           int   `json:"http_client_max_retries" default:"2" validate:"gte=-1"`
    BackoffBaseMs        int64 `json:"http_client_backoff_base_ms" default:"100" validate:"gte=50,lte=1000"`
    BackoffMaxMs         int64 `json:"http_client_backoff_max_ms" default:"1000" validate:"gte=100,lte=5000"`
    BackoffFactor        int   `json:"http_client_backoff_factor" default:"2" validate:"gte=1,lte=5"`
    DisableBackoff       bool  `json:"http_client_disable_backoff" default:"false"`
}
```

### Configuration Options
- **otel_enabled**: Enables OpenTelemetry tracing (env: `CONFIG_OTEL_ENABLED`, default: `false`).
- **otel_endpoint**: OTLP collector endpoint (env: `CONFIG_OTEL_ENDPOINT`, default: `localhost:4317`).
- **port**: Server port (env: `CONFIG_PORT`, default: `8080`).
- **http_client_timeout_ms**: Client request timeout in milliseconds (env: `CONFIG_HTTP_CLIENT_TIMEOUT_MS`, default: `1000`).
- **http_client_max_retries**: Maximum retries for client requests (env: `CONFIG_HTTP_CLIENT_MAX_RETRIES`, default: `2`).
- **http_client_backoff_base_ms**: Base backoff duration in milliseconds (env: `CONFIG_HTTP_CLIENT_BACKOFF_BASE_MS`, default: `100`).
- **http_client_backoff_max_ms**: Maximum backoff duration in milliseconds (env: `CONFIG_HTTP_CLIENT_BACKOFF_MAX_MS`, default: `1000`).
- **http_client_backoff_factor**: Backoff multiplier (env: `CONFIG_HTTP_CLIENT_BACKOFF_FACTOR`, default: `2`).
- **http_client_disable_backoff**: Disables backoff between retries (env: `CONFIG_HTTP_CLIENT_DISABLE_BACKOFF`, default: `false`).

Example configuration map:
```go
cfg, err := config.New(config.WithDefault(map[string]interface{}{
    "otel_enabled":               true,
    "otel_endpoint":              "localhost:4317",
    "port":                       8080,
    "http_client_timeout_ms":     1000,
    "http_client_max_retries":    2,
    "http_client_backoff_base_ms": 100,
    "http_client_backoff_max_ms": 1000,
    "http_client_backoff_factor": 2,
    "http_client_disable_backoff": false,
}))
```

## Testing
Comprehensive tests cover server endpoint registration, client requests, reflection-based service handling, input validation, error handling, retry logic, graceful shutdown, and optional OpenTelemetry tracing. Tests validate JSON payloads, string/struct responses, and error cases (e.g., invalid configurations, malformed JSON, validation failures, server errors). The test suite is organized into:

- `httpc_test.go`: Tests server and client functionality, including healthcheck, Swagger, tracing, `ListenAndServe`, and error handling.
- `reflect_test.go`: Tests reflection-based service registration for pointer and non-pointer services.
- `swagger_test.go`: Tests OpenAPI documentation generation.
- `client_test.go`: Tests client request handling, retries, and backoff.
- `test_service.go`: Defines test services (e.g., `TestService`, `MultiMethodService`) with error cases (e.g., `simulated server error`).
- `error_test.go`: Tests error cases (e.g., invalid HTTP methods).
- `server_test.go`: Tests server-specific functionality, including graceful shutdown.
- `testutil_test.go`: Provides utilities for test server setup with proper `Content-Length` handling.

All tests pass with Go 1.24.2, achieving 82.3% code coverage as of May 4, 2025, with ongoing efforts to reach ≥91.1% by adding tests for edge cases (e.g., invalid configurations, transient errors, shutdown scenarios).

### Running Tests
Run with coverage report:

```bash
cd httpc
go test -v -cover .
```

Ensure all `.go` files (e.g., `httpc.go`, `reflect.go`, `test_service.go`) are in the `httpc` directory. For tracing tests, set `OTEL_TEST_MOCK_EXPORTER=true` if no OTLP collector is running:

```bash
OTEL_TEST_MOCK_EXPORTER=true go test -v -cover .
```

To analyze coverage gaps:

```bash
go test -coverprofile=coverage.out ./httpc
go tool cover -html=coverage.out -o coverage.html
```

### Requirements
- `config`, `logger`, and `otel` packages available in the monorepo.
- Mock server (`net/http/httptest`) for tests.
- OTLP collector at `localhost:4317` for tracing tests (optional, see [Troubleshooting](#troubleshooting)).

## Troubleshooting
### Test Failures
- **Service Registration Errors**: If logs show "No methods defined for service" or "service must implement RegisterMethods", verify that the service implements `RegisterMethods` returning `[]httpc.MethodInfo`. For non-pointer services, ensure the service struct is passed by value (e.g., `MyService{}`):
  ```bash
  go test -v ./httpc | grep "getServiceInfo"
  ```
  Check that service files are in the `httpc` directory:
  ```bash
  ls httpc/*.go
  ```

- **Swagger Generation Errors**: If `/api/docs/swagger.json` fails, check logs for `updateSwaggerDoc` errors:
  ```bash
  go test -v ./httpc | grep "Failed to update Swagger doc"
  ```
  Verify that `MethodInfo` fields (`Name`, `HTTPMethod`, `InputType`, `OutputType`) are correctly set.

- **Client Errors**: Ensure `Content-Type: application/json` for POST/PUT/DELETE requests. Check response bodies in test logs for errors like `{"error":"simulated server error"}`:
  ```bash
  go test -v ./httpc | grep "Error response body"
  ```

- **Tracing Errors**: If tracing tests fail, ensure an OTLP collector is running at `localhost:4317` or use the mock exporter:
  ```bash
  OTEL_TEST_MOCK_EXPORTER=true go test -v ./httpc
  ```

- **ListenAndServe Errors**: If `ListenAndServe` fails (e.g., "address already in use"), verify the port is available:
  ```bash
  lsof -i :8080
  ```
  Check logs for startup errors:
  ```bash
  CONFIG_LOGGER_OUTPUT=console CONFIG_LOGGER_JSON_FORMAT=true go test -v ./httpc
  ```

- **Empty 500 Response Bodies**: If tests like `TestHTTPClient/Client_Server_Error` show empty 500 response bodies, verify logs for `c.Data` execution:
  ```bash
  go test -v ./httpc | grep "Sending error response"
  ```
  Ensure `testutil_test.go` sets `Content-Length` correctly. Test manually:
  ```bash
  curl -v http://localhost:8080/api/v1/GetMethod?name=error
  ```

- **Low Test Coverage**: If coverage is below 91.1%, analyze the coverage report:
  ```bash
  go test -coverprofile=coverage.out ./httpc
  go tool cover -html=coverage.out -o coverage.html
  ```
  Add tests for untested paths (e.g., invalid configs, retry failures, shutdown).

- Run tests with verbose logging:
  ```bash
  CONFIG_LOGGER_OUTPUT=console CONFIG_LOGGER_JSON_FORMAT=true go test -v ./httpc
  ```

### Compilation Errors
- Verify dependencies:
  ```bash
  go list -m github.com/T-Prohmpossadhorn/go-core/config
  ```
- Check conflicts:
  ```bash
  go mod graph | grep go-core
  ```
- Clear cache and update:
  ```bash
  go clean -modcache
  go mod tidy
  ```

### Server Startup Issues
- Ensure port 8080 is available.
- Check logs:
  ```bash
  CONFIG_LOGGER_OUTPUT=console CONFIG_LOGGER_JSON_FORMAT=true go run main.go
  ```
- Verify graceful shutdown:
  ```bash
  curl http://localhost:8080/health
  kill -INT <pid>
  ```

### OTLP Collector for Tracing
If tracing tests fail, start an OTLP collector:
```bash
docker run -p 4317:4317 otel/opentelemetry-collector
```

## Contributing
Contribute via:
1. Fork `github.com/T-Prohmpossadhorn/go-core`.
2. Create a branch (e.g., `feature/add-endpoint`).
3. Implement changes, ensure tests pass with ≥91.1% coverage.
4. Run:
   ```bash
   go test -v -cover ./httpc
   ```
5. Submit a pull request with a description of changes and coverage impact.

### Development Setup
```bash
git clone https://github.com/T-Prohmpossadhorn/go-core.git
cd go-core/httpc
go mod tidy
```

### Code Style
- Use `gofmt` and `golint` for formatting and linting.
- Define services with `RegisterMethods` returning `[]httpc.MethodInfo` with correct `InputType` and `OutputType`.
- Cover new functionality with tests in appropriate files (e.g., `httpc_test.go`, `client_test.go`).
- Ensure error responses use `c.Data` with JSON payloads for consistency:
  ```go
  c.Data(http.StatusInternalServerError, "application/json", []byte(`{"error":"`+err.Error()+`"}`))
  ```

### Testing Guidelines
- Add tests for edge cases (e.g., invalid JSON, nil configs, transient errors).
- Verify coverage with:
  ```bash
  go test -coverprofile=coverage.out ./httpc
  go tool cover -func=coverage.out
  ```
- Ensure 500 responses include `{"error":"message"}` with proper `Content-Length`.

## License
MIT License. See `LICENSE` file.