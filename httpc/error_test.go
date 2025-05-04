package httpc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/stretchr/testify/require"
)

func TestErrorCases(t *testing.T) {
	os.Setenv("CONFIG_LOGGER_LEVEL", "info")
	if err := logger.Init(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	serverCfg := ServerConfig{
		OtelEnabled: false,
		Port:        8080,
	}

	t.Run("Invalid HTTP Method", func(t *testing.T) {
		svc := &InvalidMethodService{}
		ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/docs/swagger.json")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&doc)
		require.NoError(t, err)
		require.NotNil(t, doc)
		paths, ok := doc["paths"].(map[string]interface{})
		if ok {
			require.NotContains(t, paths, "/v1/InvalidMethod", "Invalid HTTP method should not be in Swagger paths")
		}
	})

	t.Run("Invalid Signature", func(t *testing.T) {
		svc := &InvalidSigService{}
		cfgMap, err := toConfigMap(serverCfg)
		require.NoError(t, err)
		config, err := config.New(config.WithDefault(cfgMap))
		require.NoError(t, err)

		server, err := NewServer(config)
		require.NoError(t, err)
		err = server.RegisterService(svc, WithPathPrefix("/v1"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid signature for method BadMethod")

		ts := httptest.NewServer(server.engine)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/docs/swagger.json")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&doc)
		require.NoError(t, err)
		require.NotNil(t, doc)
		paths, ok := doc["paths"].(map[string]interface{})
		if ok {
			require.NotContains(t, paths, "/v1/BadMethod", "Invalid signature method should not be in Swagger paths")
		}
	})
}
