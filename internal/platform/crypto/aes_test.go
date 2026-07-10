package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("refresh-token-secret")
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Fatalf("round trip mismatch: %q", decrypted)
	}
}

func TestDecryptInvalidKey(t *testing.T) {
	key := make([]byte, 32)
	ciphertext, err := Encrypt([]byte("x"), key)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Decrypt(ciphertext, []byte("short"))
	if err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestEncryptProducesBase64(t *testing.T) {
	key := make([]byte, 32)
	ciphertext, err := Encrypt([]byte("test"), key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := base64.StdEncoding.DecodeString(ciphertext); err != nil {
		t.Fatalf("not valid base64: %v", err)
	}
}
