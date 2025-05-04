# Config Package

The `config` package provides a thread-safe configuration management system for Go applications, built on top of the [Viper](https://github.com/spf13/viper) library. It supports YAML/JSON files, environment variables, programmatic defaults, and flexible unmarshaling of the entire configuration into custom structs, with support for required fields and default values via struct tags.

## Features
- Load configuration from YAML or JSON files.
- Load configuration from environment variables with a prefix.
- Thread-safe access to configuration values.
- Retrieve values as strings, booleans, or string maps with defaults.
- Define configuration fields with required and default values using struct tags.
- Set programmatic default values, including nested structures, using `WithDefault`.
- Unmarshal the entire configuration into arbitrary structs using `Unmarshal`.
- Access structured configuration via `ConfigStruct` with validation.

## Installation
```bash
go get github.com/T-Prohmpossadhorn/go-core
go get github.com/spf13/viper
```

## Configuration
Configuration can be defined in YAML/JSON files or environment variables.

### YAML File (e.g., `config.yaml`):
```yaml
environment: production
debug: true
app:
  name: my-app
  port: "9090"
```

### JSON File (e.g., `config.json`):
```json
{
  "environment": "production",
  "debug": true,
  "app": {
    "name": "my-app",
    "port": "9090"
  }
}
```

### Environment Variables (with prefix `CONFIG_`):
```bash
export CONFIG_ENVIRONMENT=production
export CONFIG_DEBUG=true
export CONFIG_APP_NAME=my-app
export CONFIG_APP_PORT=9090
```

### ConfigStruct
The `ConfigStruct` defines configuration fields with `mapstructure` tags for unmarshaling, `default` tags for default values, and `required` tags for mandatory fields:
```go
type ConfigStruct struct {
    Environment string            `mapstructure:"environment,required" default:"development"`
    Debug       bool              `mapstructure:"debug" default:"false"`
    Settings    map[string]string `mapstructure:"settings" default:""`
}
```
- **Defaults**: `Environment="development"`, `Debug=false`, `Settings=map[]`.
- **Required**: `Environment` must be set in config sources or defaults.

## Usage
### Initialize Config
Below are simple examples demonstrating the `config` package with Viper. They show loading configuration from defaults, YAML files, JSON files, and environment variables, with required field validation. For complete executable examples, see:
- `examples/config/simple_default/main.go`
- `examples/config/simple_yaml/main.go`
- `examples/config/simple_json/main.go`
- `examples/config/simple_env/main.go`

#### Example 1: Using Defaults
```go
package main

import (
    "fmt"
    "github.com/T-Prohmpossadhorn/go-core/config"
)

func main() {
    // Set programmatic defaults
    defaults := map[string]interface{}{
        "environment": "staging",
        "debug":       true,
        "app.name":    "my-app",
        "app.port":    "8080",
    }

    // Initialize config
    cfg, err := config.New(config.WithDefault(defaults))
    if err != nil {
        fmt.Printf("Failed to initialize config: %v\n", err)
        return
    }

    // Access raw values
    env := cfg.GetStringWithDefault("environment", "dev")
    appName := cfg.GetStringWithDefault("app.name", "default-app")
    fmt.Printf("Environment: %s\n", env)
    fmt.Printf("App Name: %s\n", appName)
}
```

**Expected Output**:
```
Environment: staging
App Name: my-app
```

#### Example 2: Using YAML File
```go
package main

import (
    "fmt"
    "github.com/T-Prohmpossadhorn/go-core/config"
)

// Sample config.yaml:
// environment: production
// debug: false
// app:
//   name: my-app
//   port: "9090"

func main() {
    // Initialize config with YAML file
    cfg, err := config.New(config.WithFilepath("config.yaml"))
    if err != nil {
        fmt.Printf("Failed to initialize config: %v\n", err)
        return
    }

    // Access structured config
    cfgStruct := cfg.GetConfigStruct()
    fmt.Printf("Environment: %s\n", cfgStruct.Environment)

    // Unmarshal configuration
    type AppConfig struct {
        Environment string `mapstructure:"environment"`
        App         struct {
            Name string `mapstructure:"name"`
        } `mapstructure:"app"`
    }
    var appConfig AppConfig
    if err := cfg.Unmarshal(&appConfig); err != nil {
        fmt.Printf("Failed to unmarshal config: %v\n", err)
        return
    }
    fmt.Printf("Unmarshaled Environment: %s\n", appConfig.Environment)
    fmt.Printf("Unmarshaled App Name: %s\n", appConfig.App.Name)
}
```

**Expected Output**:
```
Environment: production
Unmarshaled Environment: production
Unmarshaled App Name: my-app
```

#### Example 3: Using JSON File
```go
package main

import (
    "fmt"
    "github.com/T-Prohmpossadhorn/go-core/config"
)

// Sample config.json:
// {
//   "environment": "production",
//   "debug": false,
//   "app": {
//     "name": "my-app",
//     "port": "9090"
//   }
// }

func main() {
    // Initialize config with JSON file
    cfg, err := config.New(config.WithFilepath("config.json"))
    if err != nil {
        fmt.Printf("Failed to initialize config: %v\n", err)
        return
    }

    // Access raw values
    env := cfg.GetStringWithDefault("environment", "dev")
    appName := cfg.GetStringWithDefault("app.name", "default-app")
    fmt.Printf("Environment: %s\n", env)
    fmt.Printf("App Name: %s\n", appName)
}
```

**Expected Output**:
```
Environment: production
App Name: my-app
```

#### Example 4: Using Environment Variables
```go
package main

import (
    "fmt"
    "github.com/T-Prohmpossadhorn/go-core/config"
)

// Sample environment variables:
// export CONFIG_ENVIRONMENT=production
// export CONFIG_APP_NAME=my-app

func main() {
    // Initialize config with environment variables
    cfg, err := config.New(config.WithEnv("CONFIG"))
    if err != nil {
        fmt.Printf("Failed to initialize config: %v\n", err)
        return
    }

    // Access raw values
    env := cfg.GetStringWithDefault("environment", "dev")
    appName := cfg.GetStringWithDefault("app.name", "default-app")
    fmt.Printf("Environment: %s\n", env)
    fmt.Printf("App Name: %s\n", appName)
}
```

**Expected Output**:
```
Environment: production
App Name: my-app
```

## API Reference
### Types
- `Config`: Holds the application configuration using Viper.
  ```go
  type Config struct {
      mu           sync.RWMutex
      v            *viper.Viper
      configStruct ConfigStruct
  }
  ```
- `ConfigStruct`: Defines configuration fields with defaults and required tags.
  ```go
  type ConfigStruct struct {
      Environment string            `mapstructure:"environment,required" default:"development"`
      Debug       bool              `mapstructure:"debug" default:"false"`
      Settings    map[string]string `mapstructure:"settings" default:""`
  }
  ```
- `Option`: Configures the Config instance.
  ```go
  type Option func(*Config)
  ```

### Functions
- `New(opts ...Option) (*Config, error)`: Creates a new Config instance, applying defaults and validating required fields.
  - Options: `WithFilepath(string)`, `WithDefault(map[string]interface{})`, `WithEnv(string)`.
- `WithFilepath(path string) Option`: Sets the configuration file path (YAML or JSON).
- `WithDefault(defaults map[string]interface{}) Option`: Sets default configuration values, supporting nested keys (e.g., `app.name`).
- `WithEnv(prefix string) Option`: Enables environment variable loading with the given prefix (e.g., `CONFIG`), mapping underscores to dots (e.g., `CONFIG_APP_NAME` to `app.name`).

### Methods
- `Get(key string) interface{}`: Retrieves a raw configuration value.
- `GetStringWithDefault(key, defaultValue string) string`: Retrieves a string value with a default.
- `GetBool(key string) bool`: Retrieves a boolean value.
- `GetStringMapString(key string) map[string]string`: Retrieves a string map.
- `GetConfigStruct() ConfigStruct`: Retrieves the structured configuration.
- `Unmarshal(target interface{}) error`: Unmarshals the entire configuration into the target struct using `mapstructure` tags.

## Testing
Run tests with:
```bash
go test -v ./config
```
Tests cover:
- Initialization with defaults (struct tags and programmatic).
- Loading valid YAML and JSON files.
- Loading environment variables.
- Handling missing or invalid files.
- Accessing raw and structured values.
- Unmarshaling the entire configuration into custom structs.
- Validating required fields and applying default values.
- Applying nested programmatic defaults with `WithDefault`.

## Notes
- Default values are applied in this order: struct tag defaults, programmatic defaults (`WithDefault`), environment variables (`WithEnv`), file-based configuration.
- Required fields (e.g., `Environment`) must be set in at least one configuration source or default.
- `WithDefault` and `WithEnv` support nested keys (e.g., `app.name`).
- Environment variables are parsed as strings; convert to `int` or other types as needed.
- The `settings` map is initialized as an empty map if not specified.
- Use `mapstructure` tags in structs for unmarshaling with `Unmarshal`.
- Requires the Viper library (`github.com/spf13/viper`). Ensure version `v1.19.0` or later is used.

## Debugging Tips
If tests fail, particularly `TestLoadFromJSON` or `TestRequiredFieldMissing`:
- **Check Viper Version**: Run `go list -m github.com/spf13/viper` to ensure version `v1.19.0` or later.
- **Verify Go Version**: Run `go version` to confirm `go1.24.2` or later.
- **Enable Debug Logging**: Uncomment `fmt.Printf` statements in `config.go` and `config_test.go` to trace Viper’s `AllSettings()` and configuration state.
- **Check Environment Variables**: Run `printenv | grep CONFIG` to ensure no `CONFIG_*` variables interfere with tests.
- **Verify File System Permissions**: Ensure write permissions to `/var/folders/...` or `/tmp` for temporary test files. Check if files like `/tmp/config*.json` are created correctly.
- **Inspect Temporary Files**: Manually inspect the content of temporary JSON files created in tests (e.g., `/tmp/config*.json`) to confirm they match the expected structure.
- **Run in Clean Environment**: Execute `go test -v ./config` in a fresh terminal to avoid state leakage.
- **Check Test Logs**: If failures persist, enable debug logging and share the output, along with any additional error messages.