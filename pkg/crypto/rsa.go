package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

// ParseRSAPublicKey parses a PEM-encoded RSA public key
func ParseRSAPublicKey(pemData string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}

// EncryptWithRSA encrypts data with an RSA public key using OAEP padding
func EncryptWithRSA(publicKey *rsa.PublicKey, data []byte) (string, error) {
	encrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, data, nil)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt with RSA: %w", err)
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// GenerateAESKey generates a random 32-byte AES key
func GenerateAESKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}
	return key, nil
}

// GeneratePublicKeyFingerprint generates a SHA-256 fingerprint of an RSA public key
// The fingerprint is returned as a hex-encoded string in the format:
// "SHA256:hexencodedfingerprint"
func GeneratePublicKeyFingerprint(publicKey *rsa.PublicKey) (string, error) {
	// Marshal the public key to DER format
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	// Generate SHA-256 hash
	hash := sha256.Sum256(pubKeyBytes)

	// Encode to hexadecimal
	fingerprint := fmt.Sprintf("%x", hash[:])

	return fmt.Sprintf("SHA256:%s", fingerprint), nil
}

// GeneratePublicKeyFingerprintFromPEM generates a fingerprint from a PEM-encoded public key
// This is a convenience function that combines ParseRSAPublicKey and GeneratePublicKeyFingerprint
func GeneratePublicKeyFingerprintFromPEM(pemData string) (string, error) {
	publicKey, err := ParseRSAPublicKey(pemData)
	if err != nil {
		return "", fmt.Errorf("failed to parse public key: %w", err)
	}

	return GeneratePublicKeyFingerprint(publicKey)
}
