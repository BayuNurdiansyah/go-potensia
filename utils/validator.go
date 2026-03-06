package utils

import (
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// IsValidEmail returns true if the email format is valid.
func IsValidEmail(email string) bool {
	return emailRegex.MatchString(strings.TrimSpace(email))
}

// IsValidPassword returns true if the password meets minimum requirements:
// - At least 8 characters
// - At least one letter and one digit
func IsValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	return hasLetter && hasDigit
}