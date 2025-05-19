package otel

import (
	"github.com/T-Prohmpossadhorn/go-core/config"
	"testing"
)

// TestInitDisabled ensures Init respects disabled config.
func TestInitDisabled(t *testing.T) {
	cfg, err := config.New(config.WithDefault(map[string]interface{}{"otel_enabled": false}))
	if err != nil {
		t.Fatalf("new config: %v", err)
	}
	if err := Init(cfg); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if tracerProvider != nil {
		t.Fatal("expected tracerProvider nil when disabled")
	}
}

// TestValidateEndpoint covers valid and invalid endpoints.
func TestValidateEndpoint(t *testing.T) {
	if err := validateEndpoint(""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateEndpoint("localhost:4317"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := validateEndpoint("invalid.invalid:4317"); err == nil {
		t.Fatal("expected invalid hostname error")
	}
	if err := validateEndpoint("localhost:99999"); err == nil {
		t.Fatal("expected invalid port error")
	}
}
