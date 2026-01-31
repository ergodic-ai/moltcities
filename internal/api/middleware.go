package api

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/ergodic/moltcities/internal/db"
	"github.com/ergodic/moltcities/internal/models"
)

// ContextKey is a custom type for context keys to avoid collisions.
type ContextKey string

const (
	// UserContextKey is the key for storing the authenticated user in context.
	UserContextKey ContextKey = "user"
)

// AuthMiddleware validates the API token and adds the user to the request context.
func AuthMiddleware(database *db.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				WriteError(w, http.StatusUnauthorized, "Missing authentication token", "AUTH_REQUIRED", "")
				return
			}

			// Parse token format: username:token
			parts := strings.SplitN(token, ":", 2)
			if len(parts) != 2 {
				WriteError(w, http.StatusUnauthorized, "Invalid token format", "INVALID_TOKEN", "Expected format: username:token")
				return
			}

			username, rawToken := parts[0], parts[1]
			tokenHash := HashToken(rawToken)

			user, err := database.ValidateUserToken(username, tokenHash)
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "Invalid credentials", "INVALID_CREDENTIALS", "")
				return
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext retrieves the authenticated user from the request context.
func GetUserFromContext(r *http.Request) *models.User {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// extractToken gets the API token from the request.
// Checks Authorization header (Bearer token) or X-API-Token header.
func extractToken(r *http.Request) string {
	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
	}

	// Check X-API-Token header
	token := r.Header.Get("X-API-Token")
	if token != "" {
		return token
	}

	return ""
}

// GetClientIP extracts the client IP address from the request.
// Handles proxies by checking X-Forwarded-For and X-Real-IP headers.
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For (may contain multiple IPs)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fallback to direct connection
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// CORSMiddleware adds CORS headers to responses.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Token")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
