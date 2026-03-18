package main

import (
	"fmt"
	"log"
	"time"

	logflux "github.com/logflux-io/logflux-go-sdk/v3"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/client"
)

func main() {
	// Example 1: Using the client with automatic key negotiation
	fmt.Println("=== Example 1: Client with automatic key negotiation ===")

	// Initialize client (always performs handshake)
	c, err := client.NewClient("lf_your_api_key_here", "example-node")
	if err != nil {
		log.Fatalf("Failed to initialize client with handshake: %v", err)
	}
	defer c.Close()

	// Send some logs
	if err := c.Info("This message is encrypted with the negotiated key"); err != nil {
		log.Printf("Failed to send log: %v", err)
	}

	// Client is ready to use
	fmt.Println("Client initialized successfully")

	// Example 2: Using resilient client (always with handshake)
	fmt.Println("\n=== Example 2: Resilient client with automatic key negotiation ===")

	config := client.DefaultResilientClientConfig()
	config.Node = "resilient-example"
	config.APIKey = "lf_your_api_key_here"

	rc, err := client.NewResilientClientWithHandshake(config)
	if err != nil {
		log.Fatalf("Failed to create resilient client: %v", err)
	}
	defer rc.Close()

	// Send logs asynchronously
	for i := 0; i < 5; i++ {
		msg := fmt.Sprintf("Resilient log message #%d with negotiated encryption", i+1)
		if err := rc.Info(msg); err != nil {
			log.Printf("Failed to queue log: %v", err)
		}
	}

	// Wait a bit for logs to be sent
	time.Sleep(2 * time.Second)

	// Example 3: Global initialization
	fmt.Println("\n=== Example 3: Global initialization ===")

	// Initialize LogFlux globally
	if err := logflux.InitSimple("lf_your_api_key_here", "global-app"); err != nil {
		log.Fatalf("Failed to initialize LogFlux: %v", err)
	}
	defer logflux.Close()

	// Use global functions
	logflux.Info("Global logging with negotiated key")
	logflux.Warning("All logs use the secure AES key")

	fmt.Println("\n✅ All examples completed!")
}
