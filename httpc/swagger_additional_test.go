package httpc

import "testing"

// TestUpdateSwaggerDocNilServer verifies error when server is nil.
func TestUpdateSwaggerDocNilServer(t *testing.T) {
	err := updateSwaggerDoc(nil, &TestService{}, "/v1")
	if err == nil {
		t.Fatal("expected error for nil server")
	}
}

// TestUpdateSwaggerDocInvalidMethod ensures methods with invalid HTTP verbs are skipped.
func TestUpdateSwaggerDocInvalidMethod(t *testing.T) {
	srv := &Server{swagger: map[string]interface{}{}}
	svc := &InvalidMethodService{}
	if err := updateSwaggerDoc(srv, svc, "/v1"); err != nil {
		t.Fatalf("updateSwaggerDoc returned error: %v", err)
	}
	paths := srv.swagger["paths"].(map[string]interface{})
	if len(paths) != 0 {
		t.Fatalf("expected no paths registered, got %v", paths)
	}
}

// TestUpdateSwaggerDocPrefixFormatting checks that paths have leading slash and swagger defaults initialized.
func TestUpdateSwaggerDocPrefixFormatting(t *testing.T) {
	srv := &Server{}
	svc := &TestService{}
	if err := updateSwaggerDoc(srv, svc, "v1"); err != nil {
		t.Fatalf("updateSwaggerDoc returned error: %v", err)
	}
	if srv.swagger["openapi"] == nil {
		t.Fatalf("swagger openapi not set")
	}
	paths := srv.swagger["paths"].(map[string]interface{})
	if _, ok := paths["/v1/Hello"]; !ok {
		t.Fatalf("expected path with leading slash; got %v", paths)
	}
}
