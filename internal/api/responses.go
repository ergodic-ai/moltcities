package api

import (
	"encoding/json"
	"net/http"

	"github.com/ergodic/moltcities/internal/models"
)

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteError writes a JSON error response.
func WriteError(w http.ResponseWriter, status int, message, code, details string) {
	resp := models.ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}
	WriteJSON(w, status, resp)
}
