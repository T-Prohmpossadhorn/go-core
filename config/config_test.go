package config

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// TestNewConfig tests the default configuration.
func TestNewConfig(t *testing.T) {
	cfg, err := New()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	s := cfg.GetConfigStruct()
	assert.Equal(t, "development", s.Environment)
	assert.False(t, s.Debug)
	assert.NotNil(t, s.Settings)
	assert.Empty(t, s.Settings)
}

// TestLoadFromFile tests loading configuration from a YAML file.
func TestLoadFromFile(t *testing.T) {
	content := []byte(`
environment: production
debug: true
settings:
  key1: value1
custom:
  name: test
  enabled: true
`)
	tmpfile, err := os.CreateTemp("", "config*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.Write(content)
	assert.NoError(t, err)
	tmpfile.Close()

	cfg, err := New(WithFilepath(tmpfile.Name()))
	assert.NoError(t, err)
	s := cfg.GetConfigStruct()
	assert.Equal(t, "production", s.Environment)
	assert.True(t, s.Debug)
	assert.Equal(t, map[string]string{"key1": "value1"}, s.Settings)
	assert.Equal(t, "production", cfg.GetStringWithDefault("environment", "default"))
	assert.True(t, cfg.GetBool("debug"))
	assert.Equal(t, map[string]string{"key1": "value1"}, cfg.GetStringMapString("settings"))
}

// TestLoadFromJSON tests loading configuration from a JSON buffer.
func TestLoadFromJSON(t *testing.T) {
	content := []byte(`{
        "environment": "production",
        "debug": true,
        "settings": {"key1": "value1"},
        "custom": {"name": "test", "enabled": true}
    }`)

	v := viper.New()
	v.SetConfigType("json")
	if err := v.ReadConfig(bytes.NewReader(content)); err != nil {
		t.Fatalf("Failed to read in-memory JSON: %v", err)
	}

	c := &Config{
		v:            v,
		configStruct: ConfigStruct{Settings: make(map[string]string)},
	}

	if err := v.Unmarshal(&c.configStruct); err != nil {
		t.Fatalf("Failed to unmarshal ConfigStruct: %v", err)
	}

	if err := c.validateRequiredFields(); err != nil {
		t.Fatalf("Required field validation failed: %v", err)
	}

	assert.Equal(t, "production", c.GetStringWithDefault("environment", "default"))
	assert.True(t, c.GetBool("debug"))
	assert.Equal(t, map[string]string{"key1": "value1"}, c.GetStringMapString("settings"))

	type CustomConfig struct {
		Name    string `mapstructure:"name"`
		Enabled bool   `mapstructure:"enabled"`
	}
	var custom CustomConfig
	err := v.UnmarshalKey("custom", &custom)
	assert.NoError(t, err)
	assert.Equal(t, "test", custom.Name)
	assert.True(t, custom.Enabled)
}

// TestWithDefault tests setting default configuration values.
func TestWithDefault(t *testing.T) {
	defaults := map[string]interface{}{
		"environment":   "staging",
		"debug":         true,
		"settings.key2": "value2",
		"app.name":      "my-app",
		"app.port":      "8080",
	}
	cfg, err := New(WithDefault(defaults))
	assert.NoError(t, err)
	assert.Equal(t, "staging", cfg.GetStringWithDefault("environment", "default"))
	assert.True(t, cfg.GetBool("debug"))
	assert.Equal(t, map[string]string{"key2": "value2"}, cfg.GetStringMapString("settings"))
	assert.Equal(t, "my-app", cfg.GetStringWithDefault("app.name", "default"))
	assert.Equal(t, "8080", cfg.GetStringWithDefault("app.port", "default"))
}

// TestWithEnv tests loading configuration from environment variables.
func TestWithEnv(t *testing.T) {
	os.Setenv("CONFIG_ENVIRONMENT", "testing")
	os.Setenv("CONFIG_DEBUG", "true")
	os.Setenv("CONFIG_APP_NAME", "env-app")
	os.Setenv("CONFIG_APP_PORT", "9090")
	defer os.Unsetenv("CONFIG_ENVIRONMENT")
	defer os.Unsetenv("CONFIG_DEBUG")
	defer os.Unsetenv("CONFIG_APP_NAME")
	defer os.Unsetenv("CONFIG_APP_PORT")

	viper.Reset()
	cfg, err := New(WithEnv("CONFIG"))
	assert.NoError(t, err)
	assert.Equal(t, "testing", cfg.GetStringWithDefault("environment", "default"))
	assert.True(t, cfg.GetBool("debug"))
	assert.Equal(t, "env-app", cfg.GetStringWithDefault("app.name", "default"))
	assert.Equal(t, "9090", cfg.GetStringWithDefault("app.port", "8080"))

	type AppConfig struct {
		Environment string `mapstructure:"environment"`
		Debug       bool   `mapstructure:"debug"`
		App         struct {
			Name string `mapstructure:"name"`
			Port string `mapstructure:"port"`
		} `mapstructure:"app"`
	}
	var appConfig AppConfig
	err = cfg.Unmarshal(&appConfig)
	assert.NoError(t, err)
	assert.Equal(t, "testing", appConfig.Environment)
	assert.True(t, appConfig.Debug)
	assert.Equal(t, "env-app", appConfig.App.Name)
	assert.Equal(t, "9090", appConfig.App.Port)
}

// TestUnmarshal tests unmarshaling the entire configuration.
func TestUnmarshal(t *testing.T) {
	content := []byte(`
environment: production
debug: true
settings:
  key1: value1
custom:
  name: test
  enabled: true
`)
	tmpfile, err := os.CreateTemp("", "config*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.Write(content)
	assert.NoError(t, err)
	tmpfile.Close()

	cfg, err := New(WithFilepath(tmpfile.Name()))
	assert.NoError(t, err)

	type FullConfig struct {
		Environment string            `mapstructure:"environment"`
		Debug       bool              `mapstructure:"debug"`
		Settings    map[string]string `mapstructure:"settings"`
		Custom      struct {
			Name    string `mapstructure:"name"`
			Enabled bool   `mapstructure:"enabled"`
		} `mapstructure:"custom"`
	}
	var full FullConfig
	err = cfg.Unmarshal(&full)
	assert.NoError(t, err)
	assert.Equal(t, "production", full.Environment)
	assert.True(t, full.Debug)
	assert.Equal(t, map[string]string{"key1": "value1"}, full.Settings)
	assert.Equal(t, "test", full.Custom.Name)
	assert.True(t, full.Custom.Enabled)
}

// TestDefaults tests default configuration values.
func TestDefaults(t *testing.T) {
	cfg, err := New()
	assert.NoError(t, err)
	s := cfg.GetConfigStruct()
	assert.Equal(t, "development", s.Environment)
	assert.False(t, s.Debug)
	assert.NotNil(t, s.Settings)
	assert.Empty(t, s.Settings)
}

// TestInvalidFile tests loading a nonexistent file.
func TestInvalidFile(t *testing.T) {
	cfg, err := New(WithFilepath("nonexistent.yaml"))
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// TestInvalidJSON tests loading invalid JSON.
func TestInvalidJSON(t *testing.T) {
	content := []byte(`{invalid json}`)
	tmpfile, err := os.CreateTemp("", "config*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.Write(content)
	assert.NoError(t, err)
	tmpfile.Close()

	cfg, err := New(WithFilepath(tmpfile.Name()))
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

// TestWithDefaultAndEnv tests combining defaults and environment variables.
func TestWithDefaultAndEnv(t *testing.T) {
	os.Setenv("CONFIG_ENVIRONMENT", "testing")
	os.Setenv("CONFIG_APP_NAME", "env-app")
	defer os.Unsetenv("CONFIG_ENVIRONMENT")
	defer os.Unsetenv("CONFIG_APP_NAME")

	defaults := map[string]interface{}{
		"environment": "staging",
		"debug":       true,
		"app.name":    "default-app",
		"app.port":    "8080",
	}

	viper.Reset()
	cfg, err := New(WithDefault(defaults), WithEnv("CONFIG"))
	assert.NoError(t, err)
	assert.Equal(t, "testing", cfg.GetStringWithDefault("environment", "default")) // Env overrides default
	assert.True(t, cfg.GetBool("debug"))                                           // Default preserved
	assert.Equal(t, "env-app", cfg.GetStringWithDefault("app.name", "default"))    // Env overrides default
	assert.Equal(t, "8080", cfg.GetStringWithDefault("app.port", "default"))       // Default preserved
}

// TestRequiredFieldMissing tests a configuration missing a required field.
func TestRequiredFieldMissing(t *testing.T) {
	content := []byte(`{
        "debug": true,
        "settings": {"key1": "value1"}
    }`)

	v := viper.New()
	v.SetConfigType("json")
	if err := v.ReadConfig(bytes.NewReader(content)); err != nil {
		t.Fatalf("Failed to read in-memory JSON: %v", err)
	}

	c := &Config{
		v:            v,
		configStruct: ConfigStruct{Settings: make(map[string]string)},
	}

	if err := v.Unmarshal(&c.configStruct); err != nil {
		t.Fatalf("Failed to unmarshal ConfigStruct: %v", err)
	}

	err := c.validateRequiredFields()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required field Environment is not set")
}

// TestRequiredFieldSet tests a configuration with the required field set.
func TestRequiredFieldSet(t *testing.T) {
	content := []byte(`{
        "environment": "production",
        "debug": true,
        "settings": {"key1": "value1"}
    }`)
	tmpfile, err := os.CreateTemp("", "config*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.Write(content)
	assert.NoError(t, err)
	tmpfile.Close()

	cfg, err := New(WithFilepath(tmpfile.Name()))
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	s := cfg.GetConfigStruct()
	assert.Equal(t, "production", s.Environment)
}

// TestUnsupportedFileFormat tests WithFilepath with an unsupported file extension.
func TestUnsupportedFileFormat(t *testing.T) {
	cfg, err := New(WithFilepath("config.txt"))
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "unsupported file format")
}

// TestEmptyConfigFile tests loading an empty YAML file.
func TestEmptyConfigFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "config*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	tmpfile.Close() // Empty file

	cfg, err := New(WithFilepath(tmpfile.Name()))
	assert.NoError(t, err) // Empty file is valid, defaults apply
	s := cfg.GetConfigStruct()
	assert.Equal(t, "development", s.Environment) // Default value
	assert.False(t, s.Debug)
	assert.Empty(t, s.Settings)
}

// TestUnreadableConfigFile tests loading a file with invalid permissions.
func TestUnreadableConfigFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	tmpfile, err := os.CreateTemp("", "config*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	err = os.Chmod(tmpfile.Name(), 0000) // Make file unreadable
	assert.NoError(t, err)
	tmpfile.Close()

	cfg, err := New(WithFilepath(tmpfile.Name()))
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "failed to read config file")
}

// TestWithPartialEnv tests WithEnv when only some environment variables are set.
func TestWithPartialEnv(t *testing.T) {
	os.Setenv("CONFIG_DEBUG", "true")
	defer os.Unsetenv("CONFIG_DEBUG")

	viper.Reset()
	cfg, err := New(WithEnv("CONFIG"))
	assert.NoError(t, err) // Environment is set to default
	s := cfg.GetConfigStruct()
	assert.Equal(t, "development", s.Environment) // Default
	assert.True(t, s.Debug)                       // From env
}

// TestWithInvalidEnvFormat tests environment variables with invalid formats.
func TestWithInvalidEnvFormat(t *testing.T) {
	os.Setenv("CONFIG_ENVIRONMENT", "testing")
	os.Setenv("CONFIG_DEBUG", "not-a-boolean")
	defer os.Unsetenv("CONFIG_ENVIRONMENT")
	defer os.Unsetenv("CONFIG_DEBUG")

	viper.Reset()
	v := viper.New()
	v.SetEnvPrefix("CONFIG")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.BindEnv("environment", "CONFIG_ENVIRONMENT")
	v.BindEnv("debug", "CONFIG_DEBUG")

	c := &Config{
		v:            v,
		configStruct: ConfigStruct{Settings: make(map[string]string)},
	}
	err := c.applyDefaults()
	assert.NoError(t, err)

	// Avoid Unmarshal to ConfigStruct; test Viper directly
	assert.Equal(t, "testing", v.GetString("environment"))
	assert.False(t, v.GetBool("debug")) // Invalid boolean defaults to false
}

// TestGetNonExistentKey tests Get with a non-existent key.
func TestGetNonExistentKey(t *testing.T) {
	cfg, err := New()
	assert.NoError(t, err)
	value := cfg.Get("nonexistent.key")
	assert.Nil(t, value)
}

// TestGetBoolNonBoolean tests GetBool with a non-boolean value.
func TestGetBoolNonBoolean(t *testing.T) {
	content := []byte(`{
        "environment": "production",
        "debug": "not-a-boolean",
        "settings": {"key1": "value1"}
    }`)
	v := viper.New()
	v.SetConfigType("json")
	err := v.ReadConfig(bytes.NewReader(content))
	assert.NoError(t, err)

	c := &Config{
		v:            v,
		configStruct: ConfigStruct{Settings: make(map[string]string)},
	}
	err = c.applyDefaults()
	assert.NoError(t, err)

	// Avoid Unmarshal to ConfigStruct; test Viper directly
	assert.False(t, v.GetBool("debug")) // Non-boolean value returns false
}

// TestUnmarshalInvalidTarget tests Unmarshal with an invalid target.
func TestUnmarshalInvalidTarget(t *testing.T) {
	cfg, err := New()
	assert.NoError(t, err)

	// Non-pointer target
	var invalid int
	err = cfg.Unmarshal(invalid)
	assert.Error(t, err)

	// Nil pointer
	var nilPtr *struct{}
	err = cfg.Unmarshal(nilPtr)
	assert.Error(t, err)
}

// TestConcurrentAccess tests thread safety of config methods.
func TestConcurrentAccess(t *testing.T) {
	cfg, err := New()
	assert.NoError(t, err)

	var wg sync.WaitGroup
	numGoroutines := 10
	iterations := 100

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = cfg.Get("environment")
				_ = cfg.GetStringWithDefault("debug", "false")
				_ = cfg.GetBool("debug")
				_ = cfg.GetStringMapString("settings")
				var s ConfigStruct
				_ = cfg.Unmarshal(&s)
			}
		}()
	}

	wg.Wait()
}

// TestCaseSensitivity tests case sensitivity in configuration keys.
func TestCaseSensitivity(t *testing.T) {
	content := []byte(`{
        "ENVIRONMENT": "production",
        "debug": true,
        "settings": {"key1": "value1"}
    }`)
	v := viper.New()
	v.SetConfigType("json")
	err := v.ReadConfig(bytes.NewReader(content))
	assert.NoError(t, err)

	// Viper key access is case-insensitive by default
	assert.Equal(t, "production", v.GetString("ENVIRONMENT"))
	assert.Equal(t, "production", v.GetString("environment")) // Case-insensitive

	c := &Config{
		v:            v,
		configStruct: ConfigStruct{Settings: make(map[string]string)},
	}
	err = v.Unmarshal(&c.configStruct)
	assert.NoError(t, err)
	err = c.validateRequiredFields()
	assert.NoError(t, err)                                                   // Defaults or unmarshaling satisfy required fields
	assert.Equal(t, "production", c.GetStringWithDefault("environment", "")) // ConfigStruct populated
}

// TestNestedConfig tests deeply nested configuration.
func TestNestedConfig(t *testing.T) {
	content := []byte(`{
        "environment": "production",
        "debug": true,
        "settings": {"key1": "value1"},
        "app": {
            "name": "test-app",
            "config": {
                "port": "8080",
                "timeout": "30s"
            }
        }
    }`)
	tmpfile, err := os.CreateTemp("", "config*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())
	_, err = tmpfile.Write(content)
	assert.NoError(t, err)
	tmpfile.Close()

	cfg, err := New(WithFilepath(tmpfile.Name()))
	assert.NoError(t, err)

	type NestedConfig struct {
		Environment string `mapstructure:"environment"`
		App         struct {
			Name   string `mapstructure:"name"`
			Config struct {
				Port    string `mapstructure:"port"`
				Timeout string `mapstructure:"timeout"`
			} `mapstructure:"config"`
		} `mapstructure:"app"`
	}
	var nested NestedConfig
	err = cfg.Unmarshal(&nested)
	assert.NoError(t, err)
	assert.Equal(t, "production", nested.Environment)
	assert.Equal(t, "test-app", nested.App.Name)
	assert.Equal(t, "8080", nested.App.Config.Port)
	assert.Equal(t, "30s", nested.App.Config.Timeout)
}
