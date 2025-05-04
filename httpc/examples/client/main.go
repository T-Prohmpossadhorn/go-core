package main

import (
	"context"
	"fmt"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/httpc"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/T-Prohmpossadhorn/go-core/otel"
)

// User defines the structure for user data
type User struct {
	Name    string `json:"name" validate:"required,min=1,max=50"`
	Address struct {
		City string `json:"city" validate:"required,min=1,max=50"`
	} `json:"address" validate:"required"`
}

func main() {
	// Initialize logger
	if err := logger.Init(); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	// Configure client with otel enabled
	clientCfg := map[string]interface{}{
		"otel_enabled":            true,
		"otel_endpoint":           "localhost:4317",
		"http_client_timeout_ms":  1000,
		"http_client_max_retries": 2,
	}
	cfg, err := config.New(config.WithDefault(clientCfg))
	if err != nil {
		panic("Failed to initialize config: " + err.Error())
	}

	// Initialize otel if enabled
	if cfg.GetBool("otel_enabled") {
		if err := otel.Init(cfg); err != nil {
			panic("Failed to initialize otel: " + err.Error())
		}
		defer otel.Shutdown(context.Background())
	}

	// Create client
	client, err := httpc.NewHTTPClient(cfg)
	if err != nil {
		panic("Failed to initialize HTTP client: " + err.Error())
	}

	// Make GET request
	var greetResult string
	err = client.Call("GET", "http://localhost:8080/v1/Greet?name=World", nil, &greetResult)
	if err != nil {
		fmt.Printf("GET request failed: %v\n", err)
		return
	}
	fmt.Println(greetResult) // Output: Hello, World!

	// Make POST request
	user := User{
		Name: "Alice",
		Address: struct {
			City string `json:"city" validate:"required,min=1,max=50"`
		}{
			City: "Wonderland",
		},
	}
	var createResult string
	err = client.Call("POST", "http://localhost:8080/v1/CreateUser", user, &createResult)
	if err != nil {
		fmt.Printf("POST request failed: %v\n", err)
		return
	}
	fmt.Println(createResult) // Output: Created user Alice
}
