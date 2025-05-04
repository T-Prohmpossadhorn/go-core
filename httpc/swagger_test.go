package httpc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/stretchr/testify/assert"
)

// InvalidMethodService mocks a service with an invalid HTTP method
type InvalidMethodService struct{}

func (s *InvalidMethodService) RegisterMethods() []MethodInfo {
	return []MethodInfo{
		{
			Name:        "InvalidMethod",
			HTTPMethod:  "INVALID",
			Description: "Invalid method",
			InputType:   reflect.TypeOf(""),
			OutputType:  reflect.TypeOf(""),
		},
	}
}

func TestSwagger(t *testing.T) {
	os.Setenv("CONFIG_LOGGER_LEVEL", "info")
	if err := logger.Init(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	serverCfg := ServerConfig{
		OtelEnabled: false,
		Port:        8080,
	}
	serverCfgMap, err := toConfigMap(serverCfg)
	assert.NoError(t, err)
	cfg, err := config.New(config.WithDefault(serverCfgMap))
	assert.NoError(t, err)

	t.Run("Valid Swagger Doc Generation", func(t *testing.T) {
		server, err := NewServer(cfg)
		assert.NoError(t, err)

		svc := &TestService{}
		err = server.RegisterService(svc, WithPathPrefix("/v1"))
		assert.NoError(t, err)

		ts := httptest.NewServer(server.engine)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/docs/swagger.json")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var doc map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&doc)
		assert.NoError(t, err)

		paths, ok := doc["paths"].(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, paths, "/v1/Hello")
		assert.Contains(t, paths, "/v1/Create")
	})

	t.Run("Invalid Service Signature", func(t *testing.T) {
		server, err := NewServer(cfg)
		assert.NoError(t, err)

		svc := &InvalidSigService{}
		err = server.RegisterService(svc, WithPathPrefix("/v1"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid signature")
	})

	t.Run("Nil Server", func(t *testing.T) {
		err := updateSwaggerDoc(nil, &TestService{}, "/v1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "server cannot be nil")
	})

	t.Run("No RegisterMethods", func(t *testing.T) {
		server, err := NewServer(cfg)
		assert.NoError(t, err)

		type InvalidService struct{}
		svc := &InvalidService{}
		err = updateSwaggerDoc(server, svc, "/v1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service must implement RegisterMethods")
	})

	t.Run("Invalid HTTP Method", func(t *testing.T) {
		server, err := NewServer(cfg)
		assert.NoError(t, err)

		svc := &InvalidMethodService{}
		err = server.RegisterService(svc, WithPathPrefix("/v1"))
		assert.NoError(t, err)

		ts := httptest.NewServer(server.engine)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/docs/swagger.json")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var doc map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&doc)
		assert.NoError(t, err)

		paths, ok := doc["paths"].(map[string]interface{})
		assert.True(t, ok)
		assert.NotContains(t, paths, "/v1/InvalidMethod", "Invalid HTTP method should not be in Swagger paths")
	})

	t.Run("Empty Path Prefix", func(t *testing.T) {
		server, err := NewServer(cfg)
		assert.NoError(t, err)

		svc := &TestService{}
		err = server.RegisterService(svc, WithPathPrefix(""))
		assert.NoError(t, err)

		ts := httptest.NewServer(server.engine)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/docs/swagger.json")
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var doc map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&doc)
		assert.NoError(t, err)

		paths, ok := doc["paths"].(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, paths, "/Hello")
		assert.Contains(t, paths, "/Create")
	})
}
