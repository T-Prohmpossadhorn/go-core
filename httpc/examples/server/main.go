package main

import (
	"context"
	"reflect"

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

// SampleService defines a sample service with HTTP endpoints
type SampleService struct{}

// Greet handles GET requests to return a greeting
func (s *SampleService) Greet(name string) (string, error) {
	return "Hello, " + name + "!", nil
}

// CreateUser handles POST requests to create a user
func (s *SampleService) CreateUser(user User) (string, error) {
	return "Created user " + user.Name, nil
}

// RegisterMethods defines the service's HTTP endpoints
func (s *SampleService) RegisterMethods() []httpc.MethodInfo {
	return []httpc.MethodInfo{
		{
			Name:       "Greet",
			HTTPMethod: "GET",
			InputType:  reflect.TypeOf(""),
			OutputType: reflect.TypeOf(""),
		},
		{
			Name:       "CreateUser",
			HTTPMethod: "POST",
			InputType:  reflect.TypeOf(User{}),
			OutputType: reflect.TypeOf(""),
		},
	}
}

func main() {
	// Initialize logger
	if err := logger.Init(); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	// Configure server with otel enabled
	serverCfg := map[string]interface{}{
		"otel_enabled":  true,
		"otel_endpoint": "localhost:4317",
		"port":          8080,
	}
	cfg, err := config.New(config.WithDefault(serverCfg))
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

	// Create server
	server, err := httpc.NewServer(cfg)
	if err != nil {
		panic("Failed to initialize HTTP server: " + err.Error())
	}

	// Register service (both pointer and non-pointer supported)
	if err := server.RegisterService(SampleService{}, httpc.WithPathPrefix("/v1")); err != nil {
		panic("Failed to register service: " + err.Error())
	}

	// Start server
	if err := server.ListenAndServe(); err != nil {
		panic("Failed to start server: " + err.Error())
	}
}
