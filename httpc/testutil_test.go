package httpc

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
)

// setupServer creates a test server with the given configuration, service, and prefix
func setupServer(t *testing.T, cfg ServerConfig, svc interface{}, prefix string) *httptest.Server {
	cfgMap, err := toConfigMap(cfg)
	if err != nil {
		t.Fatalf("Failed to create config map: %v", err)
	}

	c, err := config.New(config.WithDefault(cfgMap))
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	srv, err := NewServer(c)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Inline prefix logic to avoid undefined WithPrefix
	err = srv.RegisterService(svc, func(cfg *serviceConfig) {
		cfg.prefix = prefix
	})
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	// Custom handler to ensure response bodies are sent
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Buffer response to capture body
		recorder := httptest.NewRecorder()
		srv.engine.ServeHTTP(recorder, r)

		// Log recorder details
		logger.Info("Recorder headers", logger.Any("headers", recorder.Header()))
		logger.Info("Recorder body", logger.String("body", recorder.Body.String()))

		// Copy headers
		for k, v := range recorder.Header() {
			w.Header()[k] = v
		}

		// Write status code
		w.WriteHeader(recorder.Code)

		// Write body with explicit Content-Length
		body := recorder.Body.Bytes()
		if len(body) > 0 {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
			_, err := w.Write(body)
			if err != nil {
				logger.Error("Failed to write response body", logger.ErrField(err))
			}
		}

		// Flush to ensure body is sent
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}))

	return ts
}

// toConfigMap converts a ServerConfig to a map for configuration
func toConfigMap(cfg ServerConfig) (map[string]interface{}, error) {
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", cfg.Port)
	}
	return map[string]interface{}{
		"otel_enabled": cfg.OtelEnabled,
		"port":         cfg.Port,
	}, nil
}
