package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// ChannelNameRegex validates channel names: alphanumeric + hyphens, 3-32 chars
	ChannelNameRegex = regexp.MustCompile(`^[a-z0-9-]{3,32}$`)
)

// ValidateChannelName checks if a channel name is valid.
func ValidateChannelName(name string) error {
	if len(name) < 3 {
		return &ValidationError{Field: "name", Message: "must be at least 3 characters"}
	}
	if len(name) > 32 {
		return &ValidationError{Field: "name", Message: "must be at most 32 characters"}
	}
	if !ChannelNameRegex.MatchString(name) {
		return &ValidationError{Field: "name", Message: "must contain only lowercase letters, numbers, and hyphens"}
	}
	return nil
}

// CreateChannelRequest is the request body for creating a channel.
type CreateChannelRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ListChannels returns all channels.
func (h *Handler) ListChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	channels, err := h.db.ListChannels()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to list channels", "DB_ERROR", "")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"channels": channels,
	})
}

// GetChannel returns information about a specific channel.
func (h *Handler) GetChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	// Extract channel name from path: /channels/{name}
	name := strings.TrimPrefix(r.URL.Path, "/channels/")
	if name == "" || strings.Contains(name, "/") {
		WriteError(w, http.StatusBadRequest, "Invalid channel name", "INVALID_PARAM", "")
		return
	}

	channel, err := h.db.GetChannel(name)
	if err != nil {
		WriteError(w, http.StatusNotFound, "Channel not found", "NOT_FOUND", "")
		return
	}

	WriteJSON(w, http.StatusOK, channel)
}

// CreateChannel creates a new channel.
func (h *Handler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	user := GetUserFromContext(r)
	if user == nil {
		WriteError(w, http.StatusUnauthorized, "Not authenticated", "AUTH_REQUIRED", "")
		return
	}

	// Check rate limit: 3 channels per user per day
	count, err := h.db.CountUserChannelsToday(user.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to check rate limit", "DB_ERROR", "")
		return
	}
	if count >= 3 {
		WriteError(w, http.StatusTooManyRequests, "You can only create 3 channels per day", "RATE_LIMITED", "")
		return
	}

	var req CreateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid JSON body", "INVALID_JSON", err.Error())
		return
	}

	// Normalize name to lowercase
	req.Name = strings.ToLower(req.Name)

	// Validate name
	if err := ValidateChannelName(req.Name); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error(), "INVALID_NAME", "")
		return
	}

	// Validate description length
	if len(req.Description) > 256 {
		WriteError(w, http.StatusBadRequest, "Description must be at most 256 characters", "INVALID_DESCRIPTION", "")
		return
	}

	// Check if channel exists
	exists, err := h.db.ChannelExists(req.Name)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Database error", "DB_ERROR", "")
		return
	}
	if exists {
		WriteError(w, http.StatusConflict, "Channel already exists", "CHANNEL_EXISTS", "")
		return
	}

	// Create channel
	channel, err := h.db.CreateChannel(req.Name, req.Description, user.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to create channel", "DB_ERROR", "")
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"name":    channel.Name,
		"created": true,
	})
}

// PostMessageRequest is the request body for posting a message.
type PostMessageRequest struct {
	Content string `json:"content"`
}

// PostMessage posts a message to a channel.
func (h *Handler) PostMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	user := GetUserFromContext(r)
	if user == nil {
		WriteError(w, http.StatusUnauthorized, "Not authenticated", "AUTH_REQUIRED", "")
		return
	}

	// Extract channel name from path: /channels/{name}/messages
	path := strings.TrimPrefix(r.URL.Path, "/channels/")
	path = strings.TrimSuffix(path, "/messages")
	channelName := path

	if channelName == "" {
		WriteError(w, http.StatusBadRequest, "Invalid channel name", "INVALID_PARAM", "")
		return
	}

	// Check rate limit: 10 messages per user per hour
	count, err := h.db.CountUserMessagesLastHour(user.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to check rate limit", "DB_ERROR", "")
		return
	}
	if count >= 10 {
		WriteError(w, http.StatusTooManyRequests, "You can only post 10 messages per hour", "RATE_LIMITED", "")
		return
	}

	// Get channel
	channel, err := h.db.GetChannel(channelName)
	if err != nil {
		WriteError(w, http.StatusNotFound, "Channel not found", "NOT_FOUND", "")
		return
	}

	var req PostMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid JSON body", "INVALID_JSON", err.Error())
		return
	}

	// Validate content
	if len(req.Content) < 1 {
		WriteError(w, http.StatusBadRequest, "Message content cannot be empty", "INVALID_CONTENT", "")
		return
	}
	if len(req.Content) > 1000 {
		WriteError(w, http.StatusBadRequest, "Message content must be at most 1000 characters", "INVALID_CONTENT", "")
		return
	}

	// Create message
	message, err := h.db.CreateMessage(channel.ID, user.ID, req.Content)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to create message", "DB_ERROR", "")
		return
	}

	WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         message.ID,
		"created_at": message.CreatedAt.Format(time.RFC3339),
	})
}

// GetMessages retrieves messages from a channel.
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	// Extract channel name from path: /channels/{name}/messages
	path := strings.TrimPrefix(r.URL.Path, "/channels/")
	path = strings.TrimSuffix(path, "/messages")
	channelName := path

	if channelName == "" {
		WriteError(w, http.StatusBadRequest, "Invalid channel name", "INVALID_PARAM", "")
		return
	}

	// Get channel
	channel, err := h.db.GetChannel(channelName)
	if err != nil {
		WriteError(w, http.StatusNotFound, "Channel not found", "NOT_FOUND", "")
		return
	}

	// Parse query params
	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	var since *time.Time
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = &t
		}
	}

	// Get messages
	messages, err := h.db.GetChannelMessages(channel.ID, limit, since)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to get messages", "DB_ERROR", "")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"channel":  channelName,
		"messages": messages,
	})
}
