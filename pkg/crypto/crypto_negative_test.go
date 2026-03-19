package crypto

import "testing"

func TestDecrypt_WrongKeyFails(t *testing.T) {
	k1, _ := GenerateAESKey()
	k2, _ := GenerateAESKey()
	e1 := NewEncryptor(k1)
	e2 := NewEncryptor(k2)

	res, err := e1.Encrypt("secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if _, err := e2.Decrypt(res.Payload, res.Nonce); err == nil {
		t.Fatalf("expected decryption failure with wrong key")
	}
}

func TestDecrypt_BadBase64Fails(t *testing.T) {
	k, _ := GenerateAESKey()
	e := NewEncryptor(k)
	if _, err := e.Decrypt("not-base64", "also-bad"); err == nil {
		t.Fatalf("expected base64 decode error")
	}
}
