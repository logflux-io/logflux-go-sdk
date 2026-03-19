package main

import (
	"fmt"
	"log"
	"os"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/crypto"
)

func main() {
	// Create client with handshake
	logClient, err := client.NewClientFromEnv("fingerprint-example")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer logClient.Close()

	// Get the server's public key fingerprint
	fingerprint := logClient.GetServerPublicKeyFingerprint()
	fmt.Printf("Server Public Key Fingerprint: %s\n", fingerprint)

	// Get the server's public key PEM
	publicKeyPEM := logClient.GetServerPublicKeyPEM()
	fmt.Printf("\nServer Public Key:\n%s\n", publicKeyPEM)

	// You can also generate the fingerprint yourself from the PEM
	fingerprintFromPEM, err := crypto.GeneratePublicKeyFingerprintFromPEM(publicKeyPEM)
	if err != nil {
		log.Printf("Failed to generate fingerprint from PEM: %v", err)
	} else {
		fmt.Printf("\nGenerated Fingerprint: %s\n", fingerprintFromPEM)
		fmt.Printf("Fingerprints Match: %v\n", fingerprint == fingerprintFromPEM)
	}

	// Send a test log
	err = logClient.SendLog("Testing fingerprint verification")
	if err != nil {
		log.Printf("Failed to send log: %v", err)
	} else {
		fmt.Println("\nLog sent successfully!")
	}

	// Example of how you might verify a known fingerprint
	expectedFingerprint := os.Getenv("EXPECTED_SERVER_FINGERPRINT")
	if expectedFingerprint != "" {
		if fingerprint == expectedFingerprint {
			fmt.Println("\n✓ Server fingerprint matches expected value")
		} else {
			fmt.Printf("\n✗ Server fingerprint mismatch!\n")
			fmt.Printf("  Expected: %s\n", expectedFingerprint)
			fmt.Printf("  Got:      %s\n", fingerprint)
			// In a real application, you might want to abort the connection here
		}
	}
}
