package main

import (
	"errors"
	"os"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/adapters"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/client"
)

func main() {
	// Set up environment variables
	os.Setenv("LOGFLUX_API_KEY", "lf_your_api_key_here")
	// Server URL removed - using discovery service

	// Create a LogFlux client
	resilientClient, err := client.NewResilientClientFromEnvWithHandshake("logrus-example")
	if err != nil {
		panic(err)
	}
	defer resilientClient.Close()

	// Create a logrus logger adapter
	logger := adapters.NewLogrusLogger(resilientClient)

	// Set the logging level
	logger.SetLevel(adapters.LogrusDebugLevel)

	// Use the logger just like logrus
	logger.Debug("This is a debug message")
	logger.Info("This is an info message")
	logger.Warn("This is a warning message")
	logger.Error("This is an error message")

	// Use structured logging with fields
	logger.WithField("user_id", 1234).
		WithField("action", "login").
		Info("User logged in")

	// Use multiple fields
	logger.WithFields(map[string]interface{}{
		"component": "auth",
		"method":    "POST",
		"endpoint":  "/login",
		"status":    200,
	}).Info("Request completed")

	// Error handling
	err = errors.New("something went wrong")
	logger.WithError(err).Error("Operation failed")

	// Formatted logging
	logger.Infof("Processing %d items", 42)
	logger.Warnf("Memory usage: %.2f%%", 85.5)

	// Chain multiple fields
	logger.WithField("request_id", "abc123").
		WithField("user_id", 5678).
		WithField("duration", "250ms").
		Info("Request processed successfully")

	// Use different log levels
	logger.Trace("Trace message")
	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warning message")
	logger.Error("Error message")

	// Wait a bit for async processing
	time.Sleep(2 * time.Second)

	// Check statistics
	stats := resilientClient.GetStats()
	logger.WithFields(map[string]interface{}{
		"sent":    stats.EntriesSent,
		"dropped": stats.EntriesDropped,
		"queued":  stats.QueueSize,
	}).Info("LogFlux statistics")
}
