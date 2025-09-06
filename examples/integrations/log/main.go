package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	log_integration "github.com/logflux-io/logflux-go-sdk/pkg/integrations/log"
)

func main() {
	// Create LogFlux batch client
	batchConfig := config.DefaultBatchConfig()
	batchConfig.MaxBatchSize = 10
	batchConfig.FlushInterval = 2 * time.Second

	batchClient := client.NewBatchUnixClient("/tmp/logflux-agent.sock", batchConfig)

	// Connect to LogFlux agent
	ctx := context.Background()
	if err := batchClient.Connect(ctx); err != nil {
		panic("Failed to connect to LogFlux agent: " + err.Error())
	}
	defer batchClient.Close()

	// Create LogFlux log writer
	logWriter := log_integration.NewWriter(batchClient, "log-example")

	// Option 1: Replace standard log output entirely
	log.SetOutput(logWriter)
	log.Println("This goes only to LogFlux")

	// Option 2: Send to both LogFlux and stdout (more common)
	multiWriter := logWriter.MultiWriter(os.Stdout)
	log.SetOutput(multiWriter)

	// Use standard log package normally - logs will go to both stdout and LogFlux
	log.Println("Application started")
	log.Printf("User %d logged in", 123)
	log.Println("Processing request...")
	log.Printf("Operation completed in %v", 250*time.Millisecond)

	// You can also create custom loggers
	customLogger := log.New(logWriter, "[CUSTOM] ", log.LstdFlags)
	customLogger.Println("This is from a custom logger")

	// Give time for batch to flush
	time.Sleep(3 * time.Second)
}
