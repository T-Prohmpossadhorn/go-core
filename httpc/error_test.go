package httpc

import (
	"os"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/stretchr/testify/assert"
)

func TestErrorCases(t *testing.T) {
	os.Setenv("CONFIG_LOGGER_LEVEL", "info")
	if err := logger.Init(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	t.Run("Nil Config", func(t *testing.T) {
		server, err := NewServer(nil)
		assert.Error(t, err)
		assert.Nil(t, server)
		assert.Contains(t, err.Error(), "config cannot be nil")

		client, err := NewHTTPClient(nil)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "config cannot be nil")
	})

	t.Run("Invalid Service Signature", func(t *testing.T) {
		serverCfg := ServerConfig{
			OtelEnabled: false,
			Port:        8080,
		}
		serverCfgMap, err := toConfigMap(serverCfg)
		assert.NoError(t, err)
		cfg, err := config.New(config.WithDefault(serverCfgMap))
		assert.NoError(t, err)

		server, err := NewServer(cfg)
		assert.NoError(t, err)

		svc := &InvalidSigService{}
		err = server.RegisterService(svc, WithPathPrefix("/v1"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid signature")
	})
}
