package httpc

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/stretchr/testify/require"
)

func TestSwagger(t *testing.T) {
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

	t.Run("Valid Swagger Doc", func(t *testing.T) {
		svc := &TestService{}
		ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/docs/swagger.json")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var doc map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&doc)
		require.NoError(t, err)
		require.NotNil(t, doc)
		require.Equal(t, "3.0.3", doc["openapi"])

		paths, ok := doc["paths"].(map[string]interface{})
		require.True(t, ok)
		require.Contains(t, paths, "/v1/Hello")
		require.Contains(t, paths, "/v1/Create")

		helloPath, ok := paths["/v1/Hello"].(map[string]interface{})
		require.True(t, ok)
		getMethod, ok := helloPath["get"].(map[string]interface{})
		require.True(t, ok)
		parameters, ok := getMethod["parameters"].([]interface{})
		require.True(t, ok)
		require.Len(t, parameters, 1)
		nameParam, ok := parameters[0].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, "name", nameParam["name"])
		require.Equal(t, "query", nameParam["in"])

		createPath, ok := paths["/v1/Create"].(map[string]interface{})
		require.True(t, ok)
		postMethod, ok := createPath["post"].(map[string]interface{})
		require.True(t, ok)
		requestBody, ok := postMethod["requestBody"].(map[string]interface{})
		require.True(t, ok)
		content, ok := requestBody["content"].(map[string]interface{})
		require.True(t, ok)
		jsonContent, ok := content["application/json"].(map[string]interface{})
		require.True(t, ok)
		schema, ok := jsonContent["schema"].(map[string]interface{})
		require.True(t, ok)
		properties, ok := schema["properties"].(map[string]interface{})
		require.True(t, ok)
		require.Contains(t, properties, "name")
		require.Contains(t, properties, "email")
	})
}
