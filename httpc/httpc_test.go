package httpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/stretchr/testify/require"
)

// ProductService for testing
type Product struct {
	ID    int     `json:"id" validate:"required,gte=1"`
	Name  string  `json:"name" validate:"required,min=1,max=100"`
	Price float64 `json:"price" validate:"gte=0"`
}

type ProductService struct{}

func (s *ProductService) Create(product Product) (string, error) {
	return fmt.Sprintf("Created product %s", product.Name), nil
}

func (s *ProductService) RegisterMethods() []MethodInfo {
	return []MethodInfo{
		{
			Name:       "Create",
			HTTPMethod: "POST",
			InputType:  reflect.TypeOf(Product{}),
			OutputType: reflect.TypeOf(""),
			Func:       reflect.ValueOf(s.Create),
		},
	}
}

// CustomerService for testing
type Address struct {
	Street string `json:"street" validate:"required,min=1,max=200"`
	City   string `json:"city" validate:"required,min=1,max=100"`
}

type Customer struct {
	Email   string  `json:"email" validate:"required,email"`
	Age     int     `json:"age" validate:"gte=18,lte=120"`
	Address Address `json:"address" validate:"required"`
}

type CustomerService struct{}

func (s *CustomerService) Create(customer Customer) (string, error) {
	return fmt.Sprintf("Created customer %s", customer.Email), nil
}

func (s *CustomerService) RegisterMethods() []MethodInfo {
	return []MethodInfo{
		{
			Name:       "Create",
			HTTPMethod: "POST",
			InputType:  reflect.TypeOf(Customer{}),
			OutputType: reflect.TypeOf(""),
			Func:       reflect.ValueOf(s.Create),
		},
	}
}

func TestHTTPC(t *testing.T) {
	os.Setenv("CONFIG_LOGGER_LEVEL", "info")
	if err := logger.Init(); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	serverCfg := ServerConfig{
		OtelEnabled: false,
		Port:        8080,
	}
	clientCfgMap := map[string]interface{}{
		"otel_enabled":            false,
		"http_client_timeout_ms":  1000,
		"http_client_max_retries": 2,
	}
	clientDefaultCfg, err := config.New(config.WithDefault(clientCfgMap))
	require.NoError(t, err)

	t.Run("Valid Swagger Doc Generation", func(t *testing.T) {
		svc := &TestService{}
		ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/api/docs/swagger.json")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		t.Logf("Swagger JSON response: %s", string(body))

		var doc map[string]interface{}
		err = json.Unmarshal(body, &doc)
		require.NoError(t, err)
		require.NotNil(t, doc)
		require.Equal(t, "3.0.3", doc["openapi"])
		paths, ok := doc["paths"].(map[string]interface{})
		require.True(t, ok)
		require.Contains(t, paths, "/v1/Hello")
	})

	t.Run("Custom Path Prefix", func(t *testing.T) {
		svc := &CustomPathService{}
		ts := setupServer(t, serverCfg, svc, "/custom/api")
		defer ts.Close()

		client, err := NewHTTPClient(clientDefaultCfg)
		require.NoError(t, err)
		input := CustomInput{Data: "test"}
		var output CustomOutput
		err = client.Call("POST", ts.URL+"/custom/api/Process", input, &output)
		require.NoError(t, err)
		require.Equal(t, "Processed: test", output.Result)

		resp, err := http.Get(ts.URL + "/api/docs/swagger.json")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		t.Logf("Swagger JSON response: %s", string(body))

		var doc map[string]interface{}
		err = json.Unmarshal(body, &doc)
		require.NoError(t, err)
		require.NotNil(t, doc)
		require.Equal(t, "3.0.3", doc["openapi"])
		paths, ok := doc["paths"].(map[string]interface{})
		require.True(t, ok)
		require.Contains(t, paths, "/custom/api/Process")
		require.NotContains(t, paths, "/v1/Process")
	})

	t.Run("Error 500 Handling", func(t *testing.T) {
		svc := &MultiMethodService{}
		ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		client, err := NewHTTPClient(clientDefaultCfg)
		require.NoError(t, err)

		// Test GET method with error
		var output MultiOutput
		err = client.Call("GET", ts.URL+"/v1/GetMethod?name=error", nil, &output)
		require.Error(t, err)
		require.Contains(t, err.Error(), "request failed with status 500")
		require.Contains(t, err.Error(), "simulated server error")

		// Test GET method with success
		err = client.Call("GET", ts.URL+"/v1/GetMethod?name=success", nil, &output)
		require.NoError(t, err)
		require.Equal(t, "GET: success", output.Result)

		// Test POST method with error
		input := MultiInput{Value: "error"}
		err = client.Call("POST", ts.URL+"/v1/PostMethod", input, &output)
		require.Error(t, err)
		require.Contains(t, err.Error(), "request failed with status 500")
		require.Contains(t, err.Error(), "simulated server error")

		// Test POST method with success
		input = MultiInput{Value: "success"}
		err = client.Call("POST", ts.URL+"/v1/PostMethod", input, &output)
		require.NoError(t, err)
		require.Equal(t, "POST: success", output.Result)

		// Test PUT method with error
		input = MultiInput{Value: "error"}
		err = client.Call("PUT", ts.URL+"/v1/PutMethod", input, &output)
		require.Error(t, err)
		require.Contains(t, err.Error(), "request failed with status 500")
		require.Contains(t, err.Error(), "simulated server error")

		// Test PUT method with success
		input = MultiInput{Value: "success"}
		err = client.Call("PUT", ts.URL+"/v1/PutMethod", input, &output)
		require.NoError(t, err)
		require.Equal(t, "PUT: success", output.Result)

		// Test DELETE method with error
		input = MultiInput{Value: "error"}
		err = client.Call("DELETE", ts.URL+"/v1/DeleteMethod", input, &output)
		require.Error(t, err)
		require.Contains(t, err.Error(), "request failed with status 500")
		require.Contains(t, err.Error(), "simulated server error")

		// Test DELETE method with success
		input = MultiInput{Value: "success"}
		err = client.Call("DELETE", ts.URL+"/v1/DeleteMethod", input, &output)
		require.NoError(t, err)
		require.Equal(t, "DELETE: success", output.Result)
	})

	t.Run("Dynamic Swagger Schema Generation", func(t *testing.T) {
		t.Run("Product Service", func(t *testing.T) {
			svc := &ProductService{}
			ts := setupServer(t, serverCfg, svc, "/v1")
			defer ts.Close()

			resp, err := http.Get(ts.URL + "/api/docs/swagger.json")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var doc map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&doc)
			require.NoError(t, err)

			paths, ok := doc["paths"].(map[string]interface{})
			require.True(t, ok)
			require.Contains(t, paths, "/v1/Create")

			createPath, ok := paths["/v1/Create"].(map[string]interface{})
			require.True(t, ok)
			postMethod, ok := createPath["post"].(map[string]interface{})
			require.True(t, ok)
			requestBody, ok := postMethod["requestBody"].(map[string]interface{})
			require.True(t, ok)
			content, ok := requestBody["content"].(map[string]interface{})
			require.True(t, ok)
			jsonContent, ok := content["application/json"].(map[string]interface{})
			require.True(t, ok)
			schema, ok := jsonContent["schema"].(map[string]interface{})
			require.True(t, ok)
			properties, ok := schema["properties"].(map[string]interface{})
			require.True(t, ok)

			idProp, ok := properties["id"].(map[string]interface{})
			require.True(t, ok)
			require.Equal(t, "integer", idProp["type"])
			require.Equal(t, float64(1), idProp["minimum"])

			nameProp, ok := properties["name"].(map[string]interface{})
			require.True(t, ok)
			require.Equal(t, "string", nameProp["type"])
			require.Equal(t, float64(1), nameProp["minLength"])
			require.Equal(t, float64(100), nameProp["maxLength"])

			priceProp, ok := properties["price"].(map[string]interface{})
			require.True(t, ok)
			require.Equal(t, "number", priceProp["type"])
			require.Equal(t, float64(0), priceProp["minimum"])

			required, ok := schema["required"].([]interface{})
			require.True(t, ok)
			require.Contains(t, required, "id")
			require.Contains(t, required, "name")
		})

		t.Run("Customer Service", func(t *testing.T) {
			svc := &CustomerService{}
			ts := setupServer(t, serverCfg, svc, "/v2")
			defer ts.Close()

			resp, err := http.Get(ts.URL + "/api/docs/swagger.json")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var doc map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&doc)
			require.NoError(t, err)

			paths, ok := doc["paths"].(map[string]interface{})
			require.True(t, ok)
			require.Contains(t, paths, "/v2/Create")

			createPath, ok := paths["/v2/Create"].(map[string]interface{})
			require.True(t, ok)
			postMethod, ok := createPath["post"].(map[string]interface{})
			require.True(t, ok)
			requestBody, ok := postMethod["requestBody"].(map[string]interface{})
			require.True(t, ok)
			content, ok := requestBody["content"].(map[string]interface{})
			require.True(t, ok)
			jsonContent, ok := content["application/json"].(map[string]interface{})
			require.True(t, ok)
			schema, ok := jsonContent["schema"].(map[string]interface{})
			require.True(t, ok)
			properties, ok := schema["properties"].(map[string]interface{})
			require.True(t, ok)

			emailProp, ok := properties["email"].(map[string]interface{})
			require.True(t, ok)
			require.Equal(t, "string", emailProp["type"])
			require.Equal(t, "email", emailProp["format"])

			ageProp, ok := properties["age"].(map[string]interface{})
			require.True(t, ok)
			require.Equal(t, "integer", ageProp["type"])
			require.Equal(t, float64(18), ageProp["minimum"])
			require.Equal(t, float64(120), ageProp["maximum"])

			addressProp, ok := properties["address"].(map[string]interface{})
			require.True(t, ok)
			require.Equal(t, "object", addressProp["type"])
			addressProps, ok := addressProp["properties"].(map[string]interface{})
			require.True(t, ok)

			streetProp, ok := addressProps["street"].(map[string]interface{})
			require.True(t, ok)
			require.Equal(t, "string", streetProp["type"])
			require.Equal(t, float64(1), streetProp["minLength"])
			require.Equal(t, float64(200), streetProp["maxLength"])

			cityProp, ok := addressProps["city"].(map[string]interface{})
			require.True(t, ok)
			require.Equal(t, "string", cityProp["type"])
			require.Equal(t, float64(1), cityProp["minLength"])
			require.Equal(t, float64(100), cityProp["maxLength"])

			addressRequired, ok := addressProp["required"].([]interface{})
			require.True(t, ok)
			require.Contains(t, addressRequired, "street")
			require.Contains(t, addressRequired, "city")

			required, ok := schema["required"].([]interface{})
			require.True(t, ok)
			require.Contains(t, required, "email")
			require.Contains(t, required, "address")
		})
	})

	t.Run("Retry Backoff", func(t *testing.T) {
		svc := &TestService{}
		ts := setupServer(t, serverCfg, svc, "/v1")
		defer ts.Close()

		client, err := NewHTTPClient(clientDefaultCfg)
		require.NoError(t, err)

		var result string
		err = client.Call("GET", ts.URL+"/v1/Hello?name=Test", nil, &result)
		require.NoError(t, err)
		require.Equal(t, "Hello, Test!", result)
	})
}
