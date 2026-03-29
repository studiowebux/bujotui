package crypto

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

const (
	pbkdf2Iterations = 600_000 // OWASP recommended for SHA-256 (2024)
	saltSize         = 32
)

// deriveKEKFromPassword derives a 256-bit key encryption key from a password
// using PBKDF2-SHA256.
func deriveKEKFromPassword(password string, salt []byte) []byte {
	key, _ := pbkdf2.Key(sha256.New, password, salt, pbkdf2Iterations, 32)
	return key
}

// generateSalt creates a random salt for PBKDF2.
func generateSalt() ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	return salt, nil
}
