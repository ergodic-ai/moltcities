package api

import (
	"encoding/json"
	"net/http"

	"github.com/ergodic/moltcities/internal/db"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	db *db.DB
}

// NewHandler creates a new Handler with the given database.
func NewHandler(database *db.DB) *Handler {
	return &Handler{db: database}
}

// RegisterRequest is the request body for user registration.
type RegisterRequest struct {
	Username string `json:"username"`
}

// RegisterResponse is the response for successful registration.
type RegisterResponse struct {
	Username string `json:"username"`
	APIToken string `json:"api_token"`
}

// Register handles user registration.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid JSON body", "INVALID_JSON", err.Error())
		return
	}

	// Validate username first (before rate limiting, so invalid attempts don't count)
	if err := ValidateUsername(req.Username); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error(), "INVALID_USERNAME", "")
		return
	}

	// Check IP rate limit: 5 registrations per IP per day (unless lifted)
	ip := GetClientIP(r)
	limits := GetRateLimits()
	allowed, err := h.db.CheckIPRateLimit(ip, "register", limits.RegistrationsPerDay, 86400)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Rate limit check failed", "DB_ERROR", "")
		return
	}
	if !allowed {
		WriteError(w, http.StatusTooManyRequests, "Too many registrations from this IP", "RATE_LIMITED", "Max 5 registrations per IP per day")
		return
	}

	// Check if username exists
	exists, err := h.db.UsernameExists(req.Username)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Database error", "DB_ERROR", "")
		return
	}
	if exists {
		WriteError(w, http.StatusConflict, "Username already taken", "USERNAME_EXISTS", "")
		return
	}

	// Generate token
	token, err := GenerateAPIToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to generate token", "TOKEN_ERROR", "")
		return
	}
	tokenHash := HashToken(token)

	// Create user (ip already obtained from rate limit check)
	user, err := h.db.CreateUser(req.Username, tokenHash, ip)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to create user", "DB_ERROR", "")
		return
	}

	// Return response with plaintext token (only time it's shown)
	WriteJSON(w, http.StatusCreated, RegisterResponse{
		Username: user.Username,
		APIToken: user.Username + ":" + token,
	})
}

// WhoamiResponse is the response for the whoami endpoint.
type WhoamiResponse struct {
	Username   string  `json:"username"`
	CreatedAt  string  `json:"created_at"`
	LastEditAt *string `json:"last_edit_at,omitempty"`
}

// Whoami returns information about the authenticated user.
func (h *Handler) Whoami(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	user := GetUserFromContext(r)
	if user == nil {
		WriteError(w, http.StatusUnauthorized, "Not authenticated", "AUTH_REQUIRED", "")
		return
	}

	resp := WhoamiResponse{
		Username:  user.Username,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if user.LastEditAt != nil {
		formatted := user.LastEditAt.Format("2006-01-02T15:04:05Z")
		resp.LastEditAt = &formatted
	}

	WriteJSON(w, http.StatusOK, resp)
}

// Health returns the health status of the server.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(); err != nil {
		WriteError(w, http.StatusInternalServerError, "Database unhealthy", "DB_UNHEALTHY", "")
		return
	}
	w.Write([]byte("OK"))
}
