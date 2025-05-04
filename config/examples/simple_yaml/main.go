package main

import (
	"fmt"

	"github.com/T-Prohmpossadhorn/go-core/config"
)

// Sample config.yaml:
// ```yaml
// environment: production
// debug: false
// app:
//   name: my-app
//   port: 9090
// ```

func main() {
	// Initialize config with YAML file
	cfg, err := config.New(config.WithFilepath("config.yaml"))
	if err != nil {
		fmt.Printf("Failed to initialize config: %v\n", err)
		return
	}

	// Access structured config via ConfigStruct
	cfgStruct := cfg.GetConfigStruct()
	fmt.Printf("Structured Environment: %s\n", cfgStruct.Environment)
	fmt.Printf("Structured Debug: %v\n", cfgStruct.Debug)

	// Unmarshal entire configuration into a custom struct
	type AppConfig struct {
		Environment string `yaml:"environment"`
		Debug       bool   `yaml:"debug"`
		App         struct {
			Name string `yaml:"name"`
			Port int    `yaml:"port"`
		} `yaml:"app"`
	}
	var appConfig AppConfig
	if err := cfg.Unmarshal(&appConfig); err != nil {
		fmt.Printf("Failed to unmarshal config: %v\n", err)
		return
	}
	fmt.Printf("Unmarshaled Environment: %s\n", appConfig.Environment)
	fmt.Printf("Unmarshaled Debug: %v\n", appConfig.Debug)
	fmt.Printf("Unmarshaled App Name: %s\n", appConfig.App.Name)
	fmt.Printf("Unmarshaled App Port: %d\n", appConfig.App.Port)
}
