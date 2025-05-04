package httpc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/stretchr/testify/assert"
)

func TestHTTPServer(t *testing.T) {
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

	t.Run("OpenAPI JSON", func(t *testing.T) {
		server, err := NewServer(cfg)
		assert.NoError(t, err)

		svc := &TestService{}
		t.Logf("TestService type: %T", svc)
		t.Logf("TestService RegisterMethods output: %+v", svc.RegisterMethods())

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
		assert.True(t, ok, "Expected paths in Swagger JSON")
		assert.Contains(t, paths, "/v1/Hello", "Expected /v1/Hello in paths")
		assert.Contains(t, paths, "/v1/Create", "Expected /v1/Create in paths")
	})
}
