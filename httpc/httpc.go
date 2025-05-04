package httpc

import (
	"bytes"
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
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type ServerConfig struct {
	OtelEnabled bool `json:"otel_enabled" default:"false"`
	Port        int  `json:"port" default:"8080" required:"true" validate:"gt=0,lte=65535"`
}

type ClientConfig struct {
	OtelEnabled    bool  `json:"otel_enabled" default:"false"`
	TimeoutMs      int   `json:"http_client_timeout_ms" default:"3000" required:"true" validate:"gte=100,lte=30000"`
	MaxRetries     int   `json:"http_client_max_retries" default:"3" required:"true" validate:"gte=0,lte=5"`
	BackoffBaseMs  int64 `json:"http_client_backoff_base_ms" default:"100" validate:"gte=50,lte=1000"`
	BackoffMaxMs   int64 `json:"http_client_backoff_max_ms" default:"1000" validate:"gte=100,lte=5000"`
	BackoffFactor  int   `json:"http_client_backoff_factor" default:"2" validate:"gte=1,lte=5"`
	DisableBackoff bool  `json:"http_client_disable_backoff" default:"false"`
}

type Server struct {
	engine      *gin.Engine
	swagger     map[string]interface{}
	otelEnabled bool
	config      *config.Config
	server      *http.Server
}

type HTTPClient struct {
	client      *http.Client
	config      ClientConfig
	otelEnabled bool
}

func NewServer(c *config.Config) (*Server, error) {
	logger.Info("Creating new server")
	gin.SetMode(gin.DebugMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	swaggerDoc := map[string]interface{}{
		"openapi": "3.0.3",
		"info": map[string]interface{}{
			"title":   "httpc API",
			"version": "1.0.0",
		},
		"paths": map[string]interface{}{},
	}
	server := &Server{
		engine:      engine,
		swagger:     swaggerDoc,
		otelEnabled: c.GetBool("otel_enabled"),
		config:      c,
	}

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	engine.GET("/api/docs/swagger.json", func(c *gin.Context) {
		c.JSON(http.StatusOK, server.swagger)
	})

	logger.Info("Registering health and Swagger endpoints")
	return server, nil
}

func (s *Server) ListenAndServe() error {
	port := s.config.Get("port").(int)
	addr := fmt.Sprintf(":%d", port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}

	logger.Info("Starting server", logger.String("address", addr))
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed to start: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	logger.Info("Shutting down server")
	return s.server.Shutdown(ctx)
}

func (s *Server) RegisterService(svc interface{}, opts ...ServiceOption) error {
	logger.Info("Starting RegisterService")
	cfg := &serviceConfig{prefix: "/"}
	for _, opt := range opts {
		opt(cfg)
	}

	logger.Info("Service type", logger.String("type", fmt.Sprintf("%T", svc)))
	// Use reflection for all services, as getServiceInfo handles RegisterMethods
	var methods []MethodInfo
	info, err := getServiceInfo(svc)
	if err != nil {
		return fmt.Errorf("failed to get service info: %w", err)
	}
	methods = info
	logger.Info("Retrieved methods")
	return s.registerMethods(methods, cfg, svc)
}

func (s *Server) registerMethods(methods []MethodInfo, cfg *serviceConfig, svc interface{}) error {
	for _, m := range methods {
		path := fmt.Sprintf("%s/%s", cfg.prefix, m.Name)
		switch strings.ToUpper(m.HTTPMethod) {
		case http.MethodGet:
			s.engine.GET(path, s.handleMethod(m))
		case http.MethodPost, http.MethodPut, http.MethodDelete:
			s.engine.Handle(m.HTTPMethod, path, s.handleMethod(m))
		default:
			logger.Warn("Skipping invalid HTTP method", logger.String("method", m.HTTPMethod))
			continue
		}
		logger.Info("Registered endpoint", logger.String("method", m.HTTPMethod), logger.String("path", path))
	}

	if len(methods) > 0 {
		if err := updateSwaggerDoc(s, svc, cfg.prefix); err != nil {
			logger.Error("Failed to update Swagger doc", logger.ErrField(err))
		}
	}

	logger.Info("Registering endpoints with prefix", logger.String("prefix", cfg.prefix))
	logger.Info("Service registered successfully")
	return nil
}

func (s *Server) handleMethod(m MethodInfo) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Placeholder: no-op for tracing
		ctx := c.Request.Context()
		var span interface{} // Placeholder
		defer func() {
			if span != nil {
				// No-op
			}
		}()

		reqCtx := ctx
		var inputVal interface{}
		inputType := m.InputType
		if inputType.Kind() == reflect.String {
			// For string inputs, use query parameter directly
			if m.HTTPMethod == http.MethodGet {
				query := c.Query("name")
				inputVal = query
			} else {
				inputVal = reflect.New(inputType).Interface()
				if err := c.ShouldBindJSON(inputVal); err != nil {
					logger.ErrorContext(reqCtx, "JSON binding failed", logger.ErrField(err))
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			}
		} else {
			// For struct inputs, bind and validate
			inputVal = reflect.New(inputType).Interface()
			if m.HTTPMethod == http.MethodGet {
				if err := c.ShouldBindQuery(inputVal); err != nil {
					logger.ErrorContext(reqCtx, "Query binding failed", logger.ErrField(err))
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			} else {
				if err := c.ShouldBindJSON(inputVal); err != nil {
					logger.ErrorContext(reqCtx, "JSON binding failed", logger.ErrField(err))
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
			}
			validate := validator.New()
			if err := validate.Struct(inputVal); err != nil {
				logger.ErrorContext(reqCtx, "Validation failed", logger.ErrField(err))
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("validation failed: %s", err.Error())})
				return
			}
		}

		// Prepare input for method call
		var callInput reflect.Value
		if inputType.Kind() == reflect.String {
			callInput = reflect.ValueOf(inputVal)
		} else {
			callInput = reflect.ValueOf(inputVal).Elem()
		}

		// Call the method without context
		results := m.Func.Call([]reflect.Value{callInput})
		if !results[1].IsNil() {
			err := results[1].Interface().(error)
			logger.ErrorContext(reqCtx, "Method execution failed", logger.ErrField(err))
			logger.InfoContext(reqCtx, "Sending error response", logger.String("body", fmt.Sprintf(`{"error":"%s"}`, err.Error())))
			c.Data(http.StatusInternalServerError, "application/json", []byte(`{"error":"`+err.Error()+`"}`))
			logger.InfoContext(reqCtx, "After Data write", logger.Int("status", c.Writer.Status()), logger.Any("headers", c.Writer.Header()))
			return
		}

		c.JSON(http.StatusOK, results[0].Interface())
	}
}

func getIntConfig(c *config.Config, key string, defaultValue int) int {
	if val := c.Get(key); val != nil {
		if intVal, ok := val.(int); ok {
			return intVal
		}
	}
	return defaultValue
}

func getBoolConfig(c *config.Config, key string, defaultValue bool) bool {
	if val := c.Get(key); val != nil {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultValue
}

func NewHTTPClient(c *config.Config) (*HTTPClient, error) {
	logger.Info("Creating new HTTP client")
	cfg := ClientConfig{
		OtelEnabled:    getBoolConfig(c, "otel_enabled", false),
		TimeoutMs:      getIntConfig(c, "http_client_timeout_ms", 3000),
		MaxRetries:     getIntConfig(c, "http_client_max_retries", 3),
		BackoffBaseMs:  int64(getIntConfig(c, "http_client_backoff_base_ms", 100)),
		BackoffMaxMs:   int64(getIntConfig(c, "http_client_backoff_max_ms", 1000)),
		BackoffFactor:  getIntConfig(c, "http_client_backoff_factor", 2),
		DisableBackoff: getBoolConfig(c, "http_client_disable_backoff", false),
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("invalid client config: %w", err)
	}

	logger.Info("Using HTTP client timeout", logger.Int("timeout_ms", cfg.TimeoutMs))
	logger.Info("Using HTTP max retries", logger.Int("max_retries", cfg.MaxRetries))

	client := &http.Client{
		Timeout: time.Duration(cfg.TimeoutMs) * time.Millisecond,
	}
	return &HTTPClient{
		client:      client,
		config:      cfg,
		otelEnabled: cfg.OtelEnabled,
	}, nil
}

func (h *HTTPClient) Call(method, url string, input, output interface{}) error {
	// Placeholder: no-op for tracing
	ctx := context.Background()
	var span interface{} // Placeholder
	defer func() {
		if span != nil {
			// No-op
		}
	}()

	reqCtx := ctx
	method = strings.ToUpper(method)
	if !isValidHTTPMethod(method) {
		err := fmt.Errorf("invalid HTTP method: %s", method)
		logger.ErrorContext(reqCtx, "Invalid HTTP method", logger.ErrField(err))
		return err
	}

	var bodyData []byte
	var err error
	if input != nil {
		bodyData, err = json.Marshal(input)
		if err != nil {
			return fmt.Errorf("failed to marshal input: %w", err)
		}
	}

	for attempt := 1; attempt <= h.config.MaxRetries+1; attempt++ {
		var body io.Reader
		if bodyData != nil {
			body = bytes.NewReader(bodyData) // Fresh reader for each attempt
			logger.InfoContext(reqCtx, "Request body", logger.Int("length", len(bodyData)), logger.Int("attempt", attempt))
		}

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		if bodyData != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("X-Request-ID", uuid.New().String())

		logger.InfoContext(reqCtx, "Sending request", logger.String("method", method), logger.String("url", url), logger.Int("attempt", attempt))

		resp, err := h.client.Do(req)
		if err != nil {
			logger.ErrorContext(reqCtx, "Request attempt failed", logger.Int("attempt", attempt), logger.ErrField(err))
			if attempt == h.config.MaxRetries+1 {
				return fmt.Errorf("request failed: %w", err)
			}
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if output != nil {
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					logger.ErrorContext(reqCtx, "Failed to read response body", logger.ErrField(err))
					return fmt.Errorf("failed to read response body: %w", err)
				}
				if err := json.Unmarshal(bodyBytes, output); err != nil {
					return fmt.Errorf("failed to unmarshal response: %w", err)
				}
			}
			logger.InfoContext(reqCtx, "Request completed successfully")
			return nil
		}

		if resp.StatusCode < 500 || attempt == h.config.MaxRetries+1 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			logger.InfoContext(reqCtx, "Error response body", logger.String("body", string(bodyBytes)))
			logger.InfoContext(reqCtx, "Response headers", logger.Any("headers", resp.Header))
			var errResp map[string]string
			if len(bodyBytes) > 0 {
				if err := json.Unmarshal(bodyBytes, &errResp); err == nil && errResp["error"] != "" {
					logger.ErrorContext(reqCtx, "Request failed with status", logger.Int("status", resp.StatusCode), logger.String("error", errResp["error"]))
					return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, errResp["error"])
				}
			}
			logger.ErrorContext(reqCtx, "Request failed with status", logger.Int("status", resp.StatusCode), logger.String("error", "unknown error"))
			return fmt.Errorf("request failed with status %d: unknown error", resp.StatusCode)
		}

		logger.ErrorContext(reqCtx, "Request attempt failed with status", logger.Int("attempt", attempt), logger.Int("status", resp.StatusCode))

		if h.config.DisableBackoff {
			continue
		}

		backoff := h.config.BackoffBaseMs * int64(1<<uint(attempt-1))
		if backoff > h.config.BackoffMaxMs {
			backoff = h.config.BackoffMaxMs
		}
		time.Sleep(time.Duration(backoff) * time.Millisecond)
	}

	return fmt.Errorf("all retry attempts failed")
}
