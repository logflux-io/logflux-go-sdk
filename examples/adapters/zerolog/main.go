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
	resilientClient, err := client.NewResilientClientFromEnvWithHandshake("zerolog-example")
	if err != nil {
		panic(err)
	}
	defer resilientClient.Close()

	// Create a zerolog logger adapter
	logger := adapters.NewZerologLogger(resilientClient)
	defer logger.Close()

	// Set the logging level
	logger = logger.Level(adapters.ZerologDebugLevel)

	// Use the logger just like zerolog
	logger.Debug().Msg("This is a debug message")
	logger.Info().Msg("This is an info message")
	logger.Warn().Msg("This is a warning message")
	logger.Error().Msg("This is an error message")

	// Use structured logging with fields
	logger.Info().
		Str("user_id", "1234").
		Str("action", "login").
		Int("attempts", 1).
		Msg("User logged in")

	// Use multiple fields
	logger.Info().
		Str("component", "auth").
		Str("method", "POST").
		Str("endpoint", "/login").
		Int("status", 200).
		Msg("Request completed")

	// Error handling
	err = errors.New("something went wrong")
	logger.Error().
		Err(err).
		Str("operation", "database_query").
		Msg("Operation failed")

	// Different data types
	logger.Info().
		Str("string_field", "hello").
		Int("int_field", 42).
		Int64("int64_field", 1234567890).
		Float64("float_field", 3.14159).
		Bool("bool_field", true).
		Time("time_field", time.Now()).
		Dur("duration_field", 150*time.Millisecond).
		Msg("Various data types")

	// Context logger with persistent fields
	contextLogger := logger.With().
		Str("service", "user-service").
		Str("version", "1.0.0")

	contextLogger.Info().
		Str("operation", "create_user").
		Str("user_id", "new_user_123").
		Msg("User created")

	// Array fields
	logger.Info().
		Strs("tags", []string{"auth", "security", "login"}).
		Msg("Tagged event")

	// Timestamp
	logger.Info().
		Timestamp().
		Str("event", "system_startup").
		Msg("System started")

	// Complex nested structure
	logger.Info().
		Str("request_id", "req_abc123").
		Str("user_id", "user_456").
		Str("endpoint", "/api/data").
		Int("response_code", 200).
		Dur("response_time", 89*time.Millisecond).
		Int64("bytes_sent", 2048).
		Float64("cpu_usage", 23.7).
		Bool("cached", true).
		Msg("API request processed")

	// Chain multiple errors
	err1 := errors.New("first error")
	err2 := errors.New("second error")
	logger.Error().
		AnErr("primary_error", err1).
		AnErr("secondary_error", err2).
		Msg("Multiple errors occurred")

	// Use hex encoding for binary data
	binaryData := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}
	logger.Info().
		Hex("data", binaryData).
		Msg("Binary data logged")

	// Interface field for complex objects
	userObj := map[string]interface{}{
		"id":    1234,
		"name":  "John Doe",
		"email": "john@example.com",
	}
	logger.Info().
		Interface("user", userObj).
		Msg("User object")

	// Printf-style logging
	logger.Printf("Processing %d items", 42)

	// Wait a bit for async processing
	time.Sleep(2 * time.Second)

	// Check statistics
	stats := resilientClient.GetStats()
	logger.Info().
		Int64("sent", stats.EntriesSent).
		Int64("dropped", stats.EntriesDropped).
		Int64("queued", stats.QueueSize).
		Msg("LogFlux statistics")
}
