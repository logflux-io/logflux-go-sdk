package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestParseRSAPublicKeyAndEncrypt(t *testing.T) {
	// Generate RSA key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	pub, err := ParseRSAPublicKey(string(pemBytes))
	if err != nil {
		t.Fatalf("ParseRSAPublicKey: %v", err)
	}

	// Encrypt some data
	ct, err := EncryptWithRSA(pub, []byte("secret"))
	if err != nil || ct == "" {
		t.Fatalf("EncryptWithRSA failed: %v", err)
	}

	// Fingerprint
	fp, err := GeneratePublicKeyFingerprint(pub)
	if err != nil || fp == "" {
		t.Fatalf("GeneratePublicKeyFingerprint failed: %v", err)
	}
}
