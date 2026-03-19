package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
)

// This demo shows how the LogEntry model is structured and serialized.
func main() {
	log.Println("=== LogFlux LogEntry Demo ===")

	// Create a sample LogEntry
	timestamp := time.Now().UTC()
	logEntry := models.LogEntry{
		Message:   "encrypted-log-payload-data",
		Timestamp: timestamp,
		Level:     models.LogLevelInfo,
		EntryType: models.EntryTypeLog,
		Node:      "demo-node",
		Labels: map[string]string{
			"service":    "demo-service",
			"component":  "auth",
			"request_id": "req-12345",
		},
		SearchTokens: []string{"auth", "login"},
	}

	// Marshal to JSON to demonstrate proper serialization
	jsonData, err := json.MarshalIndent(logEntry, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal LogEntry: %v", err)
	}

	fmt.Println("LogEntry JSON representation:")
	fmt.Println(string(jsonData))

	// Verify fields are present in the JSON
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &jsonMap); err != nil {
		log.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if node, exists := jsonMap["Node"]; exists {
		fmt.Printf("\nNode field is properly serialized: %v\n", node)
	}

	// Demonstrate unmarshaling back to struct
	var unmarshaledEntry models.LogEntry
	if err := json.Unmarshal(jsonData, &unmarshaledEntry); err != nil {
		log.Fatalf("Failed to unmarshal back to struct: %v", err)
	}

	if unmarshaledEntry.Node == logEntry.Node {
		fmt.Printf("Node field properly deserialized: %s\n", unmarshaledEntry.Node)
	} else {
		fmt.Printf("Node mismatch: expected %s, got %s\n", logEntry.Node, unmarshaledEntry.Node)
	}

	fmt.Println("\nDemo completed successfully!")
}
