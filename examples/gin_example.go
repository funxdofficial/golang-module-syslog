//go:build gin_example
// +build gin_example

package main

import (
	"context"

	"github.com/funxdofficial/golang-module-syslog/logger"
	"github.com/gin-gonic/gin"
)

var log *logger.Logger

func init() {
	config := &logger.LoggerConfig{
		LogFile: "app.log",
		Type:    logger.LogTypeAll, // Console + File
	}
	var err error
	log, err = logger.StartLogger(config)
	if err != nil {
		panic(err)
	}
}

// GinRequestInfo implements HTTPRequestInfo for Gin framework
type GinRequestInfo struct {
	c *gin.Context
}

func (r *GinRequestInfo) Method() string {
	return r.c.Request.Method
}

func (r *GinRequestInfo) Path() string {
	return r.c.Request.URL.Path
}

func (r *GinRequestInfo) Body() string {
	// Gin sudah handle body reading
	return ""
}

func (r *GinRequestInfo) Header(key string) string {
	return r.c.GetHeader(key)
}

func (r *GinRequestInfo) Context() context.Context {
	return r.c.Request.Context()
}

// GinMiddleware untuk Gin framework
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create request info
		reqInfo := &GinRequestInfo{c: c}

		// Start logging
		config := logger.StartConfig{
			ServiceName: "gin-service",
			// Method dan Endpoint otomatis dari request
		}
		ctx := log.StartFromHTTPRequestInfo(reqInfo, config)

		// Update context
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Stop logging
		level := "SUCCESS"
		if c.Writer.Status() >= 400 {
			level = "ERROR"
		} else if c.Writer.Status() >= 300 {
			level = "WARNING"
		}

		log.Stop(ctx, level, "Request completed", "")
	}
}

func main() {
	defer log.Close()

	r := gin.Default()
	r.Use(GinMiddleware())

	r.GET("/users", func(c *gin.Context) {
		log.InfoCtx(c.Request.Context(), "Getting users")
		c.JSON(200, gin.H{"users": []string{"user1", "user2"}})
	})

	r.POST("/users", func(c *gin.Context) {
		log.InfoCtx(c.Request.Context(), "Creating user")
		c.JSON(201, gin.H{"message": "User created"})
	})

	r.Run(":8080")
}
