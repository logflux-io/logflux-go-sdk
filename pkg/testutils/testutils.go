package testutils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/handshake"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
)

// TestingInterface defines the interface that both *testing.T and *testing.B implement
type TestingInterface interface {
	TempDir() string
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Helper()
}

// TestConfig holds common test configuration
type TestConfig struct {
	Node      string
	APIKey    string
}

// DefaultTestConfig returns a default test configuration
func DefaultTestConfig() TestConfig {
	return TestConfig{
		Node:   "test-node",
		APIKey: "lf_test_api_key",
	}
}

// MockServer represents a mock LogFlux server for testing
type MockServer struct {
	Server      *httptest.Server
	Requests    []LogRequest
	Responses   []LogResponse
	mu          sync.RWMutex
	StatusCode  int
	Delay       time.Duration
	FailCount   int
	failCounter int

	// For handshake simulation
	rsaPrivateKey *rsa.PrivateKey
	aesKey        []byte
}

// LogRequest represents a received log request
type LogRequest struct {
	Method           string
	Path             string
	Headers          map[string]string
	Body             models.LogEntry
	Timestamp        time.Time
	DecryptedPayload string
}

// LogResponse represents a response to send
type LogResponse struct {
	StatusCode int
	Body       map[string]interface{}
}

// NewMockServer creates a new mock server for testing
func NewMockServer(t TestingInterface) *MockServer {
	mock := &MockServer{
		Requests:   []LogRequest{},
		Responses:  []LogResponse{},
		StatusCode: http.StatusAccepted,
		Delay:      0,
		FailCount:  0,
	}

	mock.Server = httptest.NewServer(http.HandlerFunc(mock.handleRequest))
	return mock
}

// handleRequest handles incoming requests to the mock server
func (m *MockServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add delay if configured
	if m.Delay > 0 {
		time.Sleep(m.Delay)
	}

	// Simulate failures only for log ingestion requests, not handshake requests
	if m.FailCount > 0 && m.failCounter < m.FailCount && r.URL.Path == "/v1/ingest" {
		m.failCounter++

		// Record failed request
		failedRequest := LogRequest{
			Method:           r.Method,
			Path:             r.URL.Path,
			Headers:          make(map[string]string),
			Timestamp:        time.Now(),
			DecryptedPayload: "failed request",
		}

		// Copy headers
		for key, values := range r.Header {
			if len(values) > 0 {
				failedRequest.Headers[key] = values[0]
			}
		}

		m.Requests = append(m.Requests, failedRequest)

		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal server error"))
		return
	}

	// Handle different endpoints
	switch r.URL.Path {
	case "/v1/ingest":
		m.handleIngest(w, r)
	case "/health":
		m.handleHealth(w, r)
	case "/version":
		m.handleVersion(w, r)
	case "/v1/handshake/init":
		m.handleHandshakeInit(w, r)
	case "/v1/handshake/complete":
		m.handleHandshakeComplete(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Not found"))
	}
}

// handleIngest handles log ingestion requests
func (m *MockServer) handleIngest(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var logEntry models.LogEntry
	if err := json.NewDecoder(r.Body).Decode(&logEntry); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid JSON"))
		return
	}

	// For E2E testing, we'll use a simple placeholder since proper decryption is complex
	decryptedPayload := "test-message-" + logEntry.Node

	// Store request
	request := LogRequest{
		Method:           r.Method,
		Path:             r.URL.Path,
		Headers:          make(map[string]string),
		Body:             logEntry,
		Timestamp:        time.Now(),
		DecryptedPayload: decryptedPayload,
	}

	// Copy headers
	for key, values := range r.Header {
		if len(values) > 0 {
			request.Headers[key] = values[0]
		}
	}

	m.Requests = append(m.Requests, request)

	// Send response
	response := LogResponse{
		StatusCode: m.StatusCode,
		Body: map[string]interface{}{
			"id":        len(m.Requests),
			"timestamp": time.Now().Format(time.RFC3339),
			"status":    "success",
		},
	}

	if len(m.Responses) > 0 {
		response = m.Responses[0]
		m.Responses = m.Responses[1:]
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.StatusCode)
	_ = json.NewEncoder(w).Encode(response.Body)
}

// handleHealth handles health check requests
func (m *MockServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// handleVersion handles version requests
func (m *MockServer) handleVersion(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"api_version":         "v1",
		"supported_versions":  []string{"v1"},
		"deprecated_versions": []string{},
		"service":             "logflux-ingestor",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// GetRequests returns all received requests
func (m *MockServer) GetRequests() []LogRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	requests := make([]LogRequest, len(m.Requests))
	copy(requests, m.Requests)
	return requests
}

// GetRequestCount returns the number of received requests
func (m *MockServer) GetRequestCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.Requests)
}

// ClearRequests clears all stored requests
func (m *MockServer) ClearRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Requests = []LogRequest{}
}

// SetFailCount sets the number of requests that should fail
func (m *MockServer) SetFailCount(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FailCount = count
	m.failCounter = 0
}

// SetDelay sets the delay for each request
func (m *MockServer) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Delay = delay
}

// SetStatusCode sets the status code for responses
func (m *MockServer) SetStatusCode(code int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StatusCode = code
}


// AddResponse adds a custom response to the queue
func (m *MockServer) AddResponse(response LogResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Responses = append(m.Responses, response)
}

// Close closes the mock server
func (m *MockServer) Close() {
	m.Server.Close()
}

// WaitForRequests waits for a specific number of requests with timeout
func (m *MockServer) WaitForRequests(count int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if m.GetRequestCount() >= count {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for %d requests, got %d", count, m.GetRequestCount())
}

// CreateTempConfigFile creates a temporary config file for testing
func CreateTempConfigFile(t TestingInterface, config map[string]interface{}) string {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".logflux.io")

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	return configPath
}

// SetupTestEnvironment sets up environment variables for testing
func SetupTestEnvironment(t TestingInterface, vars map[string]string) func() {
	originalVars := make(map[string]string)

	// Save original values
	for key := range vars {
		originalVars[key] = os.Getenv(key)
	}

	// Set test values
	for key, value := range vars {
		os.Setenv(key, value)
	}

	// Return cleanup function
	return func() {
		for key, originalValue := range originalVars {
			if originalValue == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, originalValue)
			}
		}
	}
}

// AssertEventually asserts that a condition becomes true within a timeout
func AssertEventually(t TestingInterface, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("Condition was not met within timeout: %s", message)
}

// AssertNever asserts that a condition never becomes true within a timeout
func AssertNever(t TestingInterface, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			t.Fatalf("Condition should not be true: %s", message)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// GenerateTestMessages generates test messages for batch testing
func GenerateTestMessages(count int, level int) []string {
	messages := make([]string, count)
	for i := 0; i < count; i++ {
		messages[i] = fmt.Sprintf("Test message %d with level %d", i+1, level)
	}
	return messages
}

// CompareLogs compares two log entries for equality
func CompareLogs(t TestingInterface, expected, actual LogRequest) {
	if expected.Method != actual.Method {
		t.Errorf("Method mismatch: expected %s, got %s", expected.Method, actual.Method)
	}

	if expected.Path != actual.Path {
		t.Errorf("Path mismatch: expected %s, got %s", expected.Path, actual.Path)
	}

	if expected.Body.Node != actual.Body.Node {
		t.Errorf("Node mismatch: expected %s, got %s", expected.Body.Node, actual.Body.Node)
	}

	if expected.Body.Level != actual.Body.Level {
		t.Errorf("Level mismatch: expected %d, got %d", expected.Body.Level, actual.Body.Level)
	}
}

// handleHandshakeInit handles the handshake initialization request
func (m *MockServer) handleHandshakeInit(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req handshake.HandshakeInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid JSON"))
		return
	}

	// Generate a test RSA key pair for the mock server
	rsaPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("Failed to generate RSA key: %v", err)))
		return
	}

	// Convert public key to PEM for the mock server
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&rsaPrivateKey.PublicKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("Failed to marshal public key: %v", err)))
		return
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	// For E2E testing, we don't need to store the private key
	// TODO: Store private key for proper handshake simulation

	// Send response
	response := handshake.HandshakeInitResponse{
		PublicKey: string(publicKeyPEM),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// handleHandshakeComplete handles the handshake completion request
func (m *MockServer) handleHandshakeComplete(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req handshake.HandshakeCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid JSON"))
		return
	}

	// For E2E testing, just generate a mock key UUID
	// TODO: Implement proper RSA decryption for full E2E testing
	keyUUID := make([]byte, 16)
	_, _ = rand.Read(keyUUID)
	keyUUIDStr := base64.StdEncoding.EncodeToString(keyUUID)

	// Send response
	response := handshake.HandshakeCompleteResponse{
		Status:  "success",
		KeyUUID: keyUUIDStr,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}
