package service

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
)

// encryptionKeySize is the required decoded length of FLGR_ENCRYPTION_KEY —
// 32 bytes for AES-256, per
// docs/architecture/adr/0010-application-configuration-and-secrets.md.
const encryptionKeySize = 32

// DecodeEncryptionKey base64-decodes cfg.EncryptionKey into the raw
// AES-256 key encryptSecret/decryptSecret need, failing if it isn't valid
// base64 or doesn't decode to exactly 32 bytes. Callers (internal/api's
// router wiring) are expected to treat a non-nil error as fatal at startup,
// per ADR-0010's fail-fast validation rule — this function itself doesn't
// exit the process, so it stays testable.
func DecodeEncryptionKey(encoded string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding FLGR_ENCRYPTION_KEY: %w", err)
	}
	if len(key) != encryptionKeySize {
		return nil, fmt.Errorf("FLGR_ENCRYPTION_KEY must decode to %d bytes (AES-256), got %d", encryptionKeySize, len(key))
	}
	return key, nil
}

// encryptSecret AES-256-GCM encrypts plaintext with key, returning the
// random nonce prepended to the ciphertext (gcm.Seal's "append to dst"
// idiom) so decryptSecret can recover it without storing it separately.
// Used to protect the HTTP notification channel signing secret at rest,
// per docs/architecture/adr/0009-kafka-and-webhook-notification-delivery.md
// and ADR-0010 — unlike a password or Service Key secret, this value must
// be retrievable in plaintext later, which is why it's reversibly
// encrypted rather than one-way hashed (see hashPassword/hashToken).
func encryptSecret(key []byte, plaintext string) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := randRead(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	return gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// decryptSecret reverses encryptSecret.
func decryptSecret(key, ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext shorter than nonce size")
	}
	nonce, sealed := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("decrypting secret: %w", err)
	}
	return string(plaintext), nil
}
