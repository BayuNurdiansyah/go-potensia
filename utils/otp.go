package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateOTP generates a cryptographically secure 6-digit OTP.
func GenerateOTP() string {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Extremely unlikely, but panic-safe fallback
		panic("failed to generate OTP: " + err.Error())
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// GenerateSecureToken generates a cryptographically secure hex token of given byte length.
// e.g., length=32 produces a 64-character hex string.
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return fmt.Sprintf("%x", bytes), nil
}