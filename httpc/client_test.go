package httpc

import (
	"os"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient(t *testing.T) {
	os.Setenv("CONFIG_LOGGER_LEVEL", "info")
	if err := logger.Init(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	serverCfg := ServerConfig{
		OtelEnabled: false,
		Port:        8080,
	}

	t.Run("Valid Client Request", func(t *testing.T) {
		svc := &TestService{}
		ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		cfgMap := map[string]interface{}{
			"otel_enabled":            false,
			"http_client_timeout_ms":  1000,
			"http_client_max_retries": 2,
		}
		config, err := config.New(config.WithDefault(cfgMap))
		require.NoError(t, err)

		client, err := NewHTTPClient(config)
		require.NoError(t, err)

		var result string
		err = client.Call("GET", ts.URL+"/v1/Hello?name=Test", nil, &result)
		require.NoError(t, err)
		require.Equal(t, "Hello, Test!", result)
	})

	t.Run("Client POST Request", func(t *testing.T) {
		svc := &TestService{}
		ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		cfgMap := map[string]interface{}{
			"otel_enabled":            false,
			"http_client_timeout_ms":  1000,
			"http_client_max_retries": 2,
		}
		config, err := config.New(config.WithDefault(cfgMap))
		require.NoError(t, err)

		client, err := NewHTTPClient(config)
		require.NoError(t, err)

		user := User{
			Name:  "TestUser",
			Email: "test@example.com",
		}
		validator := validator.New()
		err = validator.Struct(user)
		require.NoError(t, err)

		var result string
		err = client.Call("POST", ts.URL+"/v1/Create", user, &result)
		require.NoError(t, err)
		require.Equal(t, "Created user TestUser", result)
	})

	t.Run("Client Invalid Input", func(t *testing.T) {
		svc := &TestService{}
		ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		cfgMap := map[string]interface{}{
			"otel_enabled":            false,
			"http_client_timeout_ms":  1000,
			"http_client_max_retries": 2,
		}
		config, err := config.New(config.WithDefault(cfgMap))
		require.NoError(t, err)

		client, err := NewHTTPClient(config)
		require.NoError(t, err)

		user := User{
			Name:  "",        // Invalid: required
			Email: "invalid", // Invalid: not an email
		}
		var result string
		err = client.Call("POST", ts.URL+"/v1/Create", user, &result)
		require.Error(t, err)
		require.Contains(t, err.Error(), "request failed with status 400")
	})

	t.Run("Client Server Error", func(t *testing.T) {
		svc := &MultiMethodService{}
		ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		cfgMap := map[string]interface{}{
			"otel_enabled":            false,
			"http_client_timeout_ms":  1000,
			"http_client_max_retries": 2,
		}
		config, err := config.New(config.WithDefault(cfgMap))
		require.NoError(t, err)

		client, err := NewHTTPClient(config)
		require.NoError(t, err)

		var output MultiOutput
		err = client.Call("GET", ts.URL+"/v1/GetMethod?name=error", nil, &output)
		require.Error(t, err)
		require.Contains(t, err.Error(), "request failed with status 500")
		require.Contains(t, err.Error(), "simulated server error")
	})

	t.Run("Client Multi-Method Errors", func(t *testing.T) {
		svc := &MultiMethodService{}
		ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		cfgMap := map[string]interface{}{
			"otel_enabled":            false,
			"http_client_timeout_ms":  1000,
			"http_client_max_retries": 2,
		}
		config, err := config.New(config.WithDefault(cfgMap))
		require.NoError(t, err)

		client, err := NewHTTPClient(config)
		require.NoError(t, err)

		input := MultiInput{Value: "error"}
		var output MultiOutput

		// Test POST
		err = client.Call("POST", ts.URL+"/v1/PostMethod", input, &output)
		require.Error(t, err)
		require.Contains(t, err.Error(), "request failed with status 500")
		require.Contains(t, err.Error(), "simulated server error")

		// Test PUT
		err = client.Call("PUT", ts.URL+"/v1/PutMethod", input, &output)
		require.Error(t, err)
		require.Contains(t, err.Error(), "request failed with status 500")
		require.Contains(t, err.Error(), "simulated server error")

		// Test DELETE
		err = client.Call("DELETE", ts.URL+"/v1/DeleteMethod", input, &output)
		require.Error(t, err)
		require.Contains(t, err.Error(), "request failed with status 500")
		require.Contains(t, err.Error(), "simulated server error")
	})

	t.Run("Client Invalid Method", func(t *testing.T) {
		cfgMap := map[string]interface{}{
			"otel_enabled":            false,
			"http_client_timeout_ms":  1000,
			"http_client_max_retries": 2,
		}
		config, err := config.New(config.WithDefault(cfgMap))
		require.NoError(t, err)

		client, err := NewHTTPClient(config)
		require.NoError(t, err)

		var result string
		err = client.Call("INVALID", "http://example.com", nil, &result)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid HTTP method: INVALID")
	})
}
