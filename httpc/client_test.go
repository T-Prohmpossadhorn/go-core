package httpc

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/T-Prohmpossadhorn/go-core/otel"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	os.Setenv("CONFIG_LOGGER_LEVEL", "debug")
	if err := logger.Init(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	serverCfg := ServerConfig{
		OtelEnabled: false,
		Port:        8080,
	}
	serverCfgMap, err := toConfigMap(serverCfg)
	assert.NoError(t, err)
	serverConfig, err := config.New(config.WithDefault(serverCfgMap))
	assert.NoError(t, err)

	clientCfg := ClientConfig{
		OtelEnabled:          false,
		HTTPClientTimeoutMs:  1000,
		HTTPClientMaxRetries: 2,
	}
	clientCfgMap, err := toConfigMap(clientCfg)
	assert.NoError(t, err)
	clientConfig, err := config.New(config.WithDefault(clientCfgMap))
	assert.NoError(t, err)

	t.Run("Hello Method", func(t *testing.T) {
		server, err := NewServer(serverConfig)
		assert.NoError(t, err)

		svc := &TestService{}
		err = server.RegisterService(svc, WithPathPrefix("/v1"))
		assert.NoError(t, err)

		ts := httptest.NewServer(server.engine)
		defer ts.Close()

		client, err := NewHTTPClient(clientConfig)
		assert.NoError(t, err)

		var result string
		err = client.Call("GET", ts.URL+"/v1/Hello?name=Test", nil, &result)
		assert.NoError(t, err)
		assert.Equal(t, "Hello, Test!", result)
	})

	t.Run("Create Method", func(t *testing.T) {
		server, err := NewServer(serverConfig)
		assert.NoError(t, err)

		svc := &TestService{}
		err = server.RegisterService(svc, WithPathPrefix("/v1"))
		assert.NoError(t, err)

		ts := httptest.NewServer(server.engine)
		defer ts.Close()

		client, err := NewHTTPClient(clientConfig)
		assert.NoError(t, err)

		user := User{
			Name: "TestUser",
			Address: struct {
				City string `json:"city" validate:"required"`
			}{
				City: "TestCity",
			},
		}
		err = server.validate(user)
		assert.NoError(t, err)

		var result string
		err = client.Call("POST", ts.URL+"/v1/Create", user, &result)
		assert.NoError(t, err)
		assert.Equal(t, "Created user TestUser", result)
	})

	t.Run("Invalid User Validation", func(t *testing.T) {
		server, err := NewServer(serverConfig)
		assert.NoError(t, err)

		user := User{
			Name: "", // Invalid: Name is required
			Address: struct {
				City string `json:"city" validate:"required"`
			}{
				City: "TestCity",
			},
		}
		err = server.validate(user)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("Tracing Enabled", func(t *testing.T) {
		os.Setenv("OTEL_TEST_MOCK_EXPORTER", "true")
		defer os.Unsetenv("OTEL_TEST_MOCK_EXPORTER")

		serverCfg.OtelEnabled = true
		serverCfgMap, err := toConfigMap(serverCfg)
		assert.NoError(t, err)
		serverConfig, err := config.New(config.WithDefault(serverCfgMap))
		assert.NoError(t, err)

		server, err := NewServer(serverConfig)
		assert.NoError(t, err)

		svc := &TestService{}
		err = server.RegisterService(svc, WithPathPrefix("/v1"))
		assert.NoError(t, err)

		ts := httptest.NewServer(server.engine)
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

		server, err := NewServer(serverConfig)
		assert.NoError(t, err)

		svc := &TestService{}
		err = server.RegisterService(svc, WithPathPrefix("/v1"))
		assert.NoError(t, err)

		ts := httptest.NewServer(server.engine)
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
	})
}
