package httpc

import (
	"fmt"
	"reflect"

	"github.com/T-Prohmpossadhorn/go-core/logger"
)

// getServiceInfo extracts method information from a service
func getServiceInfo(service interface{}) ([]MethodInfo, error) {
	if service == nil {
		return nil, fmt.Errorf("service cannot be nil")
	}

	svcType := reflect.TypeOf(service)
	svcValue := reflect.ValueOf(service)

	// Check for RegisterMethods method
	registerMethod, ok := svcType.MethodByName("RegisterMethods")
	if !ok {
		return nil, fmt.Errorf("no RegisterMethods method found")
	}

	// Verify RegisterMethods signature
	if registerMethod.Type.NumIn() != 1 || registerMethod.Type.NumOut() != 1 ||
		registerMethod.Type.Out(0) != reflect.TypeOf([]MethodInfo{}) {
		return nil, fmt.Errorf("invalid RegisterMethods signature")
	}

	// Call RegisterMethods
	results := registerMethod.Func.Call([]reflect.Value{svcValue})
	if len(results) != 1 {
		return nil, fmt.Errorf("RegisterMethods returned unexpected results")
	}

	methods, ok := results[0].Interface().([]MethodInfo)
	if !ok {
		return nil, fmt.Errorf("RegisterMethods did not return []MethodInfo")
	}

	// Validate methods
	for _, method := range methods {
		if method.Name == "" || method.HTTPMethod == "" {
			return nil, fmt.Errorf("invalid MethodInfo: Name or HTTPMethod is empty")
		}
		// Verify method exists and has correct signature
		meth, ok := svcType.MethodByName(method.Name)
		if !ok {
			return nil, fmt.Errorf("method %s not found", method.Name)
		}
		if meth.Type.NumIn() != 2 || meth.Type.NumOut() != 2 ||
			meth.Type.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
			return nil, fmt.Errorf("invalid signature for method %s", method.Name)
		}
		// Set Func field
		method.Func = meth.Func
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no methods defined for service")
	}

	logger.Info("Retrieved methods", "count", len(methods))
	return methods, nil
}
