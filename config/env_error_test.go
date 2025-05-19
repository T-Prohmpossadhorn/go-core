package config

import (
	"os"
	"testing"
)

// TestWithEnvInvalidFormat ensures WithEnv fails when env vars are not parsable.
func TestWithEnvInvalidFormat(t *testing.T) {
	os.Setenv("CONFIG_DEBUG", "notbool")
	os.Setenv("CONFIG_ENVIRONMENT", "testing")
	defer os.Unsetenv("CONFIG_DEBUG")
	defer os.Unsetenv("CONFIG_ENVIRONMENT")

	cfg, err := New(WithEnv("CONFIG"))
	if err == nil || cfg != nil {
		t.Fatalf("expected error from invalid env vars, got cfg=%v err=%v", cfg, err)
	}
}
