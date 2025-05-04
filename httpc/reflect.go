package httpc

import (
	"fmt"
	"reflect"

	"github.com/T-Prohmpossadhorn/go-core/logger"
)

// getServiceInfo retrieves method information from a service
func getServiceInfo(svc interface{}) ([]MethodInfo, error) {
	logger.Info("Starting getServiceInfo")
	defer logger.Info("getServiceInfo completed")

	if svc == nil {
		logger.Error("Service cannot be nil")
		return nil, fmt.Errorf("service cannot be nil")
	}

	svcValue := reflect.ValueOf(svc)
	svcType := reflect.TypeOf(svc)

	// Handle non-pointer types by converting to pointer
	if svcType.Kind() != reflect.Ptr {
		svcValue = reflect.New(svcType).Elem()
		svcValue.Set(reflect.ValueOf(svc))
		svcValue = svcValue.Addr()
	}

	registerMethods := svcValue.MethodByName("RegisterMethods")
	if !registerMethods.IsValid() {
		logger.Error("No RegisterMethods method found")
		return nil, fmt.Errorf("service must implement RegisterMethods")
	}

	methodsVal := registerMethods.Call(nil)
	if len(methodsVal) != 1 || methodsVal[0].Type() != reflect.TypeOf([]MethodInfo{}) {
		logger.Error("Invalid RegisterMethods signature")
		return nil, fmt.Errorf("RegisterMethods must return []MethodInfo")
	}

	methods := methodsVal[0].Interface().([]MethodInfo)
	logger.Info("Retrieved methods", logger.Field{Key: "count", Value: len(methods)})
	if len(methods) == 0 {
		logger.Error("No methods defined for service")
		return nil, fmt.Errorf("no methods defined for service")
	}

	// Validate MethodInfo fields
	for _, method := range methods {
		if method.Name == "" || method.HTTPMethod == "" {
			logger.Error("Invalid MethodInfo: Name or HTTPMethod is empty")
			return nil, fmt.Errorf("invalid MethodInfo: Name or HTTPMethod is empty")
		}
	}

	return methods, nil
}
