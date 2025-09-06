package main

import (
	"context"
	"log"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

func main() {
	// Create a simple client (connects to /tmp/logflux-agent.sock by default)
	c := client.NewUnixClient("/tmp/logflux-agent.sock")

	ctx := context.Background()
	if err := c.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer c.Close()

	// Send a log entry
	entry := types.NewLogEntry("Hello, LogFlux!", "example-app").
		WithLogLevel(types.LevelInfo).
		WithSource("basic-example")

	if err := c.SendLogEntry(entry); err != nil {
		log.Fatalf("Failed to send log: %v", err)
	}

	log.Println("Log sent successfully!")
}
