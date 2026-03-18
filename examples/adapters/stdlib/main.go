package main

import (
	"log"
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
	resilientClient, err := client.NewResilientClientFromEnvWithHandshake("stdlib-example")
	if err != nil {
		log.Fatalf("Failed to create LogFlux client: %v", err)
	}
	defer resilientClient.Close()

	// Create a standard library logger adapter
	logger := adapters.NewStdlibLogger(resilientClient, "[STDLIB] ")

	// Use the logger just like the standard library log package
	logger.Print("This is a print message")
	logger.Printf("This is a formatted message: %d", 42)
	logger.Println("This is a println message")

	// You can also replace the standard library's default logger
	adapters.ReplaceStandardLogger(resilientClient, "[DEFAULT] ")

	// Now all standard log package calls will go to LogFlux
	log.Print("This goes to LogFlux via standard library")
	log.Printf("Formatted message: %s", "hello world")

	// Wait a bit for async processing
	time.Sleep(2 * time.Second)

	// Check statistics
	stats := resilientClient.GetStats()
	log.Printf("Sent: %d, Dropped: %d, Queued: %d",
		stats.EntriesSent, stats.EntriesDropped, stats.QueueSize)
}
