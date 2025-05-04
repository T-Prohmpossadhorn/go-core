package httpc

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/stretchr/testify/assert"
)

// ReflectTestService for testing reflection
type ReflectTestService struct{}

func (s *ReflectTestService) TestMethod(ctx context.Context, input string) (string, error) {
	return "Test response", nil
}

func (s *ReflectTestService) RegisterMethods() []MethodInfo {
	return []MethodInfo{
		{
			Name:       "TestMethod",
			HTTPMethod: "GET",
			InputType:  reflect.TypeOf(""),
			OutputType: reflect.TypeOf(""),
		},
	}
}

// InvalidReflectService for testing invalid reflection
type InvalidReflectService struct{}

func (s *InvalidReflectService) RegisterMethods() []MethodInfo {
	return nil
}

// BadSignatureService for testing invalid RegisterMethods signature
type BadSignatureService struct{}

func (s *BadSignatureService) RegisterMethods() string {
	return "invalid"
}

// EmptyMethodInfoService for testing empty MethodInfo
type EmptyMethodInfoService struct{}

func (s *EmptyMethodInfoService) RegisterMethods() []MethodInfo {
	return []MethodInfo{
		{
			Name:        "",
			HTTPMethod:  "",
			Description: "",
			InputType:   nil,
			OutputType:  nil,
		},
	}
}

func TestReflect(t *testing.T) {
	os.Setenv("CONFIG_LOGGER_LEVEL", "info")
	if err := logger.Init(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	_, err := config.New(config.WithDefault(map[string]interface{}{
		"otel_enabled": false,
	}))
	assert.NoError(t, err)

	t.Run("Valid Service", func(t *testing.T) {
		svc := &ReflectTestService{}
		methods, err := getServiceInfo(svc)
		assert.NoError(t, err)
		assert.Len(t, methods, 1)
		assert.Equal(t, "TestMethod", methods[0].Name)
	})

	t.Run("Nil Service", func(t *testing.T) {
		methods, err := getServiceInfo(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service cannot be nil")
		assert.Nil(t, methods)
	})

	t.Run("Invalid Service", func(t *testing.T) {
		svc := &InvalidReflectService{}
		methods, err := getServiceInfo(svc)
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "no methods defined for service")
		}
		assert.Nil(t, methods)
	})

	t.Run("Invalid RegisterMethods Signature", func(t *testing.T) {
		svc := &BadSignatureService{}
		methods, err := getServiceInfo(svc)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "RegisterMethods must return []MethodInfo")
		assert.Nil(t, methods)
	})

	t.Run("Empty MethodInfo", func(t *testing.T) {
		svc := &EmptyMethodInfoService{}
		methods, err := getServiceInfo(svc)
		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), "invalid MethodInfo: Name or HTTPMethod is empty")
		}
		assert.Nil(t, methods)
	})

	t.Run("Non-Pointer Service", func(t *testing.T) {
		svc := ReflectTestService{}
		methods, err := getServiceInfo(svc)
		assert.NoError(t, err)
		assert.Len(t, methods, 1)
		if len(methods) > 0 {
			assert.Equal(t, "TestMethod", methods[0].Name)
		}
	})

	t.Run("Valid MethodInfo Fields", func(t *testing.T) {
		svc := &ReflectTestService{}
		methods, err := getServiceInfo(svc)
		assert.NoError(t, err)
		assert.Len(t, methods, 1)
		if len(methods) > 0 {
			assert.Equal(t, "TestMethod", methods[0].Name)
			assert.Equal(t, "GET", methods[0].HTTPMethod)
			assert.NotNil(t, methods[0].InputType)
			assert.NotNil(t, methods[0].OutputType)
		}
	})
}
