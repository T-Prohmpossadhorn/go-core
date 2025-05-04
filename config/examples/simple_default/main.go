package main

import (
	"fmt"

	"github.com/T-Prohmpossadhorn/go-core/config"
)

func main() {
	// Set programmatic defaults, including a nested app section
	defaults := map[string]interface{}{
		"environment": "staging",
		"debug":       true,
		"app": map[string]interface{}{
			"name": "my-app",
			"port": 8080,
		},
	}

	// Initialize config with defaults
	cfg, err := config.New(config.WithDefault(defaults))
	if err != nil {
		fmt.Printf("Failed to initialize config: %v\n", err)
		return
	}

	// Access structured config via ConfigStruct
	cfgStruct := cfg.GetConfigStruct()
	fmt.Printf("Environment: %s\n", cfgStruct.Environment)
	fmt.Printf("Debug: %v\n", cfgStruct.Debug)

	// Access raw values
	env := cfg.GetStringWithDefault("environment", "dev")
	debug := cfg.GetBool("debug")
	appName := cfg.GetStringWithDefault("app.name", "default-app")
	appPort := cfg.Get("app.port") // Raw interface{} for non-string types
	fmt.Printf("Raw Environment: %s\n", env)
	fmt.Printf("Raw Debug: %v\n", debug)
	fmt.Printf("App Name: %s\n", appName)
	fmt.Printf("App Port: %v\n", appPort)
}
