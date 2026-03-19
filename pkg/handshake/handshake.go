package handshake

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/api"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/crypto"
)

// HandshakeInitRequest represents the initial request for server's public key.
// Note: APIKey is sent in both the Authorization header and the JSON body for
// backward compatibility with older ingestor versions that read it from the body.
type HandshakeInitRequest struct {
	APIKey string `json:"api_key"`
}

// HandshakeInitResponse represents the server's response with its public key
type HandshakeInitResponse struct {
	PublicKey          string           `json:"public_key"`
	SupportsMultipart  bool             `json:"supports_multipart"`
	Limits             *HandshakeLimits `json:"limits"`
}

// HandshakeCompleteRequest represents the encrypted AES key sent to server
type HandshakeCompleteRequest struct {
	APIKey          string `json:"api_key"`
	EncryptedSecret string `json:"encrypted_secret"`
}

// HandshakeCompleteResponse represents the server's acknowledgment
type HandshakeCompleteResponse struct {
	Status  string `json:"status"`
	KeyID   string `json:"key_id"`
	KeyUUID string `json:"key_uuid"` // legacy field, prefer key_id
}

// GetKeyID returns the key identifier, preferring key_id over the legacy key_uuid field.
func (r *HandshakeCompleteResponse) GetKeyID() string {
	if id := strings.TrimSpace(r.KeyID); id != "" {
		return id
	}
	return strings.TrimSpace(r.KeyUUID)
}

// HandshakeLimits contains server-enforced limits from the handshake init response.
type HandshakeLimits struct {
	MaxBatchSize   int `json:"max_batch_size"`
	MaxPayloadSize int `json:"max_payload_size"`
	MaxRequestSize int `json:"max_request_size"`
}

// HandshakeResult contains the negotiated AES key and its UUID
type HandshakeResult struct {
	AESKey               []byte
	KeyUUID              string
	ServerPublicKeyPEM   string
	ServerKeyFingerprint string
	Limits               *HandshakeLimits
	SupportsMultipart    bool
}

// ErrIngestorUnavailable is returned when the ingestor cannot be reached (connection refused, DNS failure, timeout)
var ErrIngestorUnavailable = errors.New("ingestor unavailable")

// formatNetworkError creates a concise error message without deep classification
func formatNetworkError(_ string, targetURL string, err error) error {
	// Keep ErrIngestorUnavailable for detection, but present a simple message with the underlying reason
	return fmt.Errorf("%w: cannot connect to %s: %v", ErrIngestorUnavailable, targetURL, err)
}

// Response body size limits for handshake responses.
const maxHandshakeResponseSize = 64 << 10 // 64 KiB

// PerformHandshakeWithURL performs the complete handshake with a specific handshake URL
func PerformHandshakeWithURL(handshakeURL, apiKey string, httpClient *http.Client) (*HandshakeResult, error) {
	// Step 1: Request server's public key and limits
	initResp, err := requestServerPublicKeyFromURL(handshakeURL+api.DefaultPaths.HandshakeInitSuffix, apiKey, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get server public key: %w", err)
	}

	// Step 2: Generate AES key
	aesKey, err := crypto.GenerateAESKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}

	// Step 3: Encrypt AES key with server's RSA public key
	rsaPublicKey, err := crypto.ParseRSAPublicKey(initResp.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA public key: %w", err)
	}

	fingerprint, err := crypto.GeneratePublicKeyFingerprint(rsaPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate public key fingerprint: %w", err)
	}

	encryptedSecret, err := crypto.EncryptWithRSA(rsaPublicKey, aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt AES key: %w", err)
	}

	// Step 4: Send encrypted key to server
	keyUUID, err := sendEncryptedKeyToURL(handshakeURL+api.DefaultPaths.HandshakeCompleteSuffix, apiKey, encryptedSecret, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to complete key exchange: %w", err)
	}

	return &HandshakeResult{
		AESKey:               aesKey,
		KeyUUID:              keyUUID,
		ServerPublicKeyPEM:   initResp.PublicKey,
		ServerKeyFingerprint: fingerprint,
		Limits:               initResp.Limits,
		SupportsMultipart:    initResp.SupportsMultipart,
	}, nil
}

func requestServerPublicKeyFromURL(handshakeInitURL, apiKey string, httpClient *http.Client) (*HandshakeInitResponse, error) {
	req := HandshakeInitRequest{APIKey: apiKey}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", handshakeInitURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, formatNetworkError("handshake init", handshakeInitURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxHandshakeResponseSize))
		return nil, fmt.Errorf("handshake init failed with status %d: %s", resp.StatusCode, string(body))
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxHandshakeResponseSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	validatePEM := func(key string) error {
		if !(strings.Contains(key, "-----BEGIN PUBLIC KEY-----") && strings.Contains(key, "-----END PUBLIC KEY-----")) {
			return fmt.Errorf("handshake init returned non-PEM public_key")
		}
		return nil
	}

	// Try top-level: { "public_key": "PEM", "supports_multipart": true, "limits": {...} }
	var top HandshakeInitResponse
	if err := json.Unmarshal(bodyBytes, &top); err == nil && strings.TrimSpace(top.PublicKey) != "" {
		if err := validatePEM(top.PublicKey); err != nil {
			return nil, err
		}
		return &top, nil
	}

	// Try wrapped: { "data": { "public_key": "PEM", ... } }
	var wrapped struct {
		Data    *HandshakeInitResponse `json:"data"`
		Status  string                 `json:"status"`
		Message string                 `json:"message"`
	}
	if err := json.Unmarshal(bodyBytes, &wrapped); err == nil && wrapped.Data != nil && strings.TrimSpace(wrapped.Data.PublicKey) != "" {
		if err := validatePEM(wrapped.Data.PublicKey); err != nil {
			return nil, err
		}
		return wrapped.Data, nil
	}

	snippet := string(bodyBytes)
	if len(snippet) > 200 {
		snippet = snippet[:200] + "..."
	}
	return nil, fmt.Errorf("failed to decode handshake init response (no public_key found): %s", snippet)
}

func sendEncryptedKeyToURL(handshakeCompleteURL, apiKey, encryptedSecret string, httpClient *http.Client) (string, error) {
	req := HandshakeCompleteRequest{
		APIKey:          apiKey,
		EncryptedSecret: encryptedSecret,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", handshakeCompleteURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", formatNetworkError("handshake key exchange", handshakeCompleteURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxHandshakeResponseSize))
		return "", fmt.Errorf("key exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Read body to support both top-level and wrapped formats
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxHandshakeResponseSize))
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Try top-level: { "key_id": "..." } or { "key_uuid": "..." }
	var top HandshakeCompleteResponse
	if err := json.Unmarshal(bodyBytes, &top); err == nil {
		if id := top.GetKeyID(); id != "" {
			return id, nil
		}
	}

	// Try wrapped: { "data": { "key_id": "..." } }
	var wrapped struct {
		Data    *HandshakeCompleteResponse `json:"data"`
		Status  string                     `json:"status"`
		Message string                     `json:"message"`
	}
	if err := json.Unmarshal(bodyBytes, &wrapped); err == nil && wrapped.Data != nil {
		if id := wrapped.Data.GetKeyID(); id != "" {
			return id, nil
		}
	}

	snippet := string(bodyBytes)
	if len(snippet) > 200 {
		snippet = snippet[:200] + "..."
	}
	return "", fmt.Errorf("failed to decode handshake complete response (no key_id found): %s", snippet)
}
