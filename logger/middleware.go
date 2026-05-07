package logger

import (
	"context"
	"net/http"
	"sync"
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
	if l == nil {
		return context.Background()
	}
	ctx := context.Background()
	if reqInfo != nil {
		ctx = reqInfo.Context()
	}

	// Extract method dan endpoint dari request info (otomatis)
	var method, endpoint string
	if reqInfo != nil {
		method = reqInfo.Method()
		endpoint = reqInfo.Path()
	}

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
	ctx = WithStartTime(ctx, TimestampWIB())

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

	// StartLevel adalah level log untuk event START (default: "INFO")
	// Set ke "DEBUG" atau "TRACE" bila tidak ingin event request masuk ke log produksi.
	StartLevel string

	// StopLevelFn adalah mapper kustom dari HTTP status code ke level log untuk event STOP.
	// Bila nil, dipakai default levelFromStatusCode (lihat dokumentasi fungsinya).
	StopLevelFn func(status int) string
}

// levelFromStatusCode mengembalikan log level yang sesuai untuk HTTP status code:
//
//	1xx -> "DEBUG"   (informational; jarang dipakai aplikasi)
//	2xx -> "SUCCESS" (request berhasil)
//	3xx -> "INFO"    (redirect adalah perilaku HTTP normal, bukan warning)
//	4xx -> "WARNING" (client error: bad request, unauthorized, not found, dll)
//	5xx -> "ERROR"   (server error: internal error, bad gateway, dll)
//
// Status di luar range tersebut (mis. 0 atau negatif) dianggap "INFO".
// Untuk override, set MiddlewareConfig.StopLevelFn.
func levelFromStatusCode(status int) string {
	switch {
	case status >= 500:
		return "ERROR"
	case status >= 400:
		return "WARNING"
	case status >= 300:
		return "INFO"
	case status >= 200:
		return "SUCCESS"
	case status >= 100:
		return "DEBUG"
	default:
		return "INFO"
	}
}

// passthroughMiddleware mengembalikan middleware no-op yang hanya forward ke next handler.
// Dipakai sebagai fallback bila logger nil agar pemanggil tetap aman.
func passthroughMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if next != nil {
				next.ServeHTTP(w, r)
			}
		})
	}
}

// shouldSkipPath mengecek apakah path request termasuk dalam SkipPaths.
func shouldSkipPath(r *http.Request, skipPaths []string) bool {
	if r == nil || r.URL == nil {
		return false
	}
	for _, p := range skipPaths {
		if r.URL.Path == p {
			return true
		}
	}
	return false
}

// resolveStartLevel mengembalikan level untuk event START dengan default "INFO".
func resolveStartLevel(level string) string {
	if level == "" {
		return "INFO"
	}
	return level
}

// resolveStopLevelFn mengembalikan mapper status->level dengan default levelFromStatusCode.
func resolveStopLevelFn(fn func(int) string) func(int) string {
	if fn == nil {
		return levelFromStatusCode
	}
	return fn
}

// handleHTTPRequest menjalankan satu siklus log START -> next handler -> log STOP
// untuk net/http standard library. Mengisolasi logika supaya middleware tetap ringkas.
func (l *Logger) handleHTTPRequest(
	w http.ResponseWriter,
	r *http.Request,
	next http.Handler,
	serviceName, startLevel string,
	stopLevelFn func(int) string,
) {
	reqInfo := &StandardHTTPRequest{req: r}
	startConfig := StartConfig{
		ServiceName: serviceName,
		Level:       startLevel,
	}
	ctx := l.StartFromHTTPRequestInfo(reqInfo, startConfig)

	wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	next.ServeHTTP(wrapped, r.WithContext(ctx))

	wrapped.mu.Lock()
	status := wrapped.statusCode
	body := ""
	if len(wrapped.body) > 0 {
		body = string(wrapped.body)
	}
	wrapped.mu.Unlock()

	l.Stop(ctx, stopLevelFn(status), "Request completed", body)
}

// StandardHTTPMiddleware untuk net/http standard library
func (l *Logger) StandardHTTPMiddleware(config MiddlewareConfig) func(http.Handler) http.Handler {
	if l == nil {
		return passthroughMiddleware()
	}

	startLevel := resolveStartLevel(config.StartLevel)
	stopLevelFn := resolveStopLevelFn(config.StopLevelFn)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r == nil || w == nil || shouldSkipPath(r, config.SkipPaths) {
				if next != nil {
					next.ServeHTTP(w, r)
				}
				return
			}
			l.handleHTTPRequest(w, r, next, config.ServiceName, startLevel, stopLevelFn)
		})
	}
}

// responseWriter wraps http.ResponseWriter untuk capture status code dan body
type responseWriter struct {
	http.ResponseWriter
	mu         sync.Mutex
	statusCode int
	body       []byte
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.mu.Lock()
	rw.statusCode = code
	rw.mu.Unlock()
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.mu.Lock()
	rw.body = append([]byte(nil), b...)
	rw.mu.Unlock()
	return rw.ResponseWriter.Write(b)
}
