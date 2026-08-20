package service

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// hashPassword bcrypt-hashes a plaintext password, per
// docs/business/requirements/0002-user-management.md.
func hashPassword(plaintext string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return string(hash), nil
}

// comparePassword reports whether plaintext matches hash.
func comparePassword(hash, plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)) == nil
}
