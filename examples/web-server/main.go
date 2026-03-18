package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/logger"
)

// Global logger instance
var appLogger *logger.Logger

func main() {
	// Initialize LogFlux logger
	var err error
	appLogger, err = logger.NewLoggerFromEnv("web-server", "API")
	if err != nil {
		log.Fatalf("Failed to initialize LogFlux logger: %v", err)
	}
	defer appLogger.Close()

	// Log server startup
	appLogger.Info("Web server starting up")

	// Setup HTTP routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/users", usersHandler)
	http.HandleFunc("/api/health", healthHandler)

	// Start server
	server := &http.Server{
		Addr:    ":8080",
		Handler: loggingMiddleware(http.DefaultServeMux),
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		appLogger.Info("Shutting down web server")
		server.Close()
	}()

	appLogger.Infof("Web server listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		appLogger.Errorf("Server failed to start: %v", err)
		log.Fatalf("Server error: %v", err)
	}

	appLogger.Info("Web server shutdown complete")
}

// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log incoming request
		appLogger.Infof("Request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Create response writer wrapper to capture status code
		wrapper := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapper, r)

		// Log response
		duration := time.Since(start)
		appLogger.Infof("Response: %d for %s %s in %v",
			wrapper.statusCode, r.Method, r.URL.Path, duration)
	})
}

// Response wrapper to capture status code
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Handlers
func homeHandler(w http.ResponseWriter, r *http.Request) {
	appLogger.Debug("Serving home page")

	response := map[string]interface{}{
		"message": "Welcome to LogFlux Demo Web Server",
		"time":    time.Now().Format(time.RFC3339),
		"version": "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		appLogger.Debug("Fetching users list")
		users := []map[string]interface{}{
			{"id": 1, "name": "John Doe", "email": "john@example.com"},
			{"id": 2, "name": "Jane Smith", "email": "jane@example.com"},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)

	case http.MethodPost:
		appLogger.Debug("Creating new user")

		var user map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			appLogger.Errorf("Failed to decode user data: %v", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Simulate user creation
		user["id"] = 3
		user["created_at"] = time.Now().Format(time.RFC3339)

		appLogger.Infof("Created user: %s", user["name"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)

	default:
		appLogger.Warnf("Unsupported method %s for /api/users", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	appLogger.Debug("Health check requested")

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

var startTime = time.Now()
