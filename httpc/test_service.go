package httpc

import (
	"fmt"
	"reflect"

	"github.com/T-Prohmpossadhorn/go-core/logger"
)

// TestService is a test service for HTTP endpoints
type TestService struct{}

// RegisterMethods returns the methods for TestService
func (s *TestService) RegisterMethods() []MethodInfo {
	return []MethodInfo{
		{
			Name:        "Hello",
			HTTPMethod:  "GET",
			Description: "Returns a greeting",
			InputType:   reflect.TypeOf(struct{ Name string }{}),
			OutputType:  reflect.TypeOf(""),
		},
		{
			Name:        "Create",
			HTTPMethod:  "POST",
			Description: "Creates a user",
			InputType:   reflect.TypeOf(User{}),
			OutputType:  reflect.TypeOf(""),
		},
	}
}

// Hello handles the GET /Hello endpoint
func (s *TestService) Hello(name string) (string, error) {
	logger.Info("Handling Hello request", logger.Field{Key: "name", Value: name})
	return fmt.Sprintf("Hello, %s!", name), nil
}

// Create handles the POST /Create endpoint
func (s *TestService) Create(user User) (string, error) {
	logger.Info("Handling Create request", logger.Field{Key: "user", Value: user.Name})
	return fmt.Sprintf("Created user %s", user.Name), nil
}

// InvalidSigService is a test service with an invalid method signature
type InvalidSigService struct{}

// RegisterMethods returns an invalid method signature for testing
func (s *InvalidSigService) RegisterMethods() []MethodInfo {
	return []MethodInfo{
		{
			Name:        "BadMethod",
			HTTPMethod:  "GET",
			Description: "Invalid method",
			InputType:   reflect.TypeOf(""),
			OutputType:  reflect.TypeOf(""),
		},
	}
}

// BadMethod has an invalid signature
func (s *InvalidSigService) BadMethod() string {
	return "Invalid"
}
