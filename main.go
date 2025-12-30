package main

import (
	"context"
	"net/http"
	"time"

	"github.com/funxdofficial/golang-module-syslog/logger"
)

func main() {
	// Create a new logger instance with config
	// Type: "all" -> log ditampilkan di console DENGAN WARNA dan ditulis ke file TANPA WARNA
	config := &logger.LoggerConfig{
		LogFile: "app.log",         // Path to log file (required jika Type = "file" atau "all")
		Type:    logger.LogTypeAll, // Type: "console", "file", atau "all"
	}

	log, err := logger.StartLogger(config)
	if err != nil {
		panic(err)
	}
	defer log.Close()

	// Alternative: Simple way (backward compatible)
	// log, err := logger.NewLoggerSimple("app.log") // Console + File
	// log, err := logger.NewLoggerSimple("")        // Console only

	// Example usage of different log levels (without context - will generate new UUID each time)
	log.Success("Aplikasi berhasil dimulai")
	log.Info("Sistem logging telah diinisialisasi")
	log.Warning("Ini adalah contoh pesan peringatan")
	log.Error("Ini adalah contoh pesan error")

	// Using formatted messages
	log.Successf("Operasi berhasil: %s", "menyimpan data")
	log.Warningf("Peringatan: %s mungkin tidak tersedia", "fitur X")
	log.Errorf("Error terjadi: %s", "gagal membaca file")

	// Example with context - all logs in the same context will share the same UUID
	ctx := logger.WithNewUUID(context.Background())
	log.InfoCtx(ctx, "Memulai request dengan UUID")
	log.SuccessCtx(ctx, "Data berhasil diproses")
	log.WarningCtx(ctx, "Koneksi database lambat")
	log.ErrorCtx(ctx, "Gagal menghubungkan ke server")

	// Example with custom UUID
	customCtx := logger.WithUUID(context.Background(), "custom-uuid-12345")
	log.InfoCtx(customCtx, "Request dengan custom UUID")
	log.SuccessCtx(customCtx, "Operasi berhasil dengan custom UUID")

	// Example with nested context (child context inherits UUID)
	childCtx := context.WithValue(customCtx, "user_id", "user123")
	log.InfoCtx(childCtx, "Child context menggunakan UUID yang sama")
	log.SuccessCtx(childCtx, "Operasi di child context")

	// Example with ALL MANDATORY FIELDS - Using Start() with config (RECOMMENDED)
	startConfig := logger.StartConfig{
		ServiceName:   "user-service",
		Endpoint:      "/api/v1/users",
		Method:        "POST",
		TransactionID: "txn-12345",
		TraceID:       "trace-67890",
		Body:          `{"user_id": "123", "action": "create"}`,
		Message:       "Request started",
		Level:         "INFO",
	}

	// Start logging - this sets up context and logs START event
	requestCtx := log.Start(context.Background(), startConfig)

	// Simulate some processing time
	time.Sleep(100 * time.Millisecond)

	// Stop logging - execution time will be calculated automatically
	log.Stop(requestCtx, "SUCCESS", "Request completed", `{"user_id": "123", "status": "created"}`)

	// Example with minimal config (using defaults)
	simpleConfig := logger.StartConfig{
		ServiceName: "payment-service",
		Endpoint:    "/api/v1/payments",
		Method:      "GET",
		Body:        `{"user_id": "456"}`,
	}
	simpleCtx := log.Start(context.Background(), simpleConfig)
	time.Sleep(50 * time.Millisecond)
	log.Stop(simpleCtx, "SUCCESS", "", `{"status": "processed"}`)

	// Example with different log levels and mandatory fields
	errorCtx := logger.WithNewUUID(context.Background())
	errorCtx = logger.WithServiceName(errorCtx, "payment-service")
	errorCtx = logger.WithEndpoint(errorCtx, "/api/v1/payments")
	errorCtx = logger.WithMethod(errorCtx, "GET")
	errorCtx = logger.WithStartTime(errorCtx, time.Now())

	time.Sleep(50 * time.Millisecond)
	log.LogWithBody(errorCtx, "ERROR", "Payment processing failed", `{"error": "insufficient_funds"}`)

	// Example: Otomatis extract method dan routing dari HTTP request
	// Simulasi HTTP request
	req, _ := http.NewRequest("POST", "/api/v1/users", nil)

	// StartFromRequest otomatis extract method (POST) dan endpoint (/api/v1/users) dari request
	requestConfig := logger.StartConfig{
		ServiceName: "user-service",
		// Method dan Endpoint akan otomatis di-extract dari req, tidak perlu di-set manual
		Body:    `{"user_id": "789"}`,
		Message: "Auto-extracted from HTTP request",
	}

	autoCtx := log.StartFromRequest(req, requestConfig)
	time.Sleep(75 * time.Millisecond)
	log.Stop(autoCtx, "SUCCESS", "Request completed", `{"status": "success"}`)

	// Example: GET request
	getReq, _ := http.NewRequest("GET", "/api/v1/products?page=1", nil)
	getConfig := logger.StartConfig{
		ServiceName: "product-service",
		// Method (GET) dan Endpoint (/api/v1/products?page=1) otomatis dari request
	}
	getCtx := log.StartFromRequest(getReq, getConfig)
	time.Sleep(30 * time.Millisecond)
	log.Stop(getCtx, "SUCCESS", "Products retrieved", `{"count": 10}`)
}
