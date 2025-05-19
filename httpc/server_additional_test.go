package httpc

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/config"
)

// headService provides a HEAD method for testing.
type headService struct{}

func (s headService) HeadMethod(name string) (string, error) { return name, nil }
func (s headService) RegisterMethods() []MethodInfo {
	return []MethodInfo{{Name: "HeadMethod", HTTPMethod: http.MethodHead, InputType: reflect.TypeOf(""), OutputType: reflect.TypeOf(""), Func: reflect.ValueOf(s).MethodByName("HeadMethod")}}
}

// TestHandleMethodInvalidJSON checks JSON binding failure path.
func TestHandleMethodInvalidJSON(t *testing.T) {
	cfgMap, _ := toConfigMap(ServerConfig{OtelEnabled: false, Port: 8080})
	c, _ := config.New(config.WithDefault(cfgMap))
	srv, _ := NewServer(c)
	svc := &TestService{}
	if err := srv.RegisterService(svc, WithPathPrefix("/v1")); err != nil {
		t.Fatalf("register service failed: %v", err)
	}
	ts := httptest.NewServer(srv.engine)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/Create", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// TestHandleMethodHead verifies HEAD responses.
func TestHandleMethodHead(t *testing.T) {
	cfgMap, _ := toConfigMap(ServerConfig{OtelEnabled: false, Port: 8080})
	c, _ := config.New(config.WithDefault(cfgMap))
	srv, _ := NewServer(c)
	hs := &headService{}
	if err := srv.RegisterService(hs, WithPathPrefix("/v1")); err != nil {
		t.Fatalf("register service failed: %v", err)
	}
	ts := httptest.NewServer(srv.engine)
	defer ts.Close()
	resp, err := http.Head(ts.URL + "/v1/HeadMethod?name=head")
	if err != nil {
		t.Fatalf("head request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
