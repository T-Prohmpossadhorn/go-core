package httpc

import "testing"

func TestParseInt(t *testing.T) {
	n, err := parseInt("42")
	if err != nil {
		t.Fatalf("parseInt returned error: %v", err)
	}
	if n != 42 {
		t.Fatalf("expected 42, got %d", n)
	}
	if _, err := parseInt("a"); err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestParseFloat(t *testing.T) {
	f, err := parseFloat("3.14")
	if err != nil {
		t.Fatalf("parseFloat returned error: %v", err)
	}
	if f != 3.14 {
		t.Fatalf("expected 3.14, got %f", f)
	}
	if _, err := parseFloat("a"); err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestWithPathPrefix(t *testing.T) {
	cfg := serviceConfig{}
	opt := WithPathPrefix("/api")
	opt(&cfg)
	if cfg.prefix != "/api" {
		t.Fatalf("expected /api, got %s", cfg.prefix)
	}
}

func TestMultiMethodServiceMethods(t *testing.T) {
	s := MultiMethodService{}
	if out, err := s.GetMethod("name"); err != nil || out.Result != "GET: name" {
		t.Fatalf("unexpected result: %v %v", out, err)
	}
	if out, err := s.PostMethod(MultiInput{Value: "x"}); err != nil || out.Result != "POST: x" {
		t.Fatalf("unexpected result: %v %v", out, err)
	}
	if out, err := s.PutMethod(MultiInput{Value: "x"}); err != nil || out.Result != "PUT: x" {
		t.Fatalf("unexpected result: %v %v", out, err)
	}
	if out, err := s.DeleteMethod(MultiInput{Value: "x"}); err != nil || out.Result != "DELETE: x" {
		t.Fatalf("unexpected result: %v %v", out, err)
	}
}
