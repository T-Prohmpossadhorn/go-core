package httpc

import (
	"reflect"
	"strings"
)

// MethodInfo represents a service method's metadata
type MethodInfo struct {
	Name       string
	HTTPMethod string
	InputType  reflect.Type
	OutputType reflect.Type
	Func       reflect.Value // Stores method function
}

// ServiceOption configures service registration
type ServiceOption func(*serviceConfig)

type serviceConfig struct {
	prefix string
}

// WithPathPrefix sets a custom path prefix for endpoints
func WithPathPrefix(prefix string) ServiceOption {
	return func(s *serviceConfig) {
		s.prefix = prefix
	}
}

// isValidHTTPMethod checks if the given method is a valid HTTP method
func isValidHTTPMethod(method string) bool {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}
	for _, valid := range validMethods {
		if strings.ToUpper(method) == valid {
			return true
		}
	}
	return false
}
