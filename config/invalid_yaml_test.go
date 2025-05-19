package config

import (
	"os"
	"testing"
)

// TestInvalidYAML ensures invalid YAML files produce an error.
func TestInvalidYAML(t *testing.T) {
	tmp, err := os.CreateTemp("", "bad*.yaml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write([]byte("invalid: [")); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	tmp.Close()

	cfg, err := New(WithFilepath(tmp.Name()))
	if err == nil || cfg != nil {
		t.Fatalf("expected error for invalid yaml, got cfg=%v err=%v", cfg, err)
	}
}
