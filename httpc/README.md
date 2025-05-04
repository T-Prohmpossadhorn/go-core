# httpc Package

The `httpc` package is a Gin-based HTTP server and client for Go applications, part of the `github.com/T-Prohmpossadhorn/go-core` monorepo. It provides a robust framework for building RESTful APIs with reflection-based service registration, supporting complex JSON payloads, input validation, and optional OpenTelemetry tracing. Designed for microservices, it integrates with `config` and `logger` for configuration and logging, achieving high test coverage and thread-safety.

## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
  - [Registering a Service](#registering-a-service)
  - [Sending HTTP Requests](#sending-http-requests)
  - [Healthcheck Endpoint](#healthcheck-endpoint)
  - [OpenAPI Documentation](#openapi-documentation)
  - [OpenTelemetry Integration](#opentelemetry-integration)
- [Configuration](#configuration)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [License](#license)

## Features
- **Gin-Based Server**: Uses `github.com/gin-gonic/gin@v1.10.0` for routing and middleware, supporting GET and POST methods with extensible endpoint registration via `ListenAndServe`.
- **HTTP Client**: Sends HTTP requests with configurable timeouts and retries, supporting GET and POST with JSON payloads and string responses.
- **Reflection-Based Service Registration**: Registers service methods as HTTP endpoints using `RegisterMethods`, supporting both pointer and non-pointer service types for flexibility.
- **Complex JSON Support**: Handles nested JSON payloads with strict validation using `github.com/go-playground/validator/v10@v10.26.0`, enforcing required fields, length constraints, and custom rules.
- **Healthcheck**: `/health` endpoint returning `200 OK`.
- **Swagger Documentation**: Generates Swagger JSON at `/api/docs/swagger.json` for registered endpoints, reflecting service methods and schemas.
- **Mandatory Integration**: Uses `config` for settings and `logger` for request logging.
- **Optional Tracing**: Supports `go.opentelemetry.io/otel@v1.24.0` for request tracing when enabled, for both server and client.
- **Thread-Safety**: Safe for concurrent requests.
- **High Test Coverage**: Achieves 88.3% coverage with comprehensive unit tests.
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
The `httpc` package enables building HTTP servers with reflection-based service registration and clients for sending GET and POST requests. Server endpoints support JSON payloads with validation for POST requests and query parameters for GET requests, returning JSON or string responses. The client handles JSON requests and responses with configurable retries. Both server and client support optional OpenTelemetry tracing for distributed systems.

### Registering a Service
Define a service struct with a `RegisterMethods` method to specify HTTP endpoints:

```go
package main

import (
    "context"
    "reflect"
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

    if err := server.ListenAndServe(); err != nil {
        panic("Failed to start server: " + err.Error())
    }
}
```

### Sending HTTP Requests
Create an `HTTPClient` to send GET or POST requests:

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
# Response: {"error":"validation failed: Key: 'User.Name' Error:Field validation for 'Name' failed on the 'required' tag; Key: 'User.Address.City' Error:Field validation for 'City' failed on the 'required' tag"}
```

### Healthcheck Endpoint
Access the healthcheck endpoint:

```bash
curl http://localhost:8080/health
# Response: 200 OK
```

### OpenAPI Documentation
Access the Swagger JSON at `http://localhost:8080/api/docs/swagger.json` to explore the API. The dynamically generated Swagger documentation reflects service methods and schemas.

Example:
```bash
curl http://localhost:8080/api/docs/swagger.json
```

Response (abridged):
```json
{
    "swagger": "2.0",
    "info": {
        "title": "httpc API",
        "version": "1.0.0"
    },
    "paths": {
        "/api/v1/Hello": {
            "get": {
                "summary": "Hello",
                "parameters": [
                    {
                        "name": "name",
                        "in": "query",
                        "type": "string",
                        "required": false
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Successful response",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/api/v1/Create": {
            "post": {
                "summary": "Create",
                "consumes": ["application/json"],
                "parameters": [
                    {
                        "name": "body",
                        "in": "body",
                        "required": true,
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
                ],
                "responses": {
                    "200": {
                        "description": "Successful response",
                        "schema": {
                            "type": "string"
                        }
                    },
                    "400": {
                        "description": "Bad request",
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
       "otel_enabled": true,
       "otel_endpoint": "localhost:4317",
       "port":         8080, // Server
       "http_client_timeout_ms": 1000, // Client
       "http_client_max_retries": 2,   // Client
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
     {"level":"info","ts":"2025-05-04T13:38:12.183+0700","caller":"logger/logger.go:196","msg":"Sending request","trace_id":"3b68e43a942ee1c9386c3a9f209b5dc6","span_id":"06c13a501e3ffa04","method":"GET","url":"http://127.0.0.1:62814/v1/Hello?name=Test"}
     ```
   - View traces in the collector’s UI (e.g., Jaeger at `http://localhost:16686`).

#### Example
See the example files in the `examples/` directory (`server_example.go`, `client_example.go`) for a complete implementation with otel tracing enabled.

#### Notes
- Tracing is optional and disabled by default (`otel_enabled: false`).
- Ensure the OTLP collector is running before enabling tracing, or tests may fail with connection errors.
- For testing, set `OTEL_TEST_MOCK_EXPORTER=true` to use a mock exporter:
  ```bash
  OTEL_TEST_MOCK_EXPORTER=true go test -v ./httpc
  ```

## Configuration
Configured via environment variables or a map, loaded by `config`:

```go
type ServerConfig struct {
    OtelEnabled bool `json:"otel_enabled" default:"false"`
    Port        int  `json:"port" default:"8080" required:"true" validate:"gt=0,lte=65535"`
}

type ClientConfig struct {
    OtelEnabled          bool `json:"otel_enabled" default:"false"`
    HTTPClientTimeoutMs  int  `json:"http_client_timeout_ms" default:"1000" required:"true" validate:"gt=0"`
    HTTPClientMaxRetries int  `json:"http_client_max_retries" default:"2" validate:"gte=-1"`
}
```

### Configuration Options
- **otel_enabled**: Enables OpenTelemetry tracing (env: `CONFIG_OTEL_ENABLED`, default: `false`).
- **otel_endpoint**: OTLP collector endpoint (env: `CONFIG_OTEL_ENDPOINT`, default: `localhost:4317`).
- **port**: Server port (env: `CONFIG_PORT`, default: `8080`).
- **http_client_timeout_ms**: Client request timeout in milliseconds (env: `CONFIG_HTTP_CLIENT_TIMEOUT_MS`, default: `1000`).
- **http_client_max_retries**: Maximum retries for client requests (env: `CONFIG_HTTP_CLIENT_MAX_RETRIES`, default: `2`).

Example configuration map:
```go
cfg, err := config.New(config.WithDefault(map[string]interface{}{
    "otel_enabled":           true,
    "otel_endpoint":          "localhost:4317",
    "port":                   8080,
    "http_client_timeout_ms": 1000,
    "http_client_max_retries": 2,
}))
```

## Testing
Comprehensive tests cover server endpoint registration, client requests, reflection-based service handling, input validation, and OpenTelemetry tracing. Tests validate JSON payloads, string responses, and error cases (e.g., invalid configurations, missing methods, validation failures). The test suite is organized into:

- `httpc_test.go`: Tests server and client functionality, including healthcheck, Swagger, tracing, and `ListenAndServe`.
- `reflect_test.go`: Tests reflection-based service registration, including pointer and non-pointer services.
- `swagger_test.go`: Tests Swagger documentation generation.
- `client_test.go`: Tests client request handling.
- `test_services.go`: Defines test services.
- `error_test.go`: Tests error cases.
- `server_test.go`: Tests server-specific functionality.

All tests pass with Go 1.24.2, achieving 88.3% code coverage as of May 4, 2025.

### Running Tests
Run with coverage report:

```bash
cd httpc
go test -v -cover .
```

Ensure all `.go` files (e.g., `httpc.go`, `reflect.go`, `test_services.go`) are in the `httpc` directory. For tracing tests, set `OTEL_TEST_MOCK_EXPORTER=true` if no OTLP collector is running:

```bash
OTEL_TEST_MOCK_EXPORTER=true go test -v -cover .
```

### Requirements
- `config`, `logger`, and `otel` packages available.
- Mock server (`net/http/httptest`) for tests.
- OTLP collector at `localhost:4317` for tracing tests (optional, see [Troubleshooting](#troubleshooting)).

## Troubleshooting
### Test Failures
- **Service Registration Errors**: If logs show "No methods defined for service" or "service must implement RegisterMethods", verify that the service implements `RegisterMethods` returning `[]httpc.MethodInfo`. For non-pointer services, ensure the service struct is passed by value (e.g., `MyService{}`), as the package supports both pointer and non-pointer types:
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
  Verify that `MethodInfo` fields (`Name`, `HTTPMethod`) are non-empty.

- **Client Errors**: Ensure `Content-Type: application/json` for POST requests. Check response bodies in test logs:
  ```bash
  go test -v ./httpc
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

### OTLP Collector for Tracing
If tracing tests fail, start an OTLP collector:
```bash
docker run -p 4317:4317 otel/opentelemetry-collector
```

## Contributing
Contribute via:
1. Fork `github.com/T-Prohmpossadhorn/go-core`.
2. Create a branch (e.g., `feature/add-endpoint`).
3. Implement changes, ensure tests pass with ≥86.1% coverage.
4. Run: `go test -v -cover ./httpc`.
5. Submit a pull request.

### Development Setup
```bash
git clone https://github.com/T-Prohmpossadhorn/go-core.git
cd go-core/httpc
go mod tidy
```

### Code Style
- Use `gofmt` and `golint`.
- Define services with `RegisterMethods` returning `[]httpc.MethodInfo`.
- Cover new functionality with tests in appropriate files (e.g., `httpc_test.go`, `reflect_test.go`).

## License
MIT License. See `LICENSE` file.