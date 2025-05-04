package config

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// Config holds the application configuration using Viper.
type Config struct {
	mu           sync.RWMutex
	v            *viper.Viper
	configStruct ConfigStruct
}

// ConfigStruct defines configuration fields with default and required tags.
type ConfigStruct struct {
	Environment string            `mapstructure:"environment,required" default:"development"`
	Debug       bool              `mapstructure:"debug" default:"false"`
	Settings    map[string]string `mapstructure:"settings" default:""`
}

// Option configures the Config instance.
type Option func(*Config)

// WithFilepath sets the configuration file path (YAML or JSON).
func WithFilepath(path string) Option {
	return func(c *Config) {
		c.mu.Lock()
		defer c.mu.Unlock()
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".yaml", ".yml":
			c.v.SetConfigType("yaml")
		case ".json":
			c.v.SetConfigType("json")
		default:
			c.v.Set("error", fmt.Errorf("unsupported file format: %s", path))
			return
		}
		c.v.SetConfigFile(path)
		if err := c.v.ReadInConfig(); err != nil {
			c.v.Set("error", fmt.Errorf("failed to read config file %s: %w", path, err))
			return
		}
		if err := c.v.Unmarshal(&c.configStruct); err != nil {
			c.v.Set("error", fmt.Errorf("failed to unmarshal ConfigStruct: %w", err))
			return
		}
		if err := c.validateRequiredFields(); err != nil {
			c.v.Set("error", err)
			return
		}
	}
}

// WithDefault sets default configuration values.
func WithDefault(defaults map[string]interface{}) Option {
	return func(c *Config) {
		c.mu.Lock()
		defer c.mu.Unlock()
		for k, v := range defaults {
			c.v.SetDefault(k, v)
		}
	}
}

// WithEnv loads configuration from environment variables.
func WithEnv(prefix string) Option {
	return func(c *Config) {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.v.SetEnvPrefix(strings.ToUpper(prefix))
		c.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		c.v.AutomaticEnv()
		keys := []string{"environment", "debug", "app.name", "app.port"}
		for _, key := range keys {
			envKey := strings.ReplaceAll(key, ".", "_")
			if err := c.v.BindEnv(key, fmt.Sprintf("%s_%s", strings.ToUpper(prefix), strings.ToUpper(envKey))); err != nil {
				c.v.Set("error", fmt.Errorf("failed to bind env var %s: %w", key, err))
				return
			}
		}
		if err := c.v.Unmarshal(&c.configStruct); err != nil {
			c.v.Set("error", fmt.Errorf("failed to unmarshal ConfigStruct from env: %w", err))
			return
		}
		if err := c.validateRequiredFields(); err != nil {
			c.v.Set("error", err)
			return
		}
	}
}

// New creates a new Config instance.
func New(opts ...Option) (*Config, error) {
	v := viper.New()
	c := &Config{
		v: v,
		configStruct: ConfigStruct{
			Settings: make(map[string]string),
		},
	}
	// Apply defaults before validating required fields
	if err := c.applyDefaults(); err != nil {
		return nil, fmt.Errorf("failed to apply defaults: %w", err)
	}
	if err := c.validateRequiredFields(); err != nil {
		return nil, fmt.Errorf("required field validation failed: %w", err)
	}
	for _, opt := range opts {
		opt(c)
	}
	if err := c.v.Get("error"); err != nil {
		return nil, err.(error)
	}
	return c, nil
}

// applyDefaults applies default values from struct tags.
func (c *Config) applyDefaults() error {
	v := reflect.ValueOf(&c.configStruct).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		defaultVal := field.Tag.Get("default")
		if defaultVal == "" {
			continue
		}
		f := v.Field(i)
		if !f.CanSet() {
			return fmt.Errorf("cannot set field %s: not addressable", field.Name)
		}
		if !f.IsZero() {
			continue
		}
		switch f.Kind() {
		case reflect.String:
			f.SetString(defaultVal)
		case reflect.Bool:
			f.SetBool(defaultVal == "true")
		case reflect.Map:
			if defaultVal == "" {
				f.Set(reflect.MakeMap(f.Type()))
			}
		default:
			return fmt.Errorf("unsupported field type for default: %v", f.Kind())
		}
	}
	return nil
}

// validateRequiredFields checks for required fields in ConfigStruct.
func (c *Config) validateRequiredFields() error {
	v := reflect.ValueOf(c.configStruct)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("mapstructure")
		if strings.Contains(tag, ",required") {
			f := v.Field(i)
			if f.IsZero() {
				return fmt.Errorf("required field %s is not set", field.Name)
			}
		}
	}
	return nil
}

// Get retrieves a configuration value by key.
func (c *Config) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.v.Get(key)
}

// GetStringWithDefault retrieves a string value with a default.
func (c *Config) GetStringWithDefault(key, defaultValue string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.v.IsSet(key) {
		return c.v.GetString(key)
	}
	return defaultValue
}

// GetBool retrieves a boolean value.
func (c *Config) GetBool(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.v.GetBool(key)
}

// GetStringMapString retrieves a map[string]string.
func (c *Config) GetStringMapString(key string) map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.v.GetStringMapString(key)
}

// GetConfigStruct retrieves the ConfigStruct.
func (c *Config) GetConfigStruct() ConfigStruct {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configStruct
}

// Unmarshal unmarshals the entire configuration into the target struct.
func (c *Config) Unmarshal(target interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.v.Unmarshal(target)
}
