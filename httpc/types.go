package httpc

import (
	"reflect"

	"github.com/T-Prohmpossadhorn/go-core/logger"
)

// MethodInfo represents metadata for a service method
type MethodInfo struct {
	Name        string
	HTTPMethod  string
	Description string
	InputType   reflect.Type
	OutputType  reflect.Type
}

// Option represents a configuration option for service registration
type Option func(*Server)

// WithPathPrefix sets the path prefix for service endpoints
func WithPathPrefix(prefix string) Option {
	return func(s *Server) {
		logger.Info("Applying option", logger.Field{Key: "prefix", Value: prefix})
		s.pathPrefix = prefix
	}
}

// User represents a user input struct for testing
type User struct {
	Name    string `json:"name" validate:"required"`
	Address struct {
		City string `json:"city" validate:"required"`
	} `json:"address"`
}
