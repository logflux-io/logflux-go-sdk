package crypto

import (
	"encoding/base64"
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey error: %v", err)
	}

	e := NewEncryptor(key)
	plaintext := "hello world"

	res, err := e.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	if res.Payload == "" || res.Nonce == "" {
		t.Fatalf("expected non-empty payload and nonce")
	}

	out, err := e.Decrypt(res.Payload, res.Nonce)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}

	if out != plaintext {
		t.Fatalf("roundtrip mismatch: got %q want %q", out, plaintext)
	}
}

func TestEncryptDecrypt_WithCompression(t *testing.T) {
	key, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("GenerateAESKey error: %v", err)
	}

	e := NewEncryptor(key)
	// Use a larger repetitive string to benefit compression
	plaintext := "aaaaabbbbbcccccdddddeeeee" + "fffffggggghhhhh" + "iiiiijjjjjkkkkk"

	res, err := e.EncryptWithCompression(plaintext, true)
	if err != nil {
		t.Fatalf("EncryptWithCompression error: %v", err)
	}

	out, err := e.DecryptWithCompression(res.Payload, res.Nonce, true)
	if err != nil {
		t.Fatalf("DecryptWithCompression error: %v", err)
	}

	if out != plaintext {
		t.Fatalf("roundtrip mismatch with compression: got %q want %q", out, plaintext)
	}
}

func TestDecrypt_InvalidNonce(t *testing.T) {
	key, _ := GenerateAESKey()
	e := NewEncryptor(key)

	res, err := e.Encrypt("msg")
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	// Corrupt nonce length by trimming bytes
	nonceBytes, _ := base64.StdEncoding.DecodeString(res.Nonce)
	if len(nonceBytes) == 0 {
		t.Fatalf("nonce decode failed")
	}
	badNonce := base64.StdEncoding.EncodeToString(nonceBytes[:len(nonceBytes)-1])

	if _, err := e.Decrypt(res.Payload, badNonce); err == nil {
		t.Fatalf("expected error for invalid nonce size")
	}
}
