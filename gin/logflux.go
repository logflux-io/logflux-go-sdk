// Package logfluxgin provides LogFlux middleware for the Gin web framework.
//
// Usage:
//
//	r := gin.Default()
//	r.Use(logfluxgin.Middleware())
package logfluxgin

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/logflux-io/logflux-go-sdk/v3"
)

// Middleware returns a Gin middleware that creates a span per request,
// captures panics, and records request metadata.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		span := logflux.ContinueFromRequest(c.Request, "http.server", c.Request.Method+" "+c.FullPath())
		span.SetAttribute("http.method", c.Request.Method)
		span.SetAttribute("http.url", c.Request.URL.String())
		span.SetAttribute("http.route", c.FullPath())
		if clientIP := c.ClientIP(); clientIP != "" {
			span.SetAttribute("net.peer.ip", clientIP)
		}

		defer func() {
			if r := recover(); r != nil {
				span.SetStatus("error")
				span.SetAttribute("error.message", fmt.Sprintf("%v", r))
				_ = span.End()

				logflux.CaptureErrorWithAttrs(
					fmt.Errorf("panic: %v", r),
					logflux.Fields{
						"http.method": c.Request.Method,
						"http.url":    c.Request.URL.String(),
						"http.route":  c.FullPath(),
					},
				)

				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}

			span.SetAttribute("http.status_code", fmt.Sprintf("%d", c.Writer.Status()))
			if c.Writer.Status() >= 500 {
				span.SetStatus("error")
			}
			_ = span.End()
		}()

		c.Next()
	}
}
