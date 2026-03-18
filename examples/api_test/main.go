package main

import (
	"fmt"
	"log"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
)

func main() {
	// Initialize client
	c, err := client.NewClient("lf_test_key", "test-node")
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer c.Close()

	// Test version endpoint
	fmt.Println("Testing version endpoint...")
	version, err := c.GetVersion()
	if err != nil {
		log.Printf("Version check failed: %v", err)
	} else {
		fmt.Printf("API Version: %v\n", version)
	}

	// Test health check
	fmt.Println("\nTesting health endpoint...")
	if err := c.HealthCheck(); err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Println("Health check passed")
	}

	// Test single log entry
	fmt.Println("\nTesting single log entry...")
	if err := c.SendLogWithLevel("Test log message", models.LogLevelInfo); err != nil {
		log.Printf("Failed to send log: %v", err)
	} else {
		fmt.Println("Successfully sent single log entry")
	}

	// Test batch log entries
	fmt.Println("\nTesting batch log entries...")
	messages := []client.LogMessage{
		{Message: "Batch message 1", Timestamp: time.Now(), Level: models.LogLevelDebug},
		{Message: "Batch message 2", Timestamp: time.Now(), Level: models.LogLevelInfo},
		{Message: "Batch message 3", Timestamp: time.Now(), Level: models.LogLevelWarning},
		{Message: "Batch message 4", Timestamp: time.Now(), Level: models.LogLevelError},
	}

	if err := c.SendLogBatch(messages); err != nil {
		log.Printf("Failed to send batch: %v", err)
	} else {
		fmt.Println("Successfully sent batch of log entries")
	}

	// Test all log levels
	fmt.Println("\nTesting all log levels...")
	testLevels := []struct {
		name  string
		fn    func(string) error
		level int
	}{
		{"Emergency", c.Emergency, models.LogLevelEmergency},
		{"Alert", c.Alert, models.LogLevelAlert},
		{"Critical", c.Critical, models.LogLevelCritical},
		{"Error", c.Error, models.LogLevelError},
		{"Warning", c.Warning, models.LogLevelWarning},
		{"Notice", c.Notice, models.LogLevelNotice},
		{"Info", c.Info, models.LogLevelInfo},
		{"Debug", c.Debug, models.LogLevelDebug},
	}

	for _, test := range testLevels {
		if err := test.fn(fmt.Sprintf("Test %s level message", test.name)); err != nil {
			log.Printf("Failed to send %s log: %v", test.name, err)
		} else {
			fmt.Printf("Successfully sent %s log (level %d)\n", test.name, test.level)
		}
	}

	fmt.Println("\nAll tests completed!")
}
