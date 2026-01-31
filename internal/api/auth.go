package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var (
	// UsernameRegex validates usernames: alphanumeric + underscore, 3-32 chars
	UsernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)
)

// GenerateAPIToken creates a random 32-character hex token.
func GenerateAPIToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken creates a SHA256 hash of the token.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// ValidateUsername checks if a username is valid.
func ValidateUsername(username string) error {
	if len(username) < 3 {
		return &ValidationError{Field: "username", Message: "must be at least 3 characters"}
	}
	if len(username) > 32 {
		return &ValidationError{Field: "username", Message: "must be at most 32 characters"}
	}
	if !UsernameRegex.MatchString(username) {
		return &ValidationError{Field: "username", Message: "must contain only letters, numbers, and underscores"}
	}
	// Reserved usernames
	reserved := []string{"system", "admin", "moltcities", "api", "www", "mail"}
	lower := strings.ToLower(username)
	for _, r := range reserved {
		if lower == r {
			return &ValidationError{Field: "username", Message: "this username is reserved"}
		}
	}
	return nil
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
