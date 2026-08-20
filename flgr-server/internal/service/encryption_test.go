package service

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func validTestEncryptionKey() []byte {
	return make([]byte, encryptionKeySize)
}

func TestDecodeEncryptionKey_Success(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString(validTestEncryptionKey())
	key, err := DecodeEncryptionKey(encoded)
	if err != nil {
		t.Fatalf("DecodeEncryptionKey() returned unexpected error: %v", err)
	}
	if len(key) != encryptionKeySize {
		t.Errorf("len(key) = %d, want %d", len(key), encryptionKeySize)
	}
}

func TestDecodeEncryptionKey_InvalidBase64(t *testing.T) {
	_, err := DecodeEncryptionKey("not-valid-base64!!!")
	if err == nil {
		t.Fatal("DecodeEncryptionKey() expected an error for invalid base64, got nil")
	}
}

func TestDecodeEncryptionKey_WrongLength(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("too-short"))
	_, err := DecodeEncryptionKey(encoded)
	if err == nil {
		t.Fatal("DecodeEncryptionKey() expected an error for a non-32-byte key, got nil")
	}
	if !strings.Contains(err.Error(), "32 bytes") {
		t.Errorf("error = %q, want it to mention the required 32-byte length", err.Error())
	}
}

func TestEncryptSecret_DecryptSecret_RoundTrip(t *testing.T) {
	key := validTestEncryptionKey()
	ciphertext, err := encryptSecret(key, "hmac-signing-secret")
	if err != nil {
		t.Fatalf("encryptSecret() returned unexpected error: %v", err)
	}
	if string(ciphertext) == "hmac-signing-secret" {
		t.Error("ciphertext must not equal the plaintext")
	}

	plaintext, err := decryptSecret(key, ciphertext)
	if err != nil {
		t.Fatalf("decryptSecret() returned unexpected error: %v", err)
	}
	if plaintext != "hmac-signing-secret" {
		t.Errorf("plaintext = %q, want %q", plaintext, "hmac-signing-secret")
	}
}

func TestEncryptSecret_DifferentNoncePerCall(t *testing.T) {
	key := validTestEncryptionKey()
	c1, err := encryptSecret(key, "same-plaintext")
	if err != nil {
		t.Fatalf("encryptSecret() (1) returned unexpected error: %v", err)
	}
	c2, err := encryptSecret(key, "same-plaintext")
	if err != nil {
		t.Fatalf("encryptSecret() (2) returned unexpected error: %v", err)
	}
	if string(c1) == string(c2) {
		t.Error("two encryptions of the same plaintext produced identical ciphertext, want distinct nonces")
	}
}

func TestEncryptSecret_InvalidKeySize(t *testing.T) {
	_, err := encryptSecret([]byte("too-short"), "plaintext")
	if err == nil {
		t.Fatal("encryptSecret() expected an error for an invalid AES key size, got nil")
	}
}

func TestEncryptSecret_RandReadError(t *testing.T) {
	original := randRead
	randRead = func(b []byte) (int, error) { return 0, errors.New("entropy source unavailable") }
	t.Cleanup(func() { randRead = original })

	_, err := encryptSecret(validTestEncryptionKey(), "plaintext")
	if err == nil {
		t.Fatal("encryptSecret() expected an error when randRead fails, got nil")
	}
}

func TestDecryptSecret_InvalidKeySize(t *testing.T) {
	_, err := decryptSecret([]byte("too-short"), []byte("irrelevant-ciphertext"))
	if err == nil {
		t.Fatal("decryptSecret() expected an error for an invalid AES key size, got nil")
	}
}

func TestDecryptSecret_CiphertextShorterThanNonce(t *testing.T) {
	_, err := decryptSecret(validTestEncryptionKey(), []byte("short"))
	if err == nil {
		t.Fatal("decryptSecret() expected an error for a too-short ciphertext, got nil")
	}
}

func TestDecryptSecret_TamperedCiphertext(t *testing.T) {
	key := validTestEncryptionKey()
	ciphertext, err := encryptSecret(key, "hmac-signing-secret")
	if err != nil {
		t.Fatalf("encryptSecret() returned unexpected error: %v", err)
	}
	tampered := append([]byte{}, ciphertext...)
	tampered[len(tampered)-1] ^= 0xFF

	_, err = decryptSecret(key, tampered)
	if err == nil {
		t.Fatal("decryptSecret() expected an error for tampered ciphertext (GCM auth failure), got nil")
	}
}

func TestDecryptSecret_WrongKey(t *testing.T) {
	key := validTestEncryptionKey()
	ciphertext, err := encryptSecret(key, "hmac-signing-secret")
	if err != nil {
		t.Fatalf("encryptSecret() returned unexpected error: %v", err)
	}

	wrongKey := make([]byte, encryptionKeySize)
	wrongKey[0] = 1
	_, err = decryptSecret(wrongKey, ciphertext)
	if err == nil {
		t.Fatal("decryptSecret() expected an error when decrypting with the wrong key, got nil")
	}
}
