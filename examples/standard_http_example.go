//go:build http_example
// +build http_example

package main

import (
	"net/http"

	"github.com/funxdofficial/golang-module-syslog/logger"
)

var appLogger *logger.Logger

func init() {
	config := &logger.LoggerConfig{
		LogFile: "app.log",
		Type:    logger.LogTypeAll, // Console + File
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

	// Gunakan middleware untuk standard http
	middlewareConfig := logger.MiddlewareConfig{
		ServiceName: "http-service",
		SkipPaths:   []string{"/health"}, // Skip health check
	}

	http.ListenAndServe(":8080", appLogger.StandardHTTPMiddleware(middlewareConfig)(mux))
}
