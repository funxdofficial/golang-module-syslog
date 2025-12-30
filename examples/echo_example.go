//go:build echo_example
// +build echo_example

package main

import (
	"context"

	"github.com/funxdofficial/golang-module-syslog/logger"
	"github.com/labstack/echo/v4"
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

// EchoRequestInfo implements HTTPRequestInfo for Echo framework
type EchoRequestInfo struct {
	c echo.Context
}

func (r *EchoRequestInfo) Method() string {
	return r.c.Request().Method
}

func (r *EchoRequestInfo) Path() string {
	return r.c.Request().URL.Path
}

func (r *EchoRequestInfo) Body() string {
	return ""
}

func (r *EchoRequestInfo) Header(key string) string {
	return r.c.Request().Header.Get(key)
}

func (r *EchoRequestInfo) Context() context.Context {
	return r.c.Request().Context()
}

// EchoMiddleware untuk Echo framework
func EchoMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Create request info
			reqInfo := &EchoRequestInfo{c: c}

			// Start logging
			config := logger.StartConfig{
				ServiceName: "echo-service",
			}
			ctx := log.StartFromHTTPRequestInfo(reqInfo, config)

			// Update context
			c.SetRequest(c.Request().WithContext(ctx))

			// Process request
			err := next(c)

			// Stop logging
			level := "SUCCESS"
			if c.Response().Status >= 400 {
				level = "ERROR"
			} else if c.Response().Status >= 300 {
				level = "WARNING"
			}

			log.Stop(ctx, level, "Request completed", "")

			return err
		}
	}
}

func main() {
	defer log.Close()

	e := echo.New()
	e.Use(EchoMiddleware())

	e.GET("/users", func(c echo.Context) error {
		log.InfoCtx(c.Request().Context(), "Getting users")
		return c.JSON(200, map[string]interface{}{"users": []string{"user1", "user2"}})
	})

	e.POST("/users", func(c echo.Context) error {
		log.InfoCtx(c.Request().Context(), "Creating user")
		return c.JSON(201, map[string]string{"message": "User created"})
	})

	e.Start(":8080")
}
