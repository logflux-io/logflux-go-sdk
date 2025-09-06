package utils

import (
	"os"
	"testing"
)

func TestIsAgentRunning(t *testing.T) {
	// Test when socket doesn't exist (normal case)
	running := IsAgentRunning()
	if running {
		t.Log("Agent appears to be running (socket exists)")
	} else {
		t.Log("Agent not running (socket doesn't exist)")
	}

	// Create a test socket file to simulate running agent
	socketPath := "/tmp/logflux-agent.sock"
	f, err := os.Create(socketPath)
	if err != nil {
		t.Skipf("Cannot create test socket file: %v", err)
	}
	f.Close()
	defer os.Remove(socketPath)

	// Should detect the socket
	running = IsAgentRunning()
	if !running {
		t.Error("Expected IsAgentRunning to return true when socket exists")
	}
}
