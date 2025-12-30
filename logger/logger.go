package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

// LogLevel represents the severity level of a log entry
type LogLevel int

const (
	LevelError LogLevel = iota
	LevelWarning
	LevelSuccess
	LevelInfo
)

// ContextKey is a type for context keys
type ContextKey string

const (
	// UUIDKey is the key for storing UUID in context
	UUIDKey ContextKey = "logger_uuid"
	// ServiceNameKey is the key for storing service name in context
	ServiceNameKey ContextKey = "logger_service_name"
	// EndpointKey is the key for storing endpoint in context
	EndpointKey ContextKey = "logger_endpoint"
	// MethodKey is the key for storing HTTP method in context
	MethodKey ContextKey = "logger_method"
	// TraceIDKey is the key for storing trace ID in context
	TraceIDKey ContextKey = "logger_trace_id"
	// TransactionIDKey is the key for storing transaction ID in context
	TransactionIDKey ContextKey = "logger_transaction_id"
	// StartTimeKey is the key for storing start time in context
	StartTimeKey ContextKey = "logger_start_time"
)

// LogFlag represents Start or Stop flag
type LogFlag string

const (
	FlagStart LogFlag = "START"
	FlagStop  LogFlag = "STOP"
)

// LogEntry represents a log entry with all mandatory fields
type LogEntry struct {
	Timestamp     string
	LogLevel      string
	TransactionID string
	ServiceName   string
	Endpoint      string
	MethodType    string
	ExecutionTime string
	ServerIP      string
	TraceID       string
	Body          string
	Flag          LogFlag
	Message       string
}

// StartConfig represents configuration for starting a log entry
type StartConfig struct {
	ServiceName   string
	Endpoint      string
	Method        string
	TransactionID string
	TraceID       string
	Body          string
	Message       string
	Level         string
}

// LogType represents the type of logging output
type LogType string

const (
	LogTypeConsole LogType = "console" // Hanya console (dengan warna)
	LogTypeFile    LogType = "file"    // Hanya file (tanpa warna, plain text)
	LogTypeAll     LogType = "all"     // Console + File (console dengan warna, file tanpa warna)
)

// LoggerConfig represents configuration for creating a logger instance
type LoggerConfig struct {
	LogFile string  // Path to log file (required jika Type = "file" atau "all")
	Type    LogType // Type of logging: "console", "file", atau "all"
}

// Logger is the main logging structure
type Logger struct {
	errorLog      *log.Logger
	warningLog    *log.Logger
	successLog    *log.Logger
	infoLog       *log.Logger
	file          *os.File
	useFile       bool
	enableConsole bool
	hostname      string
	ipAddress     string
}

// getLocalIP returns the local IP address
func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "unknown"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// getHostname returns the hostname of the system
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// generateUUID generates a new UUID v7 (time-based) using google/uuid library
func generateUUID() string {
	u, err := uuid.NewV7()
	if err != nil {
		return "unknown"
	}
	return u.String()
}

// getUUIDFromContext extracts UUID from context or generates a new one
func getUUIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return generateUUID()
	}
	if uuid, ok := ctx.Value(UUIDKey).(string); ok && uuid != "" {
		return uuid
	}
	return generateUUID()
}

// WithUUID adds UUID to context
func WithUUID(ctx context.Context, uuid string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, UUIDKey, uuid)
}

// WithNewUUID generates a new UUID and adds it to context
func WithNewUUID(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return WithUUID(ctx, generateUUID())
}

// WithServiceName adds service name to context
func WithServiceName(ctx context.Context, serviceName string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ServiceNameKey, serviceName)
}

// WithEndpoint adds endpoint to context
func WithEndpoint(ctx context.Context, endpoint string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, EndpointKey, endpoint)
}

// WithMethod adds HTTP method to context
func WithMethod(ctx context.Context, method string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, MethodKey, method)
}

// WithTraceID adds trace ID to context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithTransactionID adds transaction ID to context
func WithTransactionID(ctx context.Context, transactionID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, TransactionIDKey, transactionID)
}

// WithStartTime adds start time to context for execution time tracking
func WithStartTime(ctx context.Context, startTime time.Time) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, StartTimeKey, startTime)
}

// WithHTTPRequest extracts method and endpoint from HTTP request and adds to context
// Ini membuat logger bisa otomatis melihat method dan routing dari HTTP request
func WithHTTPRequest(ctx context.Context, r *http.Request) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if r == nil {
		return ctx
	}

	// Extract method
	method := r.Method
	if method != "" {
		ctx = WithMethod(ctx, method)
	}

	// Extract endpoint/route
	endpoint := r.URL.Path
	if endpoint != "" {
		ctx = WithEndpoint(ctx, endpoint)
	}

	return ctx
}

// StartFromRequest creates context from HTTP request and logs START event
// Ini adalah helper untuk otomatis extract method dan routing dari HTTP request
func (l *Logger) StartFromRequest(r *http.Request, config StartConfig) context.Context {
	ctx := context.Background()

	// Extract method dan endpoint dari HTTP request (otomatis)
	ctx = WithHTTPRequest(ctx, r)

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

	// Read request body if needed (optional)
	body := config.Body
	if body == "" && r != nil && r.Body != nil {
		// Try to read body (but don't consume it)
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil && len(bodyBytes) > 0 {
			body = string(bodyBytes)
			// Restore body for further use
			r.Body = io.NopCloser(strings.NewReader(body))
		}
	}

	// Log START event
	l.LogStart(ctx, level, message, body)

	return ctx
}

// getValueFromContext extracts a string value from context
func getValueFromContext(ctx context.Context, key ContextKey, defaultValue string) string {
	if ctx == nil {
		return defaultValue
	}
	if value, ok := ctx.Value(key).(string); ok && value != "" {
		return value
	}
	return defaultValue
}

// getStartTimeFromContext extracts start time from context
func getStartTimeFromContext(ctx context.Context) (time.Time, bool) {
	if ctx == nil {
		return time.Time{}, false
	}
	if startTime, ok := ctx.Value(StartTimeKey).(time.Time); ok {
		return startTime, true
	}
	return time.Time{}, false
}

// NewLogger creates a new logger instance with config
// If config is nil, uses default config (console only)
func StartLogger(config *LoggerConfig) (*Logger, error) {
	// Default config
	if config == nil {
		config = &LoggerConfig{
			Type: LogTypeConsole,
		}
	}

	// Set default type if empty
	if config.Type == "" {
		if config.LogFile != "" {
			config.Type = LogTypeAll // Auto-set to "all" if logFile is provided
		} else {
			config.Type = LogTypeConsole // Default to console
		}
	}

	// Validate config
	if (config.Type == LogTypeFile || config.Type == LogTypeAll) && config.LogFile == "" {
		return nil, fmt.Errorf("LogFile is required when Type is 'file' or 'all'")
	}

	// Determine enable flags based on type
	enableConsole := config.Type == LogTypeConsole || config.Type == LogTypeAll
	enableFile := config.Type == LogTypeFile || config.Type == LogTypeAll

	logger := &Logger{
		errorLog:      log.New(os.Stderr, "", 0),
		warningLog:    log.New(os.Stdout, "", 0),
		successLog:    log.New(os.Stdout, "", 0),
		infoLog:       log.New(os.Stdout, "", 0),
		useFile:       false,
		enableConsole: enableConsole,
		hostname:      getHostname(),
		ipAddress:     getLocalIP(),
	}

	// Setup file logging if enabled
	// File akan ditulis tanpa warna (plain text)
	if enableFile && config.LogFile != "" {
		file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logger.file = file
		logger.useFile = true
		// File akan ditulis tanpa ANSI color codes (plain text)
	}

	return logger, nil
}

// NewLoggerSimple creates a logger with just a file path (backward compatibility)
func NewLoggerSimple(logFile string) (*Logger, error) {
	if logFile == "" {
		config := &LoggerConfig{
			Type: LogTypeConsole,
		}
		return StartLogger(config)
	}
	config := &LoggerConfig{
		LogFile: logFile,
		Type:    LogTypeAll, // Console + File
	}
	return StartLogger(config)
}

// Close closes the log file if one was opened
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// getCallerInfo returns the file, line number, and function name of the caller
func getCallerInfo(skip int) (file string, line int, function string) {
	pc, filePath, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", 0, "unknown"
	}
	// Get only the filename, not the full path
	file = filepath.Base(filePath)

	// Get function name
	fn := runtime.FuncForPC(pc)
	if fn != nil {
		function = fn.Name()
		// Remove package path, keep only function name
		// Example: github.com/user/package.functionName -> functionName
		parts := strings.Split(function, ".")
		if len(parts) > 0 {
			function = parts[len(parts)-1]
		}
	} else {
		function = "unknown"
	}

	return file, line, function
}

// formatMessage formats the log message with timestamp, level, location, IP, hostname, and UUID
func (l *Logger) formatMessage(level string, uuid string, message string, args ...interface{}) string {
	// Get current time with more detail
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// Get caller information (skip 3 levels: formatMessage -> writeToBoth -> Error/Warning/etc)
	file, line, function := getCallerInfo(4)

	formattedMsg := fmt.Sprintf(message, args...)

	// Format: [timestamp] [level] [uuid] [hostname@ip] [file:line:function] message
	return fmt.Sprintf("[%s] [%s] [%s] [%s@%s] [%s:%d:%s] %s",
		timestamp, level, uuid, l.hostname, l.ipAddress, file, line, function, formattedMsg)
}

// formatMandatoryMessage formats the log message with all mandatory fields in a readable format
func (l *Logger) formatMandatoryMessage(entry LogEntry) string {
	var parts []string

	// Timestamp and Level
	parts = append(parts, fmt.Sprintf("[%s]", entry.Timestamp))
	parts = append(parts, fmt.Sprintf("[%s]", entry.LogLevel))

	// Flag (START/STOP) if present
	if entry.Flag != "" {
		parts = append(parts, fmt.Sprintf("[%s]", entry.Flag))
	}

	// Service information
	if entry.ServiceName != "unknown" {
		parts = append(parts, fmt.Sprintf("Service: %s", entry.ServiceName))
	}

	// Method and Routing/Endpoint - make it clear and prominent
	if entry.MethodType != "unknown" && entry.Endpoint != "unknown" {
		// Format: METHOD /route/path (more readable)
		parts = append(parts, fmt.Sprintf("[%s] %s", entry.MethodType, entry.Endpoint))
	} else if entry.MethodType != "unknown" {
		parts = append(parts, fmt.Sprintf("Method: %s", entry.MethodType))
	} else if entry.Endpoint != "unknown" {
		parts = append(parts, fmt.Sprintf("Route: %s", entry.Endpoint))
	}

	// IDs
	if entry.TransactionID != "" {
		parts = append(parts, fmt.Sprintf("TxnID: %s", entry.TransactionID))
	}

	if entry.TraceID != "" && entry.TraceID != entry.TransactionID {
		parts = append(parts, fmt.Sprintf("TraceID: %s", entry.TraceID))
	}

	// Execution time
	if entry.ExecutionTime != "0ms" && entry.ExecutionTime != "" {
		parts = append(parts, fmt.Sprintf("Duration: %s", entry.ExecutionTime))
	}

	// Server IP
	parts = append(parts, fmt.Sprintf("IP: %s", entry.ServerIP))

	// Body (if present)
	if entry.Body != "" {
		parts = append(parts, fmt.Sprintf("Body: %s", entry.Body))
	}

	// Message
	parts = append(parts, fmt.Sprintf("→ %s", entry.Message))

	return strings.Join(parts, " | ")
}

// writeToBoth writes to both console and file if enabled
func (l *Logger) writeToBoth(level string, uuid string, message string, args ...interface{}) {
	formatted := l.formatMessage(level, uuid, message, args...)

	// Write to console if enabled (DENGAN WARNA)
	if l.enableConsole {
		switch level {
		case "ERROR":
			fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", formatted) // Red
		case "WARNING":
			fmt.Fprintf(os.Stdout, "\033[33m%s\033[0m\n", formatted) // Yellow
		case "SUCCESS":
			fmt.Fprintf(os.Stdout, "\033[32m%s\033[0m\n", formatted) // Green
		case "INFO":
			fmt.Fprintf(os.Stdout, "\033[36m%s\033[0m\n", formatted) // Cyan
		default:
			fmt.Println(formatted)
		}
	}

	// Write to file if enabled (TANPA WARNA - plain text)
	// Config EnableFile menentukan apakah log ditulis ke file .log atau tidak
	if l.useFile && l.file != nil {
		fmt.Fprintln(l.file, formatted) // Plain text, no color codes
	}
}

// Error logs an error message
func (l *Logger) Error(message string, args ...interface{}) {
	l.writeToBoth("ERROR", generateUUID(), message, args...)
}

// Warning logs a warning message
func (l *Logger) Warning(message string, args ...interface{}) {
	l.writeToBoth("WARNING", generateUUID(), message, args...)
}

// Success logs a success message
func (l *Logger) Success(message string, args ...interface{}) {
	l.writeToBoth("SUCCESS", generateUUID(), message, args...)
}

// Info logs an info message
func (l *Logger) Info(message string, args ...interface{}) {
	l.writeToBoth("INFO", generateUUID(), message, args...)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(format, args...)
}

// Warningf logs a formatted warning message
func (l *Logger) Warningf(format string, args ...interface{}) {
	l.Warning(format, args...)
}

// Successf logs a formatted success message
func (l *Logger) Successf(format string, args ...interface{}) {
	l.Success(format, args...)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(format, args...)
}

// ErrorCtx logs an error message with context
func (l *Logger) ErrorCtx(ctx context.Context, message string, args ...interface{}) {
	uuid := getUUIDFromContext(ctx)
	l.writeToBoth("ERROR", uuid, message, args...)
}

// WarningCtx logs a warning message with context
func (l *Logger) WarningCtx(ctx context.Context, message string, args ...interface{}) {
	uuid := getUUIDFromContext(ctx)
	l.writeToBoth("WARNING", uuid, message, args...)
}

// SuccessCtx logs a success message with context
func (l *Logger) SuccessCtx(ctx context.Context, message string, args ...interface{}) {
	uuid := getUUIDFromContext(ctx)
	l.writeToBoth("SUCCESS", uuid, message, args...)
}

// InfoCtx logs an info message with context
func (l *Logger) InfoCtx(ctx context.Context, message string, args ...interface{}) {
	uuid := getUUIDFromContext(ctx)
	l.writeToBoth("INFO", uuid, message, args...)
}

// ErrorfCtx logs a formatted error message with context
func (l *Logger) ErrorfCtx(ctx context.Context, format string, args ...interface{}) {
	l.ErrorCtx(ctx, format, args...)
}

// WarningfCtx logs a formatted warning message with context
func (l *Logger) WarningfCtx(ctx context.Context, format string, args ...interface{}) {
	l.WarningCtx(ctx, format, args...)
}

// SuccessfCtx logs a formatted success message with context
func (l *Logger) SuccessfCtx(ctx context.Context, format string, args ...interface{}) {
	l.SuccessCtx(ctx, format, args...)
}

// InfofCtx logs a formatted info message with context
func (l *Logger) InfofCtx(ctx context.Context, format string, args ...interface{}) {
	l.InfoCtx(ctx, format, args...)
}

// LogWithMandatoryFields logs with all mandatory fields
func (l *Logger) LogWithMandatoryFields(ctx context.Context, level string, flag LogFlag, message string, body string) {
	now := time.Now()
	timestamp := now.Format("2006-01-02 15:04:05.000")

	// Extract all values from context
	transactionID := getValueFromContext(ctx, TransactionIDKey, getUUIDFromContext(ctx))
	traceID := getValueFromContext(ctx, TraceIDKey, getUUIDFromContext(ctx))
	serviceName := getValueFromContext(ctx, ServiceNameKey, "unknown")
	endpoint := getValueFromContext(ctx, EndpointKey, "unknown")
	methodType := getValueFromContext(ctx, MethodKey, "unknown")

	// Calculate execution time if start time exists
	executionTime := "0ms"
	if startTime, ok := getStartTimeFromContext(ctx); ok {
		duration := now.Sub(startTime)
		executionTime = fmt.Sprintf("%dms", duration.Milliseconds())
	}

	entry := LogEntry{
		Timestamp:     timestamp,
		LogLevel:      level,
		TransactionID: transactionID,
		ServiceName:   serviceName,
		Endpoint:      endpoint,
		MethodType:    methodType,
		ExecutionTime: executionTime,
		ServerIP:      l.ipAddress,
		TraceID:       traceID,
		Body:          body,
		Flag:          flag,
		Message:       message,
	}

	formatted := l.formatMandatoryMessage(entry)

	// Write to console if enabled
	if l.enableConsole {
		switch level {
		case "ERROR":
			fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", formatted) // Red
		case "WARNING":
			fmt.Fprintf(os.Stdout, "\033[33m%s\033[0m\n", formatted) // Yellow
		case "SUCCESS":
			fmt.Fprintf(os.Stdout, "\033[32m%s\033[0m\n", formatted) // Green
		case "INFO":
			fmt.Fprintf(os.Stdout, "\033[36m%s\033[0m\n", formatted) // Cyan
		default:
			fmt.Println(formatted)
		}
	}

	// Write to file if enabled (TANPA WARNA - plain text)
	// Config EnableFile menentukan apakah log ditulis ke file .log atau tidak
	if l.useFile && l.file != nil {
		fmt.Fprintln(l.file, formatted) // Plain text, no color codes
	}
}

// LogStart logs a START event with all mandatory fields
func (l *Logger) LogStart(ctx context.Context, level string, message string, body string) {
	l.LogWithMandatoryFields(ctx, level, FlagStart, message, body)
}

// LogStop logs a STOP event with all mandatory fields
func (l *Logger) LogStop(ctx context.Context, level string, message string, body string) {
	l.LogWithMandatoryFields(ctx, level, FlagStop, message, body)
}

// LogWithBody logs with body and all mandatory fields
func (l *Logger) LogWithBody(ctx context.Context, level string, message string, body string) {
	l.LogWithMandatoryFields(ctx, level, "", message, body)
}

// Start creates a new context with all configuration and logs a START event
// This is a convenience method that sets up everything in one call
func (l *Logger) Start(ctx context.Context, config StartConfig) context.Context {
	// If no context provided, create new one
	if ctx == nil {
		ctx = context.Background()
	}

	// Generate or use existing UUID
	if config.TransactionID == "" {
		ctx = WithNewUUID(ctx)
	} else {
		ctx = WithUUID(ctx, config.TransactionID)
		ctx = WithTransactionID(ctx, config.TransactionID)
	}

	// Set service name if provided
	if config.ServiceName != "" {
		ctx = WithServiceName(ctx, config.ServiceName)
	}

	// Set endpoint if provided
	if config.Endpoint != "" {
		ctx = WithEndpoint(ctx, config.Endpoint)
	}

	// Set method if provided
	if config.Method != "" {
		ctx = WithMethod(ctx, config.Method)
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

	// Log START event
	l.LogStart(ctx, level, message, config.Body)

	return ctx
}

// Stop logs a STOP event using the context from Start
func (l *Logger) Stop(ctx context.Context, level string, message string, body string) {
	if message == "" {
		message = "Request completed"
	}
	if level == "" {
		level = "SUCCESS"
	}
	l.LogStop(ctx, level, message, body)
}
