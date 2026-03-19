// Package logfluxecho provides LogFlux middleware for the Echo web framework.
//
// Usage:
//
//	e := echo.New()
//	e.Use(logfluxecho.Middleware())
package logfluxecho

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/logflux-io/logflux-go-sdk/v3"
)

// Middleware returns an Echo middleware that creates a span per request,
// captures panics, and records request metadata.
func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			path := c.Path() // route pattern
			if path == "" {
				path = req.URL.Path
			}

			span := logflux.ContinueFromRequest(req, "http.server", req.Method+" "+path)
			span.SetAttribute("http.method", req.Method)
			span.SetAttribute("http.url", req.URL.String())
			span.SetAttribute("http.route", path)
			if ip := c.RealIP(); ip != "" {
				span.SetAttribute("net.peer.ip", ip)
			}

			defer func() {
				if r := recover(); r != nil {
					span.SetStatus("error")
					span.SetAttribute("error.message", fmt.Sprintf("%v", r))
					_ = span.End()

					logflux.CaptureErrorWithAttrs(
						fmt.Errorf("panic: %v", r),
						logflux.Fields{
							"http.method": req.Method,
							"http.url":    req.URL.String(),
							"http.route":  path,
						},
					)
					panic(r) // re-panic for Echo's recovery middleware
				}
			}()

			err := next(c)

			status := c.Response().Status
			span.SetAttribute("http.status_code", fmt.Sprintf("%d", status))
			if status >= 500 || err != nil {
				span.SetStatus("error")
				if err != nil {
					span.SetAttribute("error.message", err.Error())
				}
			}
			_ = span.End()

			return err
		}
	}
}
