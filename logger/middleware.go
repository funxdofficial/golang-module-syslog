package logger

import (
	"context"
	"net/http"
	"time"
)

// HTTPRequestInfo contains information extracted from HTTP request
// Interface ini memungkinkan logger bekerja dengan berbagai web framework
type HTTPRequestInfo interface {
	Method() string
	Path() string
	Body() string
	Header(key string) string
	Context() context.Context
}

// StandardHTTPRequest implements HTTPRequestInfo for standard net/http
type StandardHTTPRequest struct {
	req *http.Request
}

func (r *StandardHTTPRequest) Method() string {
	if r.req == nil {
		return ""
	}
	return r.req.Method
}

func (r *StandardHTTPRequest) Path() string {
	if r.req == nil {
		return ""
	}
	return r.req.URL.Path
}

func (r *StandardHTTPRequest) Body() string {
	// Body reading should be handled by framework middleware
	return ""
}

func (r *StandardHTTPRequest) Header(key string) string {
	if r.req == nil {
		return ""
	}
	return r.req.Header.Get(key)
}

func (r *StandardHTTPRequest) Context() context.Context {
	if r.req == nil {
		return context.Background()
	}
	return r.req.Context()
}

// StartFromHTTPRequestInfo creates context from HTTPRequestInfo and logs START event
// Ini membuat logger bisa bekerja dengan berbagai web framework
func (l *Logger) StartFromHTTPRequestInfo(reqInfo HTTPRequestInfo, config StartConfig) context.Context {
	ctx := context.Background()
	if reqInfo != nil {
		ctx = reqInfo.Context()
	}

	// Extract method dan endpoint dari request info (otomatis)
	method := reqInfo.Method()
	endpoint := reqInfo.Path()

	// Set method dan endpoint jika ada
	if method != "" {
		ctx = WithMethod(ctx, method)
	}
	if endpoint != "" {
		ctx = WithEndpoint(ctx, endpoint)
	}

	// Override dengan config jika ada
	if config.Method != "" {
		ctx = WithMethod(ctx, config.Method)
	}
	if config.Endpoint != "" {
		ctx = WithEndpoint(ctx, config.Endpoint)
	}

	// Set service name if provided
	if config.ServiceName != "" {
		ctx = WithServiceName(ctx, config.ServiceName)
	}

	// Generate or use existing UUID
	if config.TransactionID == "" {
		ctx = WithNewUUID(ctx)
	} else {
		ctx = WithUUID(ctx, config.TransactionID)
		ctx = WithTransactionID(ctx, config.TransactionID)
	}

	// Set trace ID if provided
	if config.TraceID != "" {
		ctx = WithTraceID(ctx, config.TraceID)
	}

	// Set start time for execution time tracking
	ctx = WithStartTime(ctx, time.Now())

	// Set default level if not provided
	level := config.Level
	if level == "" {
		level = "INFO"
	}

	// Set default message if not provided
	message := config.Message
	if message == "" {
		message = "Request started"
	}

	// Use body from request info or config
	body := config.Body
	if body == "" && reqInfo != nil {
		body = reqInfo.Body()
	}

	// Log START event
	l.LogStart(ctx, level, message, body)

	return ctx
}

// MiddlewareConfig untuk konfigurasi middleware
type MiddlewareConfig struct {
	ServiceName string   // Nama service
	SkipPaths   []string // Path yang di-skip dari logging
}

// StandardHTTPMiddleware untuk net/http standard library
func (l *Logger) StandardHTTPMiddleware(config MiddlewareConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip paths jika ada
			for _, skipPath := range config.SkipPaths {
				if r.URL.Path == skipPath {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Create request info
			reqInfo := &StandardHTTPRequest{req: r}

			// Start logging
			startConfig := StartConfig{
				ServiceName: config.ServiceName,
				// Method dan Endpoint otomatis dari request
			}
			ctx := l.StartFromHTTPRequestInfo(reqInfo, startConfig)

			// Wrap response writer untuk capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			// Stop logging
			level := "SUCCESS"
			if wrapped.statusCode >= 400 {
				level = "ERROR"
			} else if wrapped.statusCode >= 300 {
				level = "WARNING"
			}

			body := ""
			if wrapped.body != nil {
				body = string(wrapped.body)
			}

			l.Stop(ctx, level, "Request completed", body)
		})
	}
}

// responseWriter wraps http.ResponseWriter untuk capture status code dan body
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body = b
	return rw.ResponseWriter.Write(b)
}
