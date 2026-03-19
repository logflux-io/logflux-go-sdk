// Package logfluxchi provides LogFlux middleware for the Chi router.
//
// Usage:
//
//	r := chi.NewRouter()
//	r.Use(logfluxchi.Middleware)
package logfluxchi

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/logflux-io/logflux-go-sdk/v3"
)

// Middleware is a Chi middleware that creates a span per request,
// captures panics, and records request metadata.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = r.URL.Path
		}

		span := logflux.ContinueFromRequest(r, "http.server", r.Method+" "+routePattern)
		span.SetAttribute("http.method", r.Method)
		span.SetAttribute("http.url", r.URL.String())
		span.SetAttribute("http.route", routePattern)
		if ip := r.RemoteAddr; ip != "" {
			span.SetAttribute("net.peer.ip", ip)
		}

		sw := &statusWriter{ResponseWriter: w, status: 200}

		defer func() {
			if rec := recover(); rec != nil {
				span.SetStatus("error")
				span.SetAttribute("error.message", fmt.Sprintf("%v", rec))
				_ = span.End()

				logflux.CaptureErrorWithAttrs(
					fmt.Errorf("panic: %v", rec),
					logflux.Fields{
						"http.method": r.Method,
						"http.url":    r.URL.String(),
						"http.route":  routePattern,
					},
				)

				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			span.SetAttribute("http.status_code", fmt.Sprintf("%d", sw.status))
			if sw.status >= 500 {
				span.SetStatus("error")
			}
			_ = span.End()
		}()

		next.ServeHTTP(sw, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
