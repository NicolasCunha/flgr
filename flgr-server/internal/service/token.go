package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// randRead is a seam over rand.Read so its (practically unreachable) error
// path can still be exercised in a test, per the exception clause in
// docs/architecture/adr/0004-testing-and-coverage-standards.md — mirrors
// the sqlOpen seam in internal/database/database.go.
var randRead = rand.Read

// tokenByteLength is the entropy of a session token before hex-encoding.
const tokenByteLength = 32

// generateToken returns a high-entropy, random, hex-encoded session token,
// per docs/architecture/adr/0006-authentication-and-session-strategy.md.
func generateToken() (string, error) {
	buf := make([]byte, tokenByteLength)
	if _, err := randRead(buf); err != nil {
		return "", fmt.Errorf("generating session token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// hashToken hashes an opaque session token for storage. SHA-256 is
// sufficient here (unlike bcrypt for passwords): the input is already a
// high-entropy random value, not a human-chosen secret, mirroring the
// reasoning ADR-0006 applies to Service Key secrets.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
