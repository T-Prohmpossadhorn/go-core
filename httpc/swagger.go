package httpc

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/T-Prohmpossadhorn/go-core/logger"
)

// updateSwaggerDoc updates the server's Swagger documentation with service methods
func updateSwaggerDoc(s *Server, service interface{}, pathPrefix string) error {
	if s == nil {
		logger.Error("Server cannot be nil")
		return fmt.Errorf("server cannot be nil")
	}

	logger.Info("Starting updateSwaggerDoc")

	svcValue := reflect.ValueOf(service)
	svcType := reflect.TypeOf(service)

	// Check for RegisterMethods
	registerMethods := svcValue.MethodByName("RegisterMethods")
	if !registerMethods.IsValid() {
		logger.Error("No RegisterMethods method found")
		return fmt.Errorf("service must implement RegisterMethods")
	}

	// Call RegisterMethods
	results := registerMethods.Call(nil)
	if len(results) != 1 || results[0].Type() != reflect.TypeOf([]MethodInfo{}) {
		logger.Error("Invalid RegisterMethods signature")
		return fmt.Errorf("invalid RegisterMethods signature")
	}

	methods := results[0].Interface().([]MethodInfo)
	if len(methods) == 0 {
		logger.Error("No methods defined for service")
		return fmt.Errorf("no methods defined for service")
	}

	// Initialize paths if nil
	if s.swagger["paths"] == nil {
		s.swagger["paths"] = make(map[string]interface{})
	}
	paths := s.swagger["paths"].(map[string]interface{})

	// Add each method to Swagger paths
	for _, method := range methods {
		path := fmt.Sprintf("%s/%s", strings.TrimSuffix(pathPrefix, "/"), method.Name)
		if !isValidHTTPMethod(method.HTTPMethod) {
			logger.Warn("Skipping invalid HTTP method", logger.Field{Key: "method", Value: method.HTTPMethod})
			continue
		}

		// Find the service method
		_, exists := svcType.MethodByName(method.Name)
		if !exists {
			logger.Error("Method not found", logger.Field{Key: "method", Value: method.Name})
			return fmt.Errorf("method %s not found in service", method.Name)
		}

		// Create path entry
		pathEntry := map[string]interface{}{
			strings.ToLower(method.HTTPMethod): map[string]interface{}{
				"summary":     method.Description,
				"operationId": method.Name,
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Successful response",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
			},
		}

		// Add parameters for GET methods
		if strings.ToUpper(method.HTTPMethod) == "GET" {
			getMethod := pathEntry["get"].(map[string]interface{})
			getMethod["parameters"] = []map[string]interface{}{
				{
					"name":     "name",
					"in":       "query",
					"required": false,
					"schema": map[string]interface{}{
						"type": "string",
					},
				},
			}
		}

		// Add request body for POST methods
		if strings.ToUpper(method.HTTPMethod) == "POST" {
			postMethod := pathEntry["post"].(map[string]interface{})
			postMethod["requestBody"] = map[string]interface{}{
				"required": true,
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name": map[string]interface{}{
									"type": "string",
								},
								"address": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"city": map[string]interface{}{
											"type": "string",
										},
									},
									"required": []string{"city"},
								},
							},
							"required": []string{"name"},
						},
					},
				},
			}
		}

		paths[path] = pathEntry
	}

	logger.Info("Added methods to Swagger doc", logger.Field{Key: "count", Value: len(methods)})
	logger.Info("Swagger doc updated successfully")
	return nil
}
