package crypto

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
)

const KeySize = 32

// maxDecompressSize limits gzip decompression output to prevent decompression bombs.
const maxDecompressSize = 10 << 20 // 10 MiB

// Encryptor handles AES-256-GCM encryption with a negotiated key.
// All key access is guarded by a read-write mutex for thread safety.
type Encryptor struct {
	mu     sync.RWMutex
	aesKey []byte
}

func NewEncryptor(aesKey []byte) *Encryptor {
	keyCopy := make([]byte, len(aesKey))
	copy(keyCopy, aesKey)
	return &Encryptor{aesKey: keyCopy}
}

// Close zeros the AES key material. The Encryptor must not be used after Close.
func (e *Encryptor) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := range e.aesKey {
		e.aesKey[i] = 0
	}
}

// EncryptionResult contains base64-encoded payload and nonce (legacy JSON format).
type EncryptionResult struct {
	Payload string // base64-encoded ciphertext
	Nonce   string // base64-encoded 12-byte nonce
}

// RawEncryptionResult contains raw bytes for multipart/mixed format.
type RawEncryptionResult struct {
	Ciphertext []byte // raw ciphertext (GCM sealed)
	Nonce      []byte // raw 12-byte nonce
}

// EncryptRaw encrypts plaintext and returns raw bytes (for multipart/mixed).
func (e *Encryptor) EncryptRaw(plaintext []byte, compress bool) (*RawEncryptionResult, error) {
	data := plaintext

	if compress {
		compressed, err := GzipCompress(data)
		if err != nil {
			return nil, err
		}
		data = compressed
	}

	e.mu.RLock()
	key := make([]byte, len(e.aesKey))
	copy(key, e.aesKey)
	e.mu.RUnlock()

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, data, nil)

	return &RawEncryptionResult{
		Ciphertext: ciphertext,
		Nonce:      nonce,
	}, nil
}

// EncryptWithCompression encrypts and returns base64-encoded result (legacy JSON).
func (e *Encryptor) EncryptWithCompression(plaintext string, compress bool) (*EncryptionResult, error) {
	raw, err := e.EncryptRaw([]byte(plaintext), compress)
	if err != nil {
		return nil, err
	}
	return &EncryptionResult{
		Payload: base64.StdEncoding.EncodeToString(raw.Ciphertext),
		Nonce:   base64.StdEncoding.EncodeToString(raw.Nonce),
	}, nil
}

// Encrypt encrypts plaintext without compression (legacy).
func (e *Encryptor) Encrypt(plaintext string) (*EncryptionResult, error) {
	return e.EncryptWithCompression(plaintext, false)
}

// Decrypt decrypts base64-encoded payload and nonce.
func (e *Encryptor) Decrypt(encryptedPayload, nonceBase64 string) (string, error) {
	return e.DecryptWithCompression(encryptedPayload, nonceBase64, false)
}

// DecryptWithCompression decrypts with optional decompression.
func (e *Encryptor) DecryptWithCompression(encryptedPayload, nonceBase64 string, compressed bool) (string, error) {
	payload, err := base64.StdEncoding.DecodeString(encryptedPayload)
	if err != nil {
		return "", fmt.Errorf("failed to decode payload: %w", err)
	}
	nonce, err := base64.StdEncoding.DecodeString(nonceBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	e.mu.RLock()
	key := make([]byte, len(e.aesKey))
	copy(key, e.aesKey)
	e.mu.RUnlock()

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	if len(nonce) != gcm.NonceSize() {
		return "", fmt.Errorf("invalid nonce size: expected %d, got %d", gcm.NonceSize(), len(nonce))
	}

	plaintext, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	if compressed {
		return gzipDecompress(plaintext)
	}
	return string(plaintext), nil
}

// GzipCompress compresses data with gzip.
func GzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, fmt.Errorf("failed to compress: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip: %w", err)
	}
	return buf.Bytes(), nil
}

func gzipDecompress(data []byte) (string, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer reader.Close()
	decompressed, err := io.ReadAll(io.LimitReader(reader, maxDecompressSize))
	if err != nil {
		return "", fmt.Errorf("failed to decompress: %w", err)
	}
	return string(decompressed), nil
}
