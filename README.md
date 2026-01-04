# Golang Module Syslog

Library logging untuk Go dengan dukungan error, success, warning, dan info logging. Library ini menyediakan logging dengan informasi lengkap termasuk timestamp, UUID v7, hostname, IP address, method, routing, dan semua field mandatory untuk production logging.

**Fitur Utama:**
- ✅ **Asynchronous Logging** - Non-blocking, high-performance logging dengan goroutine dan channel
- ✅ Support multiple web framework (Gin, Echo, Fiber, standard HTTP, dll)
- ✅ Auto-extract method dan routing dari HTTP request
- ✅ Built-in middleware untuk standard HTTP
- ✅ Logging ke console (dengan warna) dan file (tanpa warna) secara bersamaan
- ✅ Semua field mandatory untuk production logging

## Fitur

- ✅ **Asynchronous Logging**: Non-blocking logging dengan goroutine dan buffered channel (kapasitas 1000). Semua operasi logging tidak menghambat eksekusi kode utama, meningkatkan performa aplikasi secara signifikan.
- ✅ **Multiple Log Levels**: Error, Success, Warning, dan Info
- ✅ **UUID v7 Support**: Tracking setiap request/session dengan UUID v7 (time-based) menggunakan `github.com/google/uuid`
- ✅ **Context Support**: Integrasi dengan `context.Context` untuk request tracing
- ✅ **Rich Information**: Timestamp, hostname, IP address, file, line, dan function name (caller info di-capture saat pemanggilan, bukan di worker)
- ✅ **Color Output**: Warna berbeda untuk setiap level di console (dengan warna) dan file (tanpa warna)
- ✅ **File Logging**: Optional logging ke file dengan config terpisah
- ✅ **Formatted Messages**: Support untuk formatted messages (Printf style)
- ✅ **Auto Extract HTTP**: Otomatis extract method dan routing dari HTTP request
- ✅ **Multi-Framework Support**: Bisa digunakan dengan berbagai web framework (Gin, Echo, Fiber, standard HTTP, dll) melalui interface `HTTPRequestInfo`
- ✅ **Built-in Middleware**: Middleware siap pakai untuk standard HTTP
- ✅ **Mandatory Fields**: Support semua field mandatory (timestamp, level, transaction ID, service name, endpoint, method, execution time, server IP, trace ID, body, flag, message)
- ✅ **Thread-Safe**: Aman digunakan dari multiple goroutines secara bersamaan
- ✅ **Graceful Shutdown**: `Close()` method akan flush semua log yang tersisa sebelum shutdown

## Instalasi

```bash
go get github.com/funxdofficial/golang-module-syslog/logger
```

## Quick Start

### 1. Buat Logger Instance dengan Config

```go
package main

import (
    "github.com/funxdofficial/golang-module-syslog/logger"
)

func main() {
    // Buat config untuk logger
    config := &logger.LoggerConfig{
        LogFile:    "app.log",           // Path ke file log (required jika Type = "file" atau "all")
        Type:       logger.LogTypeAll,   // Type: "console", "file", atau "all"
        BufferSize: 1000,                // Buffer size untuk async channel (default: 1000, optional)
    }

    appLogger, err := logger.StartLogger(config)
    if err != nil {
        panic(err)
    }
    defer appLogger.Close() // Penting: selalu panggil Close() untuk flush semua log yang tersisa

    // Gunakan logger (semua method non-blocking)
    appLogger.Success("Aplikasi berhasil dimulai")
    appLogger.Info("Sistem logging telah diinisialisasi")
}
```

### 2. Penggunaan dengan Start() dan Config

```go
// Setup config dengan semua field mandatory
config := logger.StartConfig{
    ServiceName:   "user-service",
    Endpoint:      "/api/v1/users",
    Method:        "POST",
    TransactionID: "txn-12345",  // Optional, auto-generate jika kosong
    TraceID:       "trace-67890", // Optional
    Body:          `{"user_id": "123"}`,
    Message:       "Request started",
    Level:         "INFO",
}

// Start logging - setup context dan log START event
ctx := appLogger.Start(context.Background(), config)

// ... do processing ...

// Stop logging - execution time otomatis dihitung
appLogger.Stop(ctx, "SUCCESS", "Request completed", `{"status": "created"}`)
```

### 3. Otomatis Extract dari HTTP Request

```go
import (
    "net/http"
    "github.com/funxdofficial/golang-module-syslog/logger"
)

// Method dan Endpoint otomatis di-extract dari HTTP request
req, _ := http.NewRequest("POST", "/api/v1/users", nil)

config := logger.StartConfig{
    ServiceName: "user-service",
    // Method (POST) dan Endpoint (/api/v1/users) otomatis dari req
    Body:    `{"user_id": "789"}`,
}

ctx := appLogger.StartFromRequest(req, config)
// ... processing ...
appLogger.Stop(ctx, "SUCCESS", "Request completed", `{"status": "success"}`)
```

### 4. Menggunakan Built-in Middleware (Paling Mudah)

```go
import (
    "net/http"
    "github.com/funxdofficial/golang-module-syslog/logger"
)

var appLogger *logger.Logger

func init() {
    config := &logger.LoggerConfig{
        LogFile:    "app.log",
        Type:       logger.LogTypeAll, // Console + File
        BufferSize: 1000,              // Default buffer size (optional)
    }
    var err error
    appLogger, err = logger.StartLogger(config)
    if err != nil {
        panic(err)
    }
}

func handler(w http.ResponseWriter, r *http.Request) {
    // Context sudah ada method, endpoint, dan UUID dari middleware
    appLogger.InfoCtx(r.Context(), "Processing request")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"message": "success"}`))
}

func main() {
    defer appLogger.Close()
    mux := http.NewServeMux()
    mux.HandleFunc("/", handler)

    // Gunakan built-in middleware - otomatis handle START/STOP
    middlewareConfig := logger.MiddlewareConfig{
        ServiceName: "http-service",
        SkipPaths:   []string{"/health"}, // Skip health check
    }

    http.ListenAndServe(":8080", appLogger.StandardHTTPMiddleware(middlewareConfig)(mux))
}
```

## Format Log

### Format Dasar (tanpa mandatory fields):
```
[timestamp] [level] [uuid] [hostname@ip] [file:line:function] message
```

### Format dengan Mandatory Fields:
```
[timestamp] | [level] | [flag] | Service: service-name | [METHOD] /endpoint | TxnID: xxx | TraceID: xxx | Duration: xxxms | IP: xxx | Body: {...} | → message
```

**Contoh Output:**
```
[2025-12-30 10:46:03.663] | [INFO] | [START] | Service: user-service | [POST] /api/v1/users | TxnID: txn-12345 | TraceID: trace-67890 | IP: 10.233.98.142 | Body: {"user_id": "123"} | → Request started
[2025-12-30 10:46:03.764] | [SUCCESS] | [STOP] | Service: user-service | [POST] /api/v1/users | TxnID: txn-12345 | TraceID: trace-67890 | Duration: 101ms | IP: 10.233.98.142 | Body: {"status": "created"} | → Request completed
```

## Konfigurasi Logger

### LoggerConfig Options:

```go
config := &logger.LoggerConfig{
    LogFile:    "app.log",           // Path ke file log (required jika Type = "file" atau "all")
    Type:       logger.LogTypeAll,   // Type: "console", "file", atau "all"
    BufferSize: 1000,                // Buffer size untuk async logging channel (default: 1000, optional)
}
```

**Type Options:**
- **`logger.LogTypeConsole`** atau **`"console"`** - Hanya console (dengan warna), tidak ditulis ke file
- **`logger.LogTypeFile`** atau **`"file"`** - Hanya file (tanpa warna, plain text), tidak ditampilkan di console
- **`logger.LogTypeAll`** atau **`"all"`** - Console + File (console dengan warna, file tanpa warna) - **Recommended**

**Field Options:**
- **`LogFile`** (string, optional) - Path ke file log. Required jika `Type = "file"` atau `"all"`
- **`Type`** (LogType, required) - Type logging: `"console"`, `"file"`, atau `"all"`
- **`BufferSize`** (int, optional) - Buffer size untuk async logging channel. Default: `1000`. Semakin besar buffer, semakin banyak log yang bisa di-queue sebelum blocking. Untuk high-traffic aplikasi, bisa di-set lebih besar (misalnya 5000 atau 10000).

**Contoh:**
```go
// Console saja (default buffer size: 1000)
config := &logger.LoggerConfig{
    Type: logger.LogTypeConsole, // atau "console"
}

// File saja dengan custom buffer size
config := &logger.LoggerConfig{
    LogFile:    "app.log",
    Type:       logger.LogTypeFile, // atau "file"
    BufferSize: 5000,                // Custom buffer size untuk high-traffic
}

// Console + File (Recommended) dengan custom buffer size
config := &logger.LoggerConfig{
    LogFile:    "app.log",
    Type:       logger.LogTypeAll, // atau "all"
    BufferSize: 2000,              // Custom buffer size
}
```

**Tips Buffer Size:**
- **Default (1000)**: Cocok untuk sebagian besar aplikasi
- **5000-10000**: Untuk aplikasi dengan traffic tinggi atau banyak concurrent requests
- **< 100**: Tidak disarankan, bisa menyebabkan log di-drop jika channel penuh

### Backward Compatibility:

```go
// Cara lama masih bisa digunakan
appLogger, err := logger.NewLoggerSimple("app.log") // Console + File
appLogger, err := logger.NewLoggerSimple("")        // Console only
```

## API Reference

### Logger Instance Creation

- `StartLogger(config *LoggerConfig) (*Logger, error)` - Membuat logger dengan config
- `NewLoggerSimple(logFile string) (*Logger, error)` - Membuat logger sederhana (backward compatible)

### Basic Logging Methods

#### Tanpa Context (generate UUID baru setiap kali)
- `Error(message string, args ...interface{})` - Log error
- `Warning(message string, args ...interface{})` - Log warning
- `Success(message string, args ...interface{})` - Log success
- `Info(message string, args ...interface{})` - Log info

#### Formatted Methods (tanpa context)
- `Errorf(format string, args ...interface{})` - Log formatted error
- `Warningf(format string, args ...interface{})` - Log formatted warning
- `Successf(format string, args ...interface{})` - Log formatted success
- `Infof(format string, args ...interface{})` - Log formatted info

#### Context Methods (dengan context)
- `ErrorCtx(ctx context.Context, message string, args ...interface{})` - Log error dengan context
- `WarningCtx(ctx context.Context, message string, args ...interface{})` - Log warning dengan context
- `SuccessCtx(ctx context.Context, message string, args ...interface{})` - Log success dengan context
- `InfoCtx(ctx context.Context, message string, args ...interface{})` - Log info dengan context

#### Formatted Context Methods
- `ErrorfCtx(ctx context.Context, format string, args ...interface{})` - Log formatted error dengan context
- `WarningfCtx(ctx context.Context, format string, args ...interface{})` - Log formatted warning dengan context
- `SuccessfCtx(ctx context.Context, format string, args ...interface{})` - Log formatted success dengan context
- `InfofCtx(ctx context.Context, format string, args ...interface{})` - Log formatted info dengan context

### Mandatory Fields Methods

- `Start(ctx context.Context, config StartConfig) context.Context` - Setup context dan log START event
- `StartFromRequest(r *http.Request, config StartConfig) context.Context` - Otomatis extract method/routing dari HTTP request dan log START
- `StartFromHTTPRequestInfo(reqInfo HTTPRequestInfo, config StartConfig) context.Context` - Otomatis extract dari interface HTTPRequestInfo (untuk multi-framework)
- `Stop(ctx context.Context, level string, message string, body string)` - Log STOP event (execution time otomatis dihitung)
- `LogStart(ctx context.Context, level string, message string, body string)` - Log START event
- `LogStop(ctx context.Context, level string, message string, body string)` - Log STOP event
- `LogWithBody(ctx context.Context, level string, message string, body string)` - Log dengan body

### Middleware Methods

- `StandardHTTPMiddleware(config MiddlewareConfig) func(http.Handler) http.Handler` - Built-in middleware untuk standard HTTP

### Helper Functions

#### Context Helpers
- `WithUUID(ctx context.Context, uuid string) context.Context` - Menambahkan UUID ke context
- `WithNewUUID(ctx context.Context) context.Context` - Generate UUID baru dan tambahkan ke context
- `WithServiceName(ctx context.Context, serviceName string) context.Context` - Menambahkan service name ke context
- `WithEndpoint(ctx context.Context, endpoint string) context.Context` - Menambahkan endpoint ke context
- `WithMethod(ctx context.Context, method string) context.Context` - Menambahkan HTTP method ke context
- `WithTraceID(ctx context.Context, traceID string) context.Context` - Menambahkan trace ID ke context
- `WithTransactionID(ctx context.Context, transactionID string) context.Context` - Menambahkan transaction ID ke context
- `WithStartTime(ctx context.Context, startTime time.Time) context.Context` - Menambahkan start time untuk tracking execution time
- `WithHTTPRequest(ctx context.Context, r *http.Request) context.Context` - Otomatis extract method dan endpoint dari HTTP request

## StartConfig Fields

- `ServiceName` - Nama service (required)
- `Endpoint` - Route/endpoint (required, atau otomatis dari HTTP request)
- `Method` - HTTP method: GET, POST, PUT, DELETE, dll (required, atau otomatis dari HTTP request)
- `TransactionID` - Optional, auto-generate UUID v7 jika kosong
- `TraceID` - Optional
- `Body` - Optional, request/response body
- `Message` - Optional, default: "Request started"
- `Level` - Optional, default: "INFO"

## Warna Output

- **SUCCESS**: Hijau 🟢
- **ERROR**: Merah 🔴
- **WARNING**: Kuning 🟡
- **INFO**: Cyan 🔵

**Note:** Console menampilkan dengan warna, file ditulis tanpa warna (plain text) untuk memudahkan parsing.

## Asynchronous Logging

Logger menggunakan **asynchronous logging** dengan goroutine dan buffered channel untuk performa optimal:

### Keuntungan:
- **Non-blocking**: Semua method logging (`Error()`, `Warning()`, `Success()`, `Info()`, dll) langsung return tanpa menunggu I/O selesai
- **High Performance**: Operasi I/O (console/file) dilakukan di background, tidak menghambat eksekusi kode utama
- **Thread-Safe**: Aman digunakan dari multiple goroutines secara bersamaan
- **Buffered Channel**: Channel dengan kapasitas configurable (default: 1000) untuk menampung log messages. Bisa di-set melalui `LoggerConfig.BufferSize`
- **Graceful Shutdown**: `Close()` method akan menunggu semua log yang tersisa diproses sebelum shutdown

### Cara Kerja:
1. Saat memanggil method logging (misalnya `log.Success("message")`):
   - Caller info (file, line, function) di-capture saat pemanggilan
   - Log message dibuat dan dikirim ke buffered channel (non-blocking)
   - Method langsung return, tidak menunggu log ditulis

2. Worker goroutine (background):
   - Membaca log messages dari channel
   - Memformat dan menulis ke console/file
   - Menangani shutdown dengan flush semua log yang tersisa

### Contoh:
```go
// Semua method ini non-blocking dan langsung return
appLogger.Success("Aplikasi berhasil dimulai")
appLogger.Info("Sistem logging telah diinisialisasi")
appLogger.Warning("Peringatan")
appLogger.Error("Error terjadi")

// Logging tidak menghambat eksekusi kode berikutnya
doSomethingImportant() // Akan langsung dieksekusi, tidak menunggu log selesai
```

### Important Notes:
- **Selalu panggil `defer logger.Close()`** untuk memastikan semua log ter-flush sebelum aplikasi exit
- Jika channel penuh (sangat jarang terjadi), log akan di-drop dan error message akan ditampilkan ke stderr
- Caller info (file, line, function) di-capture saat pemanggilan method, bukan di worker goroutine, sehingga selalu akurat
- **Buffer Size**: Default adalah 1000. Untuk aplikasi dengan traffic tinggi, bisa di-set lebih besar melalui `LoggerConfig.BufferSize` (misalnya 5000 atau 10000)

## Contoh Penggunaan Lengkap

### 1. Standard HTTP dengan Built-in Middleware (Recommended)

```go
package main

import (
    "net/http"
    "github.com/funxdofficial/golang-module-syslog/logger"
)

var appLogger *logger.Logger

func init() {
    config := &logger.LoggerConfig{
        LogFile:    "app.log",
        Type:       logger.LogTypeAll, // Console + File
        BufferSize: 1000,              // Default buffer size (optional)
    }
    var err error
    appLogger, err = logger.StartLogger(config)
    if err != nil {
        panic(err)
    }
}

func handler(w http.ResponseWriter, r *http.Request) {
    // Context sudah ada method, endpoint, dan UUID dari middleware
    appLogger.InfoCtx(r.Context(), "Processing request")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"message": "success"}`))
}

func main() {
    defer appLogger.Close()
    mux := http.NewServeMux()
    mux.HandleFunc("/", handler)

    // Built-in middleware otomatis handle START/STOP logging
    middlewareConfig := logger.MiddlewareConfig{
        ServiceName: "http-service",
        SkipPaths:   []string{"/health"}, // Skip health check
    }

    http.ListenAndServe(":8080", appLogger.StandardHTTPMiddleware(middlewareConfig)(mux))
}
```

### 2. Gin Framework

```go
package main

import (
    "context"
    "github.com/funxdofficial/golang-module-syslog/logger"
    "github.com/gin-gonic/gin"
)

var appLogger *logger.Logger

func init() {
    config := &logger.LoggerConfig{
        LogFile:    "app.log",
        Type:       logger.LogTypeAll, // Console + File
        BufferSize: 1000,              // Default buffer size (optional)
    }
    var err error
    appLogger, err = logger.StartLogger(config)
    if err != nil {
        panic(err)
    }
}

// GinRequestInfo implements HTTPRequestInfo for Gin
type GinRequestInfo struct {
    c *gin.Context
}

func (r *GinRequestInfo) Method() string {
    return r.c.Request.Method
}

func (r *GinRequestInfo) Path() string {
    return r.c.Request.URL.Path
}

func (r *GinRequestInfo) Body() string { return "" }
func (r *GinRequestInfo) Header(key string) string { return r.c.GetHeader(key) }
func (r *GinRequestInfo) Context() context.Context { return r.c.Request.Context() }

func GinMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        reqInfo := &GinRequestInfo{c: c}
        config := logger.StartConfig{ServiceName: "gin-service"}
        ctx := appLogger.StartFromHTTPRequestInfo(reqInfo, config)
        c.Request = c.Request.WithContext(ctx)
        c.Next()
        
        level := "SUCCESS"
        if c.Writer.Status() >= 400 { level = "ERROR" }
        appLogger.Stop(ctx, level, "Request completed", "")
    }
}

func main() {
    defer appLogger.Close()

    r := gin.Default()
    r.Use(GinMiddleware())
    r.GET("/users", func(c *gin.Context) {
        appLogger.InfoCtx(c.Request.Context(), "Getting users")
        c.JSON(200, gin.H{"users": []string{"user1"}})
    })
    r.Run(":8080")
}
```

### 3. Echo Framework

```go
package main

import (
    "context"
    "github.com/funxdofficial/golang-module-syslog/logger"
    "github.com/labstack/echo/v4"
)

var appLogger *logger.Logger

func init() {
    config := &logger.LoggerConfig{
        LogFile:    "app.log",
        Type:       logger.LogTypeAll, // Console + File
        BufferSize: 1000,              // Default buffer size (optional)
    }
    var err error
    appLogger, err = logger.StartLogger(config)
    if err != nil {
        panic(err)
    }
}

// EchoRequestInfo implements HTTPRequestInfo for Echo
type EchoRequestInfo struct {
    c echo.Context
}

func (r *EchoRequestInfo) Method() string { return r.c.Request().Method }
func (r *EchoRequestInfo) Path() string { return r.c.Request().URL.Path }
func (r *EchoRequestInfo) Body() string { return "" }
func (r *EchoRequestInfo) Header(key string) string { return r.c.Request().Header.Get(key) }
func (r *EchoRequestInfo) Context() context.Context { return r.c.Request().Context() }

func EchoMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            reqInfo := &EchoRequestInfo{c: c}
            config := logger.StartConfig{ServiceName: "echo-service"}
            ctx := appLogger.StartFromHTTPRequestInfo(reqInfo, config)
            c.SetRequest(c.Request().WithContext(ctx))
            
            err := next(c)
            
            level := "SUCCESS"
            if c.Response().Status >= 400 { level = "ERROR" }
            appLogger.Stop(ctx, level, "Request completed", "")
            return err
        }
    }
}

func main() {
    defer appLogger.Close()

    e := echo.New()
    e.Use(EchoMiddleware())
    e.GET("/users", func(c echo.Context) error {
        appLogger.InfoCtx(c.Request().Context(), "Getting users")
        return c.JSON(200, map[string]interface{}{"users": []string{"user1"}})
    })
    e.Start(":8080")
}
```

### 4. Framework Lain (Custom Implementation)

Logger menggunakan interface `HTTPRequestInfo` untuk membuat logger bisa bekerja dengan berbagai web framework. Implementasikan interface ini untuk framework yang berbeda:

```go
type HTTPRequestInfo interface {
    Method() string      // HTTP method (GET, POST, dll)
    Path() string        // Request path/endpoint
    Body() string        // Request body (optional)
    Header(key string) string  // Get header value
    Context() context.Context  // Get context
}
```

**Contoh untuk Fiber Framework:**

```go
package main

import (
    "context"
    "github.com/funxdofficial/golang-module-syslog/logger"
    "github.com/gofiber/fiber/v2"
)

var appLogger *logger.Logger

func init() {
    config := &logger.LoggerConfig{
        LogFile:    "app.log",
        Type:       logger.LogTypeAll, // Console + File
        BufferSize: 1000,              // Default buffer size (optional)
    }
    var err error
    appLogger, err = logger.StartLogger(config)
    if err != nil {
        panic(err)
    }
}

type FiberRequestInfo struct {
    c *fiber.Ctx
}

func (r *FiberRequestInfo) Method() string {
    return r.c.Method()
}

func (r *FiberRequestInfo) Path() string {
    return r.c.Path()
}

func (r *FiberRequestInfo) Body() string {
    return string(r.c.Body())
}

func (r *FiberRequestInfo) Header(key string) string {
    return r.c.Get(key)
}

func (r *FiberRequestInfo) Context() context.Context {
    return r.c.UserContext()
}

func FiberMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        reqInfo := &FiberRequestInfo{c: c}
        config := logger.StartConfig{ServiceName: "fiber-service"}
        ctx := appLogger.StartFromHTTPRequestInfo(reqInfo, config)
        c.SetUserContext(ctx)
        
        err := c.Next()
        
        level := "SUCCESS"
        if c.Response().StatusCode() >= 400 { level = "ERROR" }
        appLogger.Stop(ctx, level, "Request completed", "")
        return err
    }
}

func main() {
    defer appLogger.Close()
    
    app := fiber.New()
    app.Use(FiberMiddleware())
    app.Get("/users", func(c *fiber.Ctx) error {
        appLogger.InfoCtx(c.UserContext(), "Getting users")
        return c.JSON(fiber.Map{"users": []string{"user1"}})
    })
    app.Listen(":8080")
}
```

**Cara Menggunakan untuk Framework Lain:**

```go
// 1. Implementasikan HTTPRequestInfo interface
reqInfo := &YourFrameworkRequestInfo{...}

// 2. Gunakan StartFromHTTPRequestInfo
config := logger.StartConfig{ServiceName: "your-service"}
ctx := appLogger.StartFromHTTPRequestInfo(reqInfo, config)

// 3. Process request
// ...

// 4. Stop logging
appLogger.Stop(ctx, "SUCCESS", "Request completed", "")
```

## Contoh Penggunaan Tambahan

### Manual Setup dengan Start() (tanpa middleware)

Jika tidak menggunakan middleware, Anda bisa setup manual:

```go
import (
    "context"
    "github.com/funxdofficial/golang-module-syslog/logger"
)

var appLogger *logger.Logger

func init() {
    config := &logger.LoggerConfig{
        LogFile:    "app.log",
        Type:       logger.LogTypeAll, // Console + File
        BufferSize: 1000,              // Default buffer size (optional)
    }
    var err error
    appLogger, err = logger.StartLogger(config)
    if err != nil {
        panic(err)
    }
}

func someFunction() {
    config := logger.StartConfig{
        ServiceName:   "user-service",
        Endpoint:      "/api/v1/users",
        Method:        "POST",
        TransactionID: "txn-12345",
        TraceID:       "trace-67890",
        Body:          `{"user_id": "123"}`,
        Message:       "Request started",
        Level:         "INFO",
    }

    ctx := appLogger.Start(context.Background(), config)
    // ... processing ...
    appLogger.Stop(ctx, "SUCCESS", "Request completed", `{"status": "created"}`)
}
```

### Minimal Config (menggunakan defaults)

```go
config := logger.StartConfig{
    ServiceName: "payment-service",
    Endpoint:    "/api/v1/payments",
    Method:      "GET",
    Body:        `{"user_id": "456"}`,
}

ctx := appLogger.Start(context.Background(), config)
// ... processing ...
appLogger.Stop(ctx, "SUCCESS", "", `{"status": "processed"}`)
```

## Mandatory Fields

Semua field berikut akan otomatis diisi dalam setiap log entry:

1. **Timestamp** - Format: `2006-01-02 15:04:05.000` (dengan milidetik)
2. **Log Level** - ERROR, WARNING, SUCCESS, INFO
3. **Transaction ID** - UUID v7 (auto-generate atau dari context/config)
4. **Service Name** - Dari config atau context
5. **Endpoint** - Dari config, context, atau HTTP request (otomatis)
6. **Method Type** - Dari config, context, atau HTTP request (otomatis)
7. **Execution Time** - Otomatis dihitung dari start time (untuk STOP event)
8. **Server IP** - Otomatis dari sistem
9. **Trace ID** - Dari config atau context
10. **Body** - Request/response body (optional)
11. **Flag** - START atau STOP (untuk Start/Stop methods)
12. **Message** - Pesan log

**Catatan:** Field yang tidak disediakan akan menggunakan nilai default atau di-generate otomatis.

## Performance & Best Practices

### Asynchronous Logging
- Semua method logging adalah **non-blocking** dan langsung return
- Operasi I/O dilakukan di background worker goroutine
- Channel buffered dengan kapasitas 1000 untuk menampung log messages
- Jika channel penuh (sangat jarang), log akan di-drop dengan error message ke stderr

### Best Practices:
1. **Selalu panggil `defer logger.Close()`** untuk memastikan semua log ter-flush sebelum aplikasi exit
2. **Jangan membuat multiple logger instance** untuk aplikasi yang sama (gunakan singleton pattern)
3. **Gunakan context** untuk tracking request yang sama dengan UUID yang sama
4. **Gunakan middleware** untuk automatic START/STOP logging pada HTTP requests

### Contoh Pattern yang Disarankan:
```go
var appLogger *logger.Logger

func init() {
    config := &logger.LoggerConfig{
        LogFile: "app.log",
        Type:    logger.LogTypeAll,
    }
    var err error
    appLogger, err = logger.StartLogger(config)
    if err != nil {
        panic(err)
    }
}

func main() {
    defer appLogger.Close() // Penting!
    // ... rest of your code
}
```

## License

MIT
