package httpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/T-Prohmpossadhorn/go-core/config"
	"github.com/T-Prohmpossadhorn/go-core/logger"
	"github.com/T-Prohmpossadhorn/go-core/otel"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel/trace"
)

// ServerConfig defines the configuration for the server
type ServerConfig struct {
	OtelEnabled bool `json:"otel_enabled" default:"false"`
	Port        int  `json:"port" default:"8080" required:"true" validate:"gt=0,lte=65535"`
}

// ClientConfig defines the configuration for the HTTP client
type ClientConfig struct {
	OtelEnabled          bool `json:"otel_enabled" default:"false"`
	HTTPClientTimeoutMs  int  `json:"http_client_timeout_ms" default:"1000" required:"true" validate:"gt=0"`
	HTTPClientMaxRetries int  `json:"http_client_max_retries" default:"2" validate:"gte=-1"`
}

// toConfigMap converts a struct to a map for config.WithDefault
func toConfigMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return m, nil
}

// validateConfig validates the config struct
func validateConfig(v interface{}) error {
	validate := validator.New()
	if err := validate.Struct(v); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}

// Server represents the HTTP server
type Server struct {
	engine     *gin.Engine
	config     *config.Config
	pathPrefix string
	swagger    map[string]interface{}
}

// HTTPClient represents the HTTP client
type HTTPClient struct {
	client *http.Client
	config *config.Config
}

// NewServer creates a new server with the given configuration
func NewServer(cfg *config.Config) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	logger.Info("Creating new server")

	port := 8080
	if val := cfg.Get("port"); val != nil {
		if p, ok := val.(float64); ok {
			port = int(p)
		}
	}
	if port <= 0 {
		return nil, fmt.Errorf("port must be greater than 0")
	}

	engine := gin.Default()
	server := &Server{
		engine:     engine,
		config:     cfg,
		pathPrefix: "",
		swagger:    map[string]interface{}{"paths": map[string]interface{}{}},
	}
	engine.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	engine.GET("/api/docs/swagger.json", func(c *gin.Context) {
		c.JSON(http.StatusOK, server.swagger)
	})
	logger.Info("Registering health and Swagger endpoints")
	return server, nil
}

// ListenAndServe starts the HTTP server on the configured port
func (s *Server) ListenAndServe() error {
	if s == nil {
		return fmt.Errorf("server cannot be nil")
	}
	port := 8080
	if val := s.config.Get("port"); val != nil {
		if p, ok := val.(float64); ok {
			port = int(p)
		}
	}
	addr := fmt.Sprintf(":%d", port)
	logger.Info("Starting server", logger.Field{Key: "address", Value: addr})
	return s.engine.Run(addr)
}

// NewHTTPClient creates a new HTTP client with the given configuration
func NewHTTPClient(cfg *config.Config) (*HTTPClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	logger.Info("Creating new HTTP client")

	timeoutMs := 1000
	if val := cfg.Get("http_client_timeout_ms"); val != nil {
		if ms, ok := val.(float64); ok {
			timeoutMs = int(ms)
		}
	}
	if timeoutMs <= 0 {
		return nil, fmt.Errorf("http_client_timeout_ms must be greater than 0")
	}
	maxRetries := 2
	if val := cfg.Get("http_client_max_retries"); val != nil {
		if retries, ok := val.(float64); ok {
			maxRetries = int(retries)
		}
	}

	logger.Info("Using HTTP client timeout", logger.Field{Key: "timeout_ms", Value: timeoutMs})
	logger.Info("Using HTTP max retries", logger.Field{Key: "max_retries", Value: maxRetries})

	client := &http.Client{
		Timeout: time.Duration(timeoutMs) * time.Millisecond,
	}

	return &HTTPClient{
		client: client,
		config: cfg,
	}, nil
}

// validate validates the input struct
func (s *Server) validate(input interface{}) error {
	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		logger.Error("Validation failed", logger.Field{Key: "error", Value: err.Error()})
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}

// RegisterService registers a service with the server
func (s *Server) RegisterService(service interface{}, opts ...Option) error {
	ctx := context.Background()
	otelEnabled := s.config.GetBool("otel_enabled")
	if otelEnabled {
		tracer := otel.GetTracer("httpc-server")
		var span trace.Span
		ctx, span = tracer.Start(ctx, "RegisterService")
		defer span.End()
	}

	logger.InfoContext(ctx, "Starting RegisterService")
	for _, opt := range opts {
		opt(s)
	}
	logger.InfoContext(ctx, "Service type", logger.Field{Key: "type", Value: reflect.TypeOf(service).String()})

	// Update Swagger documentation
	if err := updateSwaggerDoc(s, service, s.pathPrefix); err != nil {
		logger.ErrorContext(ctx, "Failed to update Swagger doc", logger.ErrField(err))
		return err
	}

	// Register service methods with Gin router
	svcValue := reflect.ValueOf(service)
	svcType := reflect.TypeOf(service)
	registerMethods := svcValue.MethodByName("RegisterMethods")
	if !registerMethods.IsValid() {
		err := fmt.Errorf("service must implement RegisterMethods")
		logger.ErrorContext(ctx, "Service registration failed", logger.ErrField(err))
		return err
	}

	methodsVal := registerMethods.Call(nil)
	if len(methodsVal) != 1 || methodsVal[0].Type() != reflect.TypeOf([]MethodInfo{}) {
		err := fmt.Errorf("RegisterMethods must return []MethodInfo")
		logger.ErrorContext(ctx, "Service registration failed", logger.ErrField(err))
		return err
	}

	methods := methodsVal[0].Interface().([]MethodInfo)
	for _, method := range methods {
		if !isValidHTTPMethod(method.HTTPMethod) {
			logger.WarnContext(ctx, "Skipping invalid method", logger.Field{Key: "method", Value: method.Name})
			continue
		}

		// Find the service method
		svcMethod, exists := svcType.MethodByName(method.Name)
		if !exists {
			err := fmt.Errorf("method %s not found in service", method.Name)
			logger.ErrorContext(ctx, "Service registration failed", logger.ErrField(err))
			return err
		}

		// Validate method signature
		mt := svcMethod.Type
		if mt.NumIn() != 2 || mt.NumOut() != 2 || mt.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
			err := fmt.Errorf("invalid signature for method %s", method.Name)
			logger.ErrorContext(ctx, "Service registration failed", logger.ErrField(err))
			return err
		}

		// Register endpoint with Gin
		path := fmt.Sprintf("%s/%s", strings.TrimSuffix(s.pathPrefix, "/"), method.Name)
		switch strings.ToUpper(method.HTTPMethod) {
		case "GET":
			s.engine.GET(path, func(c *gin.Context) {
				reqCtx := c.Request.Context()
				if otelEnabled {
					tracer := otel.GetTracer("httpc-server")
					var span trace.Span
					reqCtx, span = tracer.Start(reqCtx, "HandleGET:"+method.Name)
					defer span.End()
				}
				name := c.Query("name")
				results := svcValue.MethodByName(method.Name).Call([]reflect.Value{reflect.ValueOf(name)})
				if !results[1].IsNil() {
					logger.ErrorContext(reqCtx, "Method execution failed", logger.ErrField(results[1].Interface().(error)))
					c.JSON(http.StatusInternalServerError, gin.H{"error": results[1].Interface().(error).Error()})
					return
				}
				c.JSON(http.StatusOK, results[0].Interface())
			})
		case "POST":
			s.engine.POST(path, func(c *gin.Context) {
				reqCtx := c.Request.Context()
				if otelEnabled {
					tracer := otel.GetTracer("httpc-server")
					var span trace.Span
					reqCtx, span = tracer.Start(reqCtx, "HandlePOST:"+method.Name)
					defer span.End()
				}
				var input interface{}
				if method.InputType != nil {
					input = reflect.New(method.InputType).Interface()
					if err := c.ShouldBindJSON(input); err != nil {
						logger.ErrorContext(reqCtx, "Invalid request body", logger.ErrField(err))
						c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
						return
					}
					if err := validator.New().Struct(input); err != nil {
						logger.ErrorContext(reqCtx, "Validation failed", logger.ErrField(err))
						c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("validation failed: %v", err)})
						return
					}
				}
				results := svcValue.MethodByName(method.Name).Call([]reflect.Value{reflect.ValueOf(input).Elem()})
				if !results[1].IsNil() {
					logger.ErrorContext(reqCtx, "Method execution failed", logger.ErrField(results[1].Interface().(error)))
					c.JSON(http.StatusInternalServerError, gin.H{"error": results[1].Interface().(error).Error()})
					return
				}
				c.JSON(http.StatusOK, results[0].Interface())
			})
		default:
			logger.WarnContext(ctx, "Unsupported HTTP method", logger.Field{Key: "method", Value: method.HTTPMethod})
			continue
		}
		logger.InfoContext(ctx, "Registered endpoint", logger.Field{Key: "method", Value: method.HTTPMethod}, logger.Field{Key: "path", Value: path})
	}

	logger.InfoContext(ctx, "Registering endpoints with prefix", logger.Field{Key: "prefix", Value: s.pathPrefix})
	logger.InfoContext(ctx, "Service registered successfully")
	return nil
}

// Call makes an HTTP request with the given method, URL, input, and output
func (c *HTTPClient) Call(method, url string, input, output interface{}) error {
	ctx := context.Background()
	otelEnabled := c.config.GetBool("otel_enabled")
	if otelEnabled {
		tracer := otel.GetTracer("httpc-client")
		var span trace.Span
		ctx, span = tracer.Start(ctx, "HTTPClient.Call")
		defer span.End()
	}

	logger.InfoContext(ctx, "Sending request", logger.Field{Key: "method", Value: method}, logger.Field{Key: "url", Value: url})

	if !isValidHTTPMethod(method) {
		err := fmt.Errorf("invalid HTTP method: %s", method)
		logger.ErrorContext(ctx, "Invalid HTTP method", logger.ErrField(err))
		return err
	}

	var body io.Reader
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to marshal JSON input", logger.ErrField(err))
			return fmt.Errorf("failed to marshal JSON input: %w", err)
		}
		body = strings.NewReader(string(data))
		logger.InfoContext(ctx, "Request body", logger.Field{Key: "length", Value: len(data)})
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to create request", logger.ErrField(err))
		return fmt.Errorf("failed to create request: %w", err)
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		logger.ErrorContext(ctx, "Request failed", logger.ErrField(err))
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		errorMsg := "unknown error"
		if bodyBytes != nil {
			var errResp map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &errResp); err == nil {
				if msg, ok := errResp["error"].(string); ok {
					errorMsg = msg
				}
			}
		}
		logger.ErrorContext(ctx, "Request failed with status", logger.Field{Key: "status", Value: resp.StatusCode}, logger.Field{Key: "error", Value: errorMsg})
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, errorMsg)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to read response body", logger.ErrField(err))
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(bodyBytes, output); err != nil {
		logger.ErrorContext(ctx, "Failed to parse response JSON", logger.ErrField(err))
		return fmt.Errorf("failed to parse response JSON: %w", err)
	}

	logger.InfoContext(ctx, "Request completed successfully")
	return nil
}

// isValidHTTPMethod checks if the HTTP method is valid
func isValidHTTPMethod(method string) bool {
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, m := range validMethods {
		if strings.EqualFold(method, m) {
			return true
		}
	}
	logger.Warn("Skipping invalid HTTP method", logger.Field{Key: "method", Value: method})
	return false
}
