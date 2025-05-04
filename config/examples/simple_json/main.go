package main

import (
	"fmt"

	"github.com/T-Prohmpossadhorn/go-core/config"
)

// Sample config.json:
// ```json
// {
//   "environment": "production",
//   "debug": false,
//   "app": {
//     "name": "my-app",
//     "port": 9090
//   }
// }
// ```

func main() {
	// Initialize config with JSON file
	cfg, err := config.New(config.WithFilepath("config.json"))
	if err != nil {
		fmt.Printf("Failed to initialize config: %v\n", err)
		return
	}

	// Access raw values
	env := cfg.GetStringWithDefault("environment", "development")
	debug := cfg.GetBool("debug")
	appName := cfg.GetStringWithDefault("app.name", "default-app")
	appPort := cfg.Get("app.port")
	fmt.Printf("Environment: %s\n", env)
	fmt.Printf("Debug: %v\n", debug)
	fmt.Printf("App Name: %s\n", appName)
	fmt.Printf("App Port: %v\n", appPort)

	// Unmarshal entire configuration into a custom struct
	type AppConfig struct {
		Environment string `mapstructure:"environment"`
		Debug       bool   `mapstructure:"debug"`
		App         struct {
			Name string `mapstructure:"name"`
			Port int    `mapstructure:"port"`
		} `mapstructure:"app"`
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
