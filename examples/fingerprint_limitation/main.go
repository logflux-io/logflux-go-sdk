package main

import (
	"fmt"
	"log"

	logflux "github.com/logflux-io/logflux-go-sdk/v3"
)

func main() {
	// This example demonstrates the limitation of fingerprint verification
	// with global logger functions

	fmt.Println("=== Fingerprint Verification Limitation Example ===")
	fmt.Println()

	// When using the global Init functions
	err := logflux.InitFromEnv("my-app")
	if err != nil {
		log.Fatalf("Failed to initialize LogFlux: %v", err)
	}
	defer logflux.Close()

	// The global functions work great for logging
	logflux.Info("This is an info message")
	logflux.Warning("This is a warning message")
	logflux.Error("This is an error message")

	// BUT: You CANNOT access the server's fingerprint through global functions
	// The following would NOT work (these methods don't exist):
	//
	// fingerprint := logflux.GetServerFingerprint()  // ❌ Does not exist
	// publicKey := logflux.GetServerPublicKey()      // ❌ Does not exist

	fmt.Println()
	fmt.Println("✓ Global logger is working fine for logging")
	fmt.Println("✗ But you cannot access the server's fingerprint through global functions")
	fmt.Println()
	fmt.Println("To access the fingerprint, you must:")
	fmt.Println("1. Create a client directly using client.NewClient() or client.NewResilientClient()")
	fmt.Println("2. Call client.GetServerPublicKeyFingerprint() on that client instance")
	fmt.Println()
	fmt.Println("See the 'fingerprint' example for how to do this correctly.")
}
