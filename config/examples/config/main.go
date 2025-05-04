package main

import (
	"fmt"

	"github.com/T-Prohmpossadhorn/go-core/config"
)

// Sample config.yaml:
// ```yaml
// environment: production
// debug: true
// settings:
//   key1: value1
//   key2: value2
// custom:
//   name: test
//   enabled: true
// otel:
//   enabled: true
//   endpoint: localhost:4317
//   service_name: my-service
// ```

func main() {
	// Set programmatic defaults with nested structures
	defaults := map[string]interface{}{
		"environment": "staging",
		"debug":       true,
		"settings": map[string]interface{}{
			"theme": "dark",
			"mode":  "standard",
		},
		"custom": map[string]interface{}{
			"name":    "default-app",
			"enabled": false,
		},
		"otel": map[string]interface{}{
			"enabled":      false,
			"endpoint":     "localhost:4317",
			"service_name": "default-service",
		},
	}

	// Initialize config with defaults and YAML file
	cfg, err := config.New(
		config.WithDefault(defaults),
		config.WithFilepath("config.yaml"),
	)
	if err != nil {
		fmt.Printf("Failed to initialize config: %v\n", err)
		return
	}

	// Access structured config via ConfigStruct
	cfgStruct := cfg.GetConfigStruct()
	fmt.Printf("Structured Environment: %s\n", cfgStruct.Environment)
	fmt.Printf("Structured Debug: %v\n", cfgStruct.Debug)
	for k, v := range cfgStruct.Settings {
		fmt.Printf("Structured Setting: %s = %s\n", k, v)
	}

	// Access raw values
	env := cfg.GetStringWithDefault("environment", "dev")
	debug := cfg.GetBool("debug")
	settings := cfg.GetStringMapString("settings")
	fmt.Printf("Raw Environment: %s\n", env)
	fmt.Printf("Raw Debug: %v\n", debug)
	for k, v := range settings {
		fmt.Printf("Raw Setting: %s = %s\n", k, v)
	}

	// Unmarshal entire configuration into a custom struct
	type FullConfig struct {
		Environment string            `yaml:"environment"`
		Debug       bool              `yaml:"debug"`
		Settings    map[string]string `yaml:"settings"`
		Custom      struct {
			Name    string `yaml:"name"`
			Enabled bool   `yaml:"enabled"`
		} `yaml:"custom"`
		Otel struct {
			Enabled     bool   `yaml:"enabled"`
			Endpoint    string `yaml:"endpoint"`
			ServiceName string `yaml:"service_name"`
		} `yaml:"otel"`
	}
	var full FullConfig
	if err := cfg.Unmarshal(&full); err != nil {
		fmt.Printf("Failed to unmarshal config: %v\n", err)
		return
	}
	fmt.Printf("Unmarshaled Environment: %s\n", full.Environment)
	fmt.Printf("Unmarshaled Custom Name: %s\n", full.Custom.Name)
	fmt.Printf("Unmarshaled Custom Enabled: %v\n", full.Custom.Enabled)
	fmt.Printf("Unmarshaled Otel Enabled: %v\n", full.Otel.Enabled)
	fmt.Printf("Unmarshaled Otel Endpoint: %s\n", full.Otel.Endpoint)
	fmt.Printf("Unmarshaled Otel ServiceName: %s\n", full.Otel.ServiceName)
}
