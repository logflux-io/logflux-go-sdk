package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	"github.com/logflux-io/logflux-go-sdk/pkg/types"
)

func main() {
	// Create a batch client for high-throughput logging
	batchConfig := config.DefaultBatchConfig()
	batchConfig.MaxBatchSize = 100
	batchConfig.FlushInterval = 5 * time.Second

	batchClient := client.NewBatchUnixClient("/tmp/logflux-agent.sock", batchConfig)

	ctx := context.Background()
	if err := batchClient.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer batchClient.Close()

	// Send multiple entries - they'll be automatically batched
	for i := 0; i < 1000; i++ {
		entry := types.NewLogEntry(fmt.Sprintf("Log message %d", i), "batch-example").
			WithLogLevel(types.LevelInfo)
		if err := batchClient.SendLogEntry(entry); err != nil {
			log.Printf("Failed to send log entry %d: %v", i, err)
		}
	}

	// Flush any remaining entries
	if err := batchClient.Flush(); err != nil {
		log.Fatalf("Failed to flush batch: %v", err)
	}

	log.Println("All logs sent successfully!")
}
