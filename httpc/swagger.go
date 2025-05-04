package httpc

import (
	"fmt"
	"reflect"
	"strings"
)

// generateSchema generates a Swagger schema for a given type
func generateSchema(t reflect.Type) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return map[string]interface{}{
			"type": t.Kind().String(),
		}
	}

	properties := schema["properties"].(map[string]interface{})
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		jsonName := strings.Split(jsonTag, ",")[0]
		validateTag := field.Tag.Get("validate")
		fieldSchema := map[string]interface{}{}

		switch field.Type.Kind() {
		case reflect.String:
			fieldSchema["type"] = "string"
			if strings.Contains(validateTag, "min=") {
				for _, part := range strings.Split(validateTag, ",") {
					if strings.HasPrefix(part, "min=") {
						if min, err := parseInt(strings.TrimPrefix(part, "min=")); err == nil {
							fieldSchema["minLength"] = float64(min)
						}
					}
				}
			}
			if strings.Contains(validateTag, "max=") {
				for _, part := range strings.Split(validateTag, ",") {
					if strings.HasPrefix(part, "max=") {
						if max, err := parseInt(strings.TrimPrefix(part, "max=")); err == nil {
							fieldSchema["maxLength"] = float64(max)
						}
					}
				}
			}
			if strings.Contains(validateTag, "email") {
				fieldSchema["format"] = "email"
			}
		case reflect.Int, reflect.Int32, reflect.Int64:
			fieldSchema["type"] = "integer"
			if strings.Contains(validateTag, "gte=") {
				for _, part := range strings.Split(validateTag, ",") {
					if strings.HasPrefix(part, "gte=") {
						if min, err := parseInt(strings.TrimPrefix(part, "gte=")); err == nil {
							fieldSchema["minimum"] = float64(min)
						}
					}
				}
			}
			if strings.Contains(validateTag, "lte=") {
				for _, part := range strings.Split(validateTag, ",") {
					if strings.HasPrefix(part, "lte=") {
						if max, err := parseInt(strings.TrimPrefix(part, "lte=")); err == nil {
							fieldSchema["maximum"] = float64(max)
						}
					}
				}
			}
		case reflect.Float32, reflect.Float64:
			fieldSchema["type"] = "number"
			if strings.Contains(validateTag, "gte=") {
				for _, part := range strings.Split(validateTag, ",") {
					if strings.HasPrefix(part, "gte=") {
						if min, err := parseFloat(strings.TrimPrefix(part, "gte=")); err == nil {
							fieldSchema["minimum"] = min
						}
					}
				}
			}
		case reflect.Struct:
			fieldSchema = generateSchema(field.Type)
		}

		if strings.Contains(validateTag, "required") {
			required = append(required, jsonName)
		}

		properties[jsonName] = fieldSchema
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// parseInt is a helper function to parse string to int
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// parseFloat is a helper function to parse string to float64
func parseFloat(s string) (float64, error) {
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}

// updateSwaggerDoc updates the Swagger documentation for the given service
func updateSwaggerDoc(s *Server, service interface{}, prefix string) error {
	if s == nil {
		return fmt.Errorf("server cannot be nil")
	}

	// Initialize swagger if not already set or missing required fields
	if s.swagger == nil || s.swagger["openapi"] == nil || s.swagger["info"] == nil {
		s.swagger = map[string]interface{}{
			"openapi": "3.0.3",
			"info": map[string]interface{}{
				"title":   "httpc API",
				"version": "1.0.0",
			},
			"paths": map[string]interface{}{},
		}
	}

	info, err := getServiceInfo(service)
	if err != nil {
		return err
	}

	paths := s.swagger["paths"].(map[string]interface{})
	for _, method := range info {
		// Skip invalid HTTP methods
		if !isValidHTTPMethod(method.HTTPMethod) {
			continue
		}

		path := prefix + "/" + method.Name
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		pathItem := map[string]interface{}{}
		if existing, ok := paths[path]; ok {
			pathItem = existing.(map[string]interface{})
		}

		operation := map[string]interface{}{
			"operationId": method.Name,
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Successful response",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"type": method.OutputType.Kind().String(),
							},
						},
					},
				},
				"400": map[string]interface{}{
					"description": "Bad request",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"error": map[string]interface{}{
										"type": "string",
									},
								},
							},
						},
					},
				},
				"500": map[string]interface{}{
					"description": "Internal server error",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"error": map[string]interface{}{
										"type": "string",
									},
								},
							},
						},
					},
				},
			},
			"summary": method.Name,
		}

		if method.HTTPMethod == "GET" {
			operation["parameters"] = []map[string]interface{}{
				{
					"name":     "name",
					"in":       "query",
					"required": false,
					"schema": map[string]interface{}{
						"type": "string",
					},
				},
			}
		} else {
			// POST, PUT, DELETE, PATCH, OPTIONS, HEAD
			schema := generateSchema(method.InputType)
			operation["requestBody"] = map[string]interface{}{
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": schema,
					},
				},
				"required": true,
			}
		}

		pathItem[strings.ToLower(method.HTTPMethod)] = operation
		paths[path] = pathItem
	}

	return nil
}
