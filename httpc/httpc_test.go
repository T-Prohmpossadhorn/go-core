package httpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/T-Prohmpossadhorn/go-core/otel"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

// errorReader simulates a read error for response body
type errorReader struct{}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}

// customTransport for injecting errorReader into response body
type customTransport struct {
	handler func(*http.Request) (*http.Response, error)
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.handler(req)
}

func setupServer(t *testing.T, cfg ServerConfig, svc interface{}, prefix string) (*Server, *httptest.Server) {
	if err := validateConfig(&cfg); err != nil {
		t.Fatalf("Server config validation failed: %v", err)
	}
	cfgMap, err := toConfigMap(cfg)
	assert.NoError(t, err)
	config, err := config.New(config.WithDefault(cfgMap))
	assert.NoError(t, err)

	server, err := NewServer(config)
	assert.NoError(t, err)
	err = server.RegisterService(svc, WithPathPrefix(prefix))
	assert.NoError(t, err)
	ts := httptest.NewServer(server.engine)
	return server, ts
}

// ProductService for testing
type Product struct {
	ID    int     `json:"id" validate:"required,gte=1"`
	Name  string  `json:"name" validate:"required,min=1,max=100"`
	Price float64 `json:"price" validate:"gte=0"`
}

type ProductService struct{}

func (s *ProductService) Create(product Product) (string, error) {
	return fmt.Sprintf("Created product %s", product.Name), nil
}

func (s *ProductService) RegisterMethods() []MethodInfo {
	return []MethodInfo{
		{
			Name:       "Create",
			HTTPMethod: "POST",
			InputType:  reflect.TypeOf(Product{}),
			OutputType: reflect.TypeOf(""),
		},
	}
}

// CustomerService for testing
type Address struct {
	Street string `json:"street" validate:"required,min=1,max=200"`
	City   string `json:"city" validate:"required,min=1,max=100"`
}

type Customer struct {
	Email   string  `json:"email" validate:"required,email"`
	Age     int     `json:"age" validate:"gte=18,lte=120"`
	Address Address `json:"address" validate:"required"`
}

type CustomerService struct{}

func (s *CustomerService) Create(customer Customer) (string, error) {
	return fmt.Sprintf("Created customer %s", customer.Email), nil
}

func (s *CustomerService) RegisterMethods() []MethodInfo {
	return []MethodInfo{
		{
			Name:       "Create",
			HTTPMethod: "POST",
			InputType:  reflect.TypeOf(Customer{}),
			OutputType: reflect.TypeOf(""),
		},
	}
}

func TestHTTPC(t *testing.T) {
	os.Setenv("CONFIG_LOGGER_LEVEL", "info")
	if err := logger.Init(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	serverCfg := ServerConfig{
		OtelEnabled: false,
		Port:        8080,
	}
	clientCfg := ClientConfig{
		OtelEnabled:          false,
		HTTPClientTimeoutMs:  1000,
		HTTPClientMaxRetries: 2,
	}
	clientCfgMap, err := toConfigMap(clientCfg)
	assert.NoError(t, err)
	clientDefaultCfg, err := config.New(config.WithDefault(clientCfgMap))
	assert.NoError(t, err)

	t.Run("Valid Swagger Doc Generation", func(t *testing.T) {
		svc := &TestService{}
		_, ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/docs/swagger.json")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Read raw response for debugging
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		t.Logf("Swagger JSON response: %s", string(body))

		var doc map[string]interface{}
		err = json.Unmarshal(body, &doc)
		assert.NoError(t, err)
		assert.NotNil(t, doc)
		assert.Equal(t, "3.0.3", doc["openapi"])
		paths, ok := doc["paths"].(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, paths, "/v1/Hello")
	})

	t.Run("Dynamic Swagger Schema Generation", func(t *testing.T) {
		t.Run("Product Service", func(t *testing.T) {
			svc := &ProductService{}
			_, ts := setupServer(t, serverCfg, svc, "/v1")
			defer ts.Close()

			resp, err := http.Get(ts.URL + "/api/docs/swagger.json")
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var doc map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&doc)
			assert.NoError(t, err)

			paths, ok := doc["paths"].(map[string]interface{})
			assert.True(t, ok)
			assert.Contains(t, paths, "/v1/Create")

			// Verify POST /v1/Create
			createPath, ok := paths["/v1/Create"].(map[string]interface{})
			assert.True(t, ok)
			postMethod, ok := createPath["post"].(map[string]interface{})
			assert.True(t, ok)
			requestBody, ok := postMethod["requestBody"].(map[string]interface{})
			assert.True(t, ok)
			content, ok := requestBody["content"].(map[string]interface{})
			assert.True(t, ok)
			jsonContent, ok := content["application/json"].(map[string]interface{})
			assert.True(t, ok)
			schema, ok := jsonContent["schema"].(map[string]interface{})
			assert.True(t, ok)
			properties, ok := schema["properties"].(map[string]interface{})
			assert.True(t, ok)

			idProp, ok := properties["id"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "integer", idProp["type"])
			assert.Equal(t, float64(1), idProp["minimum"])

			nameProp, ok := properties["name"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "string", nameProp["type"])
			assert.Equal(t, float64(1), nameProp["minLength"])
			assert.Equal(t, float64(100), nameProp["maxLength"])

			priceProp, ok := properties["price"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "number", priceProp["type"])
			assert.Equal(t, float64(0), priceProp["minimum"])

			required, ok := schema["required"].([]interface{})
			assert.True(t, ok)
			assert.Contains(t, required, "id")
			assert.Contains(t, required, "name")
		})

		t.Run("Customer Service", func(t *testing.T) {
			svc := &CustomerService{}
			_, ts := setupServer(t, serverCfg, svc, "/v2")
			defer ts.Close()

			resp, err := http.Get(ts.URL + "/api/docs/swagger.json")
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var doc map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&doc)
			assert.NoError(t, err)

			paths, ok := doc["paths"].(map[string]interface{})
			assert.True(t, ok)
			assert.Contains(t, paths, "/v2/Create")

			createPath, ok := paths["/v2/Create"].(map[string]interface{})
			assert.True(t, ok)
			postMethod, ok := createPath["post"].(map[string]interface{})
			assert.True(t, ok)
			requestBody, ok := postMethod["requestBody"].(map[string]interface{})
			assert.True(t, ok)
			content, ok := requestBody["content"].(map[string]interface{})
			assert.True(t, ok)
			jsonContent, ok := content["application/json"].(map[string]interface{})
			assert.True(t, ok)
			schema, ok := jsonContent["schema"].(map[string]interface{})
			assert.True(t, ok)
			properties, ok := schema["properties"].(map[string]interface{})
			assert.True(t, ok)

			emailProp, ok := properties["email"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "string", emailProp["type"])
			assert.Equal(t, "email", emailProp["format"])

			ageProp, ok := properties["age"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "integer", ageProp["type"])
			assert.Equal(t, float64(18), ageProp["minimum"])
			assert.Equal(t, float64(120), ageProp["maximum"])

			addressProp, ok := properties["address"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "object", addressProp["type"])
			addressProps, ok := addressProp["properties"].(map[string]interface{})
			assert.True(t, ok)

			streetProp, ok := addressProps["street"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "string", streetProp["type"])
			assert.Equal(t, float64(1), streetProp["minLength"])
			assert.Equal(t, float64(200), streetProp["maxLength"])

			cityProp, ok := addressProps["city"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "string", cityProp["type"])
			assert.Equal(t, float64(1), cityProp["minLength"])
			assert.Equal(t, float64(100), cityProp["maxLength"])

			addressRequired, ok := addressProp["required"].([]interface{})
			assert.True(t, ok)
			assert.Contains(t, addressRequired, "street")
			assert.Contains(t, addressRequired, "city")

			required, ok := schema["required"].([]interface{})
			assert.True(t, ok)
			assert.Contains(t, required, "email")
			assert.Contains(t, required, "address")
		})
	})

	t.Run("Retry Backoff", func(t *testing.T) {
		svc := &TestService{}
		_, ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		client, err := NewHTTPClient(clientDefaultCfg)
		assert.NoError(t, err)

		var result string
		err = client.Call("GET", ts.URL+"/v1/Hello?name=Test", nil, &result)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, Test!", result)
	})

	t.Run("Invalid HTTP Method", func(t *testing.T) {
		t.Parallel()
		client, err := NewHTTPClient(clientDefaultCfg)
		assert.NoError(t, err)

		var result string
		err = client.Call("INVALID", "http://example.com", nil, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid HTTP method")
	})

	t.Run("POST Create User", func(t *testing.T) {
		svc := &TestService{}
		_, ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		client, err := NewHTTPClient(clientDefaultCfg)
		assert.NoError(t, err)

		user := User{
			Name: "TestUser",
			Address: struct {
				City string `json:"city" validate:"required"`
			}{
				City: "TestCity",
			},
		}
		var result string
		err = client.Call("POST", ts.URL+"/v1/Create", user, &result)
		assert.NoError(t, err)
		assert.Equal(t, "Created user TestUser", result)
	})

	t.Run("Invalid JSON Response", func(t *testing.T) {
		t.Parallel()
		serverCfgMap, err := toConfigMap(serverCfg)
		assert.NoError(t, err)
		serverConfig, err := config.New(config.WithDefault(serverCfgMap))
		assert.NoError(t, err)
		server, err := NewServer(serverConfig)
		assert.NoError(t, err)

		server.engine.POST("/v1/TestInvalidJSON", func(c *gin.Context) {
			c.String(http.StatusOK, "invalid json")
		})

		ts := httptest.NewServer(server.engine)
		defer ts.Close()

		client, err := NewHTTPClient(clientDefaultCfg)
		assert.NoError(t, err)

		user := User{
			Name: "TestUser",
			Address: struct {
				City string `json:"city" validate:"required"`
			}{
				City: "TestCity",
			},
		}
		var result string
		err = client.Call("POST", ts.URL+"/v1/TestInvalidJSON", user, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse response JSON")
	})

	t.Run("Non-JSON Error Response", func(t *testing.T) {
		t.Parallel()
		serverCfgMap, err := toConfigMap(serverCfg)
		assert.NoError(t, err)
		serverConfig, err := config.New(config.WithDefault(serverCfgMap))
		assert.NoError(t, err)
		server, err := NewServer(serverConfig)
		assert.NoError(t, err)

		server.engine.POST("/v1/TestNonJSONError", func(c *gin.Context) {
			c.String(http.StatusBadRequest, "plain text error")
		})

		ts := httptest.NewServer(server.engine)
		defer ts.Close()

		retryCfg := ClientConfig{
			OtelEnabled:          false,
			HTTPClientTimeoutMs:  1000,
			HTTPClientMaxRetries: 0,
		}
		retryCfgMap, err := toConfigMap(retryCfg)
		assert.NoError(t, err)
		retryConfig, err := config.New(config.WithDefault(retryCfgMap))
		assert.NoError(t, err)

		client, err := NewHTTPClient(retryConfig)
		assert.NoError(t, err)

		var result string
		err = client.Call("POST", ts.URL+"/v1/TestNonJSONError", nil, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request failed with status 400")
	})

	t.Run("Failed Response Body Read", func(t *testing.T) {
		serverCfgMap, err := toConfigMap(serverCfg)
		assert.NoError(t, err)
		serverConfig, err := config.New(config.WithDefault(serverCfgMap))
		assert.NoError(t, err)
		server, err := NewServer(serverConfig)
		assert.NoError(t, err)

		server.engine.POST("/v1/TestReadError", func(c *gin.Context) {
			c.Status(http.StatusOK)
			c.Writer.Write([]byte("partial"))
		})

		ts := httptest.NewServer(server.engine)
		defer ts.Close()

		retryCfg := ClientConfig{
			OtelEnabled:          false,
			HTTPClientTimeoutMs:  1000,
			HTTPClientMaxRetries: 0,
		}
		retryCfgMap, err := toConfigMap(retryCfg)
		assert.NoError(t, err)
		retryConfig, err := config.New(config.WithDefault(retryCfgMap))
		assert.NoError(t, err)

		client, err := NewHTTPClient(retryConfig)
		assert.NoError(t, err)

		customClient := &http.Client{
			Transport: &customTransport{
				handler: func(req *http.Request) (*http.Response, error) {
					resp, err := http.DefaultTransport.RoundTrip(req)
					if err != nil {
						return nil, err
					}
					resp.Body = io.NopCloser(&errorReader{})
					return resp, nil
				},
			},
		}
		client.client = customClient

		var result string
		err = client.Call("POST", ts.URL+"/v1/TestReadError", nil, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "simulated read error")
	})

	t.Run("Server Error Status", func(t *testing.T) {
		serverCfgMap, err := toConfigMap(serverCfg)
		assert.NoError(t, err)
		serverConfig, err := config.New(config.WithDefault(serverCfgMap))
		assert.NoError(t, err)
		server, err := NewServer(serverConfig)
		assert.NoError(t, err)

		server.engine.POST("/v1/TestServerError", func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		})

		ts := httptest.NewServer(server.engine)
		defer ts.Close()

		retryCfg := ClientConfig{
			OtelEnabled:          false,
			HTTPClientTimeoutMs:  1000,
			HTTPClientMaxRetries: 0,
		}
		retryCfgMap, err := toConfigMap(retryCfg)
		assert.NoError(t, err)
		retryConfig, err := config.New(config.WithDefault(retryCfgMap))
		assert.NoError(t, err)

		client, err := NewHTTPClient(retryConfig)
		assert.NoError(t, err)

		user := User{
			Name: "TestUser",
			Address: struct {
				City string `json:"city" validate:"required"`
			}{
				City: "TestCity",
			},
		}
		var result string
		err = client.Call("POST", ts.URL+"/v1/TestServerError", user, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request failed with status 500")
	})

	t.Run("Invalid User Validation", func(t *testing.T) {
		svc := &TestService{}
		_, ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		retryCfg := ClientConfig{
			OtelEnabled:          false,
			HTTPClientTimeoutMs:  1000,
			HTTPClientMaxRetries: 0,
		}
		retryCfgMap, err := toConfigMap(retryCfg)
		assert.NoError(t, err)
		retryConfig, err := config.New(config.WithDefault(retryCfgMap))
		assert.NoError(t, err)

		client, err := NewHTTPClient(retryConfig)
		assert.NoError(t, err)

		user := User{
			Name: "", // Invalid: Name is required
			Address: struct {
				City string `json:"city" validate:"required"`
			}{
				City: "TestCity",
			},
		}
		var result string
		err = client.Call("POST", ts.URL+"/v1/Create", user, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("Context Cancellation", func(t *testing.T) {
		serverCfgMap, err := toConfigMap(serverCfg)
		assert.NoError(t, err)
		serverConfig, err := config.New(config.WithDefault(serverCfgMap))
		assert.NoError(t, err)
		server, err := NewServer(serverConfig)
		assert.NoError(t, err)

		svc := &TestService{}
		err = server.RegisterService(svc, WithPathPrefix("/v1"))
		assert.NoError(t, err)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
		}))
		defer ts.Close()

		retryCfg := ClientConfig{
			OtelEnabled:          false,
			HTTPClientTimeoutMs:  1000,
			HTTPClientMaxRetries: 0,
		}
		retryCfgMap, err := toConfigMap(retryCfg)
		assert.NoError(t, err)
		retryConfig, err := config.New(config.WithDefault(retryCfgMap))
		assert.NoError(t, err)

		client, err := NewHTTPClient(retryConfig)
		assert.NoError(t, err)

		var result string
		err = client.Call("GET", ts.URL+"/v1/Slow", nil, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "EOF")
	})

	t.Run("Network Error", func(t *testing.T) {
		retryCfg := ClientConfig{
			OtelEnabled:          false,
			HTTPClientTimeoutMs:  1000,
			HTTPClientMaxRetries: 0,
		}
		retryCfgMap, err := toConfigMap(retryCfg)
		assert.NoError(t, err)
		retryConfig, err := config.New(config.WithDefault(retryCfgMap))
		assert.NoError(t, err)

		client, err := NewHTTPClient(retryConfig)
		assert.NoError(t, err)

		var result string
		err = client.Call("GET", "http://invalid-host:9999", nil, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dial")
	})

	t.Run("Retry Exhaustion", func(t *testing.T) {
		serverCfgMap, err := toConfigMap(serverCfg)
		assert.NoError(t, err)
		serverConfig, err := config.New(config.WithDefault(serverCfgMap))
		assert.NoError(t, err)
		server, err := NewServer(serverConfig)
		assert.NoError(t, err)

		server.engine.GET("/v1/Fail", func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "unavailable"})
		})

		ts := httptest.NewServer(server.engine)
		defer ts.Close()

		retryCfg := ClientConfig{
			OtelEnabled:          false,
			HTTPClientTimeoutMs:  1000,
			HTTPClientMaxRetries: 0,
		}
		retryCfgMap, err := toConfigMap(retryCfg)
		assert.NoError(t, err)
		retryConfig, err := config.New(config.WithDefault(retryCfgMap))
		assert.NoError(t, err)

		client, err := NewHTTPClient(retryConfig)
		assert.NoError(t, err)

		var result string
		err = client.Call("GET", ts.URL+"/v1/Fail", nil, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request failed with status 503")
	})

	t.Run("Timeout Error", func(t *testing.T) {
		serverCfgMap, err := toConfigMap(serverCfg)
		assert.NoError(t, err)
		serverConfig, err := config.New(config.WithDefault(serverCfgMap))
		assert.NoError(t, err)
		server, err := NewServer(serverConfig)
		assert.NoError(t, err)

		server.engine.GET("/v1/Delay", func(c *gin.Context) {
			time.Sleep(100 * time.Millisecond)
			c.JSON(http.StatusOK, gin.H{"error": "delayed"})
		})

		ts := httptest.NewServer(server.engine)
		defer ts.Close()

		timeoutCfg := ClientConfig{
			OtelEnabled:          false,
			HTTPClientTimeoutMs:  50,
			HTTPClientMaxRetries: 0,
		}
		timeoutCfgMap, err := toConfigMap(timeoutCfg)
		assert.NoError(t, err)
		timeoutConfig, err := config.New(config.WithDefault(timeoutCfgMap))
		assert.NoError(t, err)

		client, err := NewHTTPClient(timeoutConfig)
		assert.NoError(t, err)

		var result string
		err = client.Call("GET", ts.URL+"/v1/Delay", nil, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("Malformed JSON Input", func(t *testing.T) {
		svc := &TestService{}
		_, ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		retryCfg := ClientConfig{
			OtelEnabled:          false,
			HTTPClientTimeoutMs:  1000,
			HTTPClientMaxRetries: 0,
		}
		retryCfgMap, err := toConfigMap(retryCfg)
		assert.NoError(t, err)
		retryConfig, err := config.New(config.WithDefault(retryCfgMap))
		assert.NoError(t, err)

		client, err := NewHTTPClient(retryConfig)
		assert.NoError(t, err)

		input := "invalid json"
		var result string
		err = client.Call("POST", ts.URL+"/v1/Create", input, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid request body")
	})

	t.Run("Backoff Error", func(t *testing.T) {
		t.Parallel()
		backoffCfg := ClientConfig{
			OtelEnabled:          false,
			HTTPClientTimeoutMs:  1000,
			HTTPClientMaxRetries: -1,
		}
		backoffCfgMap, err := toConfigMap(backoffCfg)
		assert.NoError(t, err)
		backoffConfig, err := config.New(config.WithDefault(backoffCfgMap))
		assert.NoError(t, err)

		client, err := NewHTTPClient(backoffConfig)
		assert.NoError(t, err)
		var result string
		err = client.Call("GET", "http://invalid-host:9999", nil, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dial")
	})

	t.Run("Panic in RegisterService", func(t *testing.T) {
		serverCfgMap, err := toConfigMap(serverCfg)
		assert.NoError(t, err)
		serverConfig, err := config.New(config.WithDefault(serverCfgMap))
		assert.NoError(t, err)
		server, err := NewServer(serverConfig)
		assert.NoError(t, err)
		svc := &InvalidSigService{}
		err = server.RegisterService(svc, WithPathPrefix("/v1"))
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "invalid signature")
		}
	})

	t.Run("Empty Query Parameter", func(t *testing.T) {
		t.Parallel()
		svc := &TestService{}
		_, ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		client, err := NewHTTPClient(clientDefaultCfg)
		assert.NoError(t, err)

		var result string
		err = client.Call("GET", ts.URL+"/v1/Hello?name=", nil, &result)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, !", result)
	})

	t.Run("Missing Query Parameter", func(t *testing.T) {
		t.Parallel()
		svc := &TestService{}
		_, ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		client, err := NewHTTPClient(clientDefaultCfg)
		assert.NoError(t, err)

		var result string
		err = client.Call("GET", ts.URL+"/v1/Hello", nil, &result)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, !", result)
	})

	t.Run("Invalid Query Parameters", func(t *testing.T) {
		t.Parallel()
		serverCfgMap, err := toConfigMap(serverCfg)
		assert.NoError(t, err)
		serverConfig, err := config.New(config.WithDefault(serverCfgMap))
		assert.NoError(t, err)
		server, err := NewServer(serverConfig)
		assert.NoError(t, err)

		server.engine.GET("/v1/TestInvalidQuery", func(c *gin.Context) {
			type queryInput struct {
				Name string `form:"name" validate:"required,min=1"`
			}
			var input queryInput
			validator := validator.New()
			if err := c.ShouldBindQuery(&input); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
				return
			}
			if err := validator.Struct(input); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters"})
				return
			}
			c.JSON(http.StatusOK, "Success")
		})

		ts := httptest.NewServer(server.engine)
		defer ts.Close()

		client, err := NewHTTPClient(clientDefaultCfg)
		assert.NoError(t, err)

		var result string
		err = client.Call("GET", ts.URL+"/v1/TestInvalidQuery", nil, &result)
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "invalid query parameters")
		}
	})

	t.Run("Invalid Input Marshaling", func(t *testing.T) {
		svc := &TestService{}
		_, ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		client, err := NewHTTPClient(clientDefaultCfg)
		assert.NoError(t, err)

		input := func() {}
		var result string
		err = client.Call("POST", ts.URL+"/v1/Create", input, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal JSON input")
	})

	t.Run("Invalid JSON Binding", func(t *testing.T) {
		svc := &TestService{}
		_, ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		client, err := NewHTTPClient(clientDefaultCfg)
		assert.NoError(t, err)

		input := map[string]interface{}{"name": make(chan int)}
		var result string
		err = client.Call("POST", ts.URL+"/v1/Create", input, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal JSON input")
	})

	t.Run("Tracing Enabled", func(t *testing.T) {
		os.Setenv("OTEL_TEST_MOCK_EXPORTER", "true")
		defer os.Unsetenv("OTEL_TEST_MOCK_EXPORTER")

		serverCfg.OtelEnabled = true
		svc := &TestService{}
		_, ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		clientCfg.OtelEnabled = true
		clientCfgMap, err := toConfigMap(clientCfg)
		assert.NoError(t, err)
		clientConfig, err := config.New(config.WithDefault(clientCfgMap))
		assert.NoError(t, err)

		client, err := NewHTTPClient(clientConfig)
		assert.NoError(t, err)

		var result string
		err = client.Call("GET", ts.URL+"/v1/Hello?name=Test", nil, &result)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, Test!", result)

		err = client.Call("POST", ts.URL+"/v1/Create", User{
			Name: "TestUser",
			Address: struct {
				City string `json:"city" validate:"required"`
			}{
				City: "TestCity",
			},
		}, &result)
		assert.NoError(t, err)
		assert.Equal(t, "Created user TestUser", result)
	})

	t.Run("Tracing Enabled Error Case", func(t *testing.T) {
		os.Setenv("OTEL_TEST_MOCK_EXPORTER", "true")
		defer os.Unsetenv("OTEL_TEST_MOCK_EXPORTER")

		serverCfg.OtelEnabled = true
		svc := &TestService{}
		_, ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		clientCfg.OtelEnabled = true
		clientCfgMap, err := toConfigMap(clientCfg)
		assert.NoError(t, err)
		clientConfig, err := config.New(config.WithDefault(clientCfgMap))
		assert.NoError(t, err)

		client, err := NewHTTPClient(clientConfig)
		assert.NoError(t, err)

		var result string
		err = client.Call("POST", ts.URL+"/v1/Create", "invalid json", &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid request body")
	})

	t.Run("Tracing Enabled Initialized", func(t *testing.T) {
		os.Setenv("OTEL_TEST_MOCK_EXPORTER", "true")
		defer os.Unsetenv("OTEL_TEST_MOCK_EXPORTER")

		serverCfg.OtelEnabled = true
		serverCfgMap, err := toConfigMap(serverCfg)
		assert.NoError(t, err)
		serverConfig, err := config.New(config.WithDefault(serverCfgMap))
		assert.NoError(t, err)

		err = otel.Init(serverConfig)
		assert.NoError(t, err)
		defer otel.Shutdown(context.Background())

		svc := &TestService{}
		_, ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		clientCfg.OtelEnabled = true
		clientCfgMap, err := toConfigMap(clientCfg)
		assert.NoError(t, err)
		clientConfig, err := config.New(config.WithDefault(clientCfgMap))
		assert.NoError(t, err)

		err = otel.Init(clientConfig)
		assert.NoError(t, err)
		defer otel.Shutdown(context.Background())

		client, err := NewHTTPClient(clientConfig)
		assert.NoError(t, err)

		var result string
		err = client.Call("GET", ts.URL+"/v1/Hello?name=Test", nil, &result)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, Test!", result)

		err = client.Call("POST", ts.URL+"/v1/Create", User{
			Name: "TestUser",
			Address: struct {
				City string `json:"city" validate:"required"`
			}{
				City: "TestCity",
			},
		}, &result)
		assert.NoError(t, err)
		assert.Equal(t, "Created user TestUser", result)
	})

	t.Run("Invalid Config Timeout", func(t *testing.T) {
		invalidCfg := ClientConfig{
			OtelEnabled:          false,
			HTTPClientTimeoutMs:  -1, // Invalid: must be > 0
			HTTPClientMaxRetries: 0,
		}
		cfgMap, err := toConfigMap(invalidCfg)
		assert.NoError(t, err)
		config, err := config.New(config.WithDefault(cfgMap))
		assert.NoError(t, err)

		_, err = NewHTTPClient(config)
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "http_client_timeout_ms must be greater than 0")
		}
	})

	t.Run("Invalid Config Port", func(t *testing.T) {
		invalidCfg := ServerConfig{
			OtelEnabled: false,
			Port:        0, // Invalid: must be > 0
		}
		cfgMap, err := toConfigMap(invalidCfg)
		assert.NoError(t, err)
		config, err := config.New(config.WithDefault(cfgMap))
		assert.NoError(t, err)

		_, err = NewServer(config)
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "port must be greater than 0")
		}
	})

	t.Run("Invalid URL", func(t *testing.T) {
		client, err := NewHTTPClient(clientDefaultCfg)
		assert.NoError(t, err)

		var result string
		err = client.Call("GET", "://invalid-url", nil, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create request")
	})

	t.Run("Max Retries", func(t *testing.T) {
		serverCfgMap, err := toConfigMap(serverCfg)
		assert.NoError(t, err)
		serverConfig, err := config.New(config.WithDefault(serverCfgMap))
		assert.NoError(t, err)
		server, err := NewServer(serverConfig)
		assert.NoError(t, err)

		server.engine.GET("/v1/Fail", func(c *gin.Context) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "unavailable"})
		})

		ts := httptest.NewServer(server.engine)
		defer ts.Close()

		retryCfg := ClientConfig{
			OtelEnabled:          false,
			HTTPClientTimeoutMs:  1000,
			HTTPClientMaxRetries: 3,
		}
		retryCfgMap, err := toConfigMap(retryCfg)
		assert.NoError(t, err)
		retryConfig, err := config.New(config.WithDefault(retryCfgMap))
		assert.NoError(t, err)

		client, err := NewHTTPClient(retryConfig)
		assert.NoError(t, err)

		var result string
		err = client.Call("GET", ts.URL+"/v1/Fail", nil, &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request failed with status 503")
	})

	t.Run("Nil Config Server", func(t *testing.T) {
		_, err := NewServer(nil)
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "config cannot be nil")
		}
	})

	t.Run("ListenAndServe", func(t *testing.T) {
		var server *Server
		err := server.ListenAndServe()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server cannot be nil")

		serverCfgMap, err := toConfigMap(ServerConfig{
			OtelEnabled: false,
			Port:        8080,
		})
		assert.NoError(t, err)
		serverConfig, err := config.New(config.WithDefault(serverCfgMap))
		assert.NoError(t, err)
		server, err = NewServer(serverConfig)
		if err != nil {
			t.Fatalf("NewServer failed: %v", err)
		}
		assert.NotNil(t, server)
		assert.NotNil(t, server.config)

		errChan := make(chan error, 1)
		go func() {
			errChan <- server.ListenAndServe()
		}()

		time.Sleep(100 * time.Millisecond)

		select {
		case err := <-errChan:
			t.Fatalf("ListenAndServe failed: %v", err)
		default:
		}
	})
}
