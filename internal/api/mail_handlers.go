package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

const (
	// MaxMailSize is the maximum message size (10KB)
	MaxMailSize = 10 * 1024
)

// SendMailRequest is the request body for sending mail.
type SendMailRequest struct {
	To   string `json:"to"`
	Body string `json:"body"`
}

// SendMail handles POST /mail
func (h *Handler) SendMail(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	if user == nil {
		WriteError(w, http.StatusUnauthorized, "Not authenticated", "AUTH_REQUIRED", "")
		return
	}

	// Parse request
	var req SendMailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid JSON", "INVALID_JSON", "")
		return
	}

	// Validate recipient
	req.To = strings.TrimSpace(strings.ToLower(req.To))
	if req.To == "" {
		WriteError(w, http.StatusBadRequest, "Recipient is required", "MISSING_TO", "")
		return
	}

	// Can't send to yourself
	if req.To == user.Username {
		WriteError(w, http.StatusBadRequest, "Cannot send mail to yourself", "SELF_MAIL", "")
		return
	}

	// Validate body
	if len(req.Body) == 0 {
		WriteError(w, http.StatusBadRequest, "Message body is required", "MISSING_BODY", "")
		return
	}

	if len(req.Body) > MaxMailSize {
		WriteError(w, http.StatusRequestEntityTooLarge, "Message too large. Maximum size is 10KB.", "TOO_LARGE", "")
		return
	}

	// Check rate limit
	count, err := h.db.CountMailSentToday(user.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to check rate limit", "DB_ERROR", "")
		return
	}
	limits := GetRateLimits()
	if count >= limits.MailSendsPerDay {
		WriteError(w, http.StatusTooManyRequests, "You can only send 20 messages per day", "RATE_LIMITED", "")
		return
	}

	// Send mail
	mail, err := h.db.SendMail(user.ID, req.To, req.Body)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteError(w, http.StatusNotFound, "User not found", "USER_NOT_FOUND", "")
			return
		}
		WriteError(w, http.StatusInternalServerError, "Failed to send mail", "DB_ERROR", "")
		return
	}

	// Record send for rate limiting
	h.db.RecordMailSend(user.ID)

	WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         mail.ID,
		"to":         mail.ToUser,
		"created_at": mail.CreatedAt,
	})
}

// GetInbox handles GET /mail
func (h *Handler) GetInbox(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	if user == nil {
		WriteError(w, http.StatusUnauthorized, "Not authenticated", "AUTH_REQUIRED", "")
		return
	}

	// Parse pagination
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	messages, unreadCount, totalCount, err := h.db.GetInbox(user.ID, limit, offset)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to get inbox", "DB_ERROR", "")
		return
	}

	// Convert to response format
	msgList := make([]map[string]interface{}, 0, len(messages))
	for _, m := range messages {
		msgList = append(msgList, map[string]interface{}{
			"id":         m.ID,
			"from":       m.FromUser,
			"body":       m.Body,
			"read":       m.Read,
			"created_at": m.CreatedAt,
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"messages":     msgList,
		"unread_count": unreadCount,
		"total_count":  totalCount,
	})
}

// GetMessage handles GET /mail/{id}
func (h *Handler) GetMessage(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	if user == nil {
		WriteError(w, http.StatusUnauthorized, "Not authenticated", "AUTH_REQUIRED", "")
		return
	}

	// Extract message ID from path
	path := strings.TrimPrefix(r.URL.Path, "/mail/")
	messageID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid message ID", "INVALID_ID", "")
		return
	}

	mail, err := h.db.GetMessage(user.ID, messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteError(w, http.StatusNotFound, "Message not found", "NOT_FOUND", "")
			return
		}
		WriteError(w, http.StatusInternalServerError, "Failed to get message", "DB_ERROR", "")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"id":         mail.ID,
		"from":       mail.FromUser,
		"body":       mail.Body,
		"read_at":    mail.ReadAt,
		"created_at": mail.CreatedAt,
	})
}

// DeleteMail handles DELETE /mail/{id}
func (h *Handler) DeleteMail(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	if user == nil {
		WriteError(w, http.StatusUnauthorized, "Not authenticated", "AUTH_REQUIRED", "")
		return
	}

	// Extract message ID from path
	path := strings.TrimPrefix(r.URL.Path, "/mail/")
	messageID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid message ID", "INVALID_ID", "")
		return
	}

	err = h.db.DeleteMessage(user.ID, messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			WriteError(w, http.StatusNotFound, "Message not found", "NOT_FOUND", "")
			return
		}
		WriteError(w, http.StatusInternalServerError, "Failed to delete message", "DB_ERROR", "")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// ListUsers handles GET /users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse pagination
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	users, totalCount, err := h.db.ListUsers(limit, offset)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to list users", "DB_ERROR", "")
		return
	}

	// Convert to response format
	userList := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		userList = append(userList, map[string]interface{}{
			"username":   u.Username,
			"created_at": u.CreatedAt,
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"users":       userList,
		"total_count": totalCount,
	})
}
