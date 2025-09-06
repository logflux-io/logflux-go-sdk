package utils

import (
	"os"
)

// IsAgentRunning checks if the LogFlux agent is running
func IsAgentRunning() bool {
	// Try to stat the default socket
	socketPath := "/tmp/logflux-agent.sock"
	_, err := os.Stat(socketPath)
	return err == nil
}
