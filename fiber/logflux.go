// Package logfluxfiber provides LogFlux middleware for the Fiber web framework.
//
// Usage:
//
//	app := fiber.New()
//	app.Use(logfluxfiber.Middleware())
package logfluxfiber

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/logflux-io/logflux-go-sdk/v3"
)

// Middleware returns a Fiber middleware that creates a span per request,
// captures panics, and records request metadata.
func Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Fiber doesn't use net/http.Request, so we build trace context manually
		traceHeader := c.Get(logflux.TraceHeader)
		var span *logflux.Span
		if tc := logflux.ParseTraceHeader(traceHeader); tc != nil {
			span = logflux.StartSpanWithTraceID(tc.TraceID, "http.server", c.Method()+" "+c.Route().Path)
		} else {
			span = logflux.StartSpan("http.server", c.Method()+" "+c.Route().Path)
		}

		span.SetAttribute("http.method", c.Method())
		span.SetAttribute("http.url", c.OriginalURL())
		span.SetAttribute("http.route", c.Route().Path)
		if ip := c.IP(); ip != "" {
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
						"http.method": c.Method(),
						"http.url":    c.OriginalURL(),
						"http.route":  c.Route().Path,
					},
				)

				_ = c.SendStatus(http.StatusInternalServerError)
				return
			}
		}()

		err := c.Next()

		span.SetAttribute("http.status_code", fmt.Sprintf("%d", c.Response().StatusCode()))
		if c.Response().StatusCode() >= 500 || err != nil {
			span.SetStatus("error")
			if err != nil {
				span.SetAttribute("error.message", err.Error())
			}
		}
		_ = span.End()

		return err
	}
}
