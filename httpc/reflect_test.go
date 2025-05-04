package httpc

import (
	"os"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/stretchr/testify/assert"
)

func TestGetServiceInfo(t *testing.T) {
	os.Setenv("CONFIG_LOGGER_LEVEL", "info")
	if err := logger.Init(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	t.Run("Valid Service", func(t *testing.T) {
		svc := &TestService{}
		info, err := getServiceInfo(svc)
		assert.NoError(t, err)
		assert.Len(t, info, 2)
		assert.Equal(t, "Hello", info[0].Name)
		assert.Equal(t, "GET", info[0].HTTPMethod)
		assert.Equal(t, "Create", info[1].Name)
		assert.Equal(t, "POST", info[1].HTTPMethod)
	})

	t.Run("Invalid Signature", func(t *testing.T) {
		svc := &InvalidSigService{}
		info, err := getServiceInfo(svc)
		assert.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "invalid signature for method BadMethod")
	})

	t.Run("Nil Service", func(t *testing.T) {
		info, err := getServiceInfo(nil)
		assert.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "service cannot be nil")
	})

	t.Run("No RegisterMethods", func(t *testing.T) {
		type NoRegisterService struct{}
		svc := &NoRegisterService{}
		info, err := getServiceInfo(svc)
		assert.Error(t, err)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "no RegisterMethods method found")
	})
}
