package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ergodic/moltcities/internal/canvas"
	"github.com/ergodic/moltcities/internal/models"
)

// imageCache stores the cached canvas image.
var (
	imageCache     []byte
	imageCacheMu   sync.RWMutex
	imageCacheTime time.Time
	imageCacheTTL  = 10 * time.Second // Cache for 10 seconds
)

// GetCanvasImage returns the full canvas as a PNG image.
func (h *Handler) GetCanvasImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	// Check cache
	imageCacheMu.RLock()
	if imageCache != nil && time.Since(imageCacheTime) < imageCacheTTL {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Write(imageCache)
		imageCacheMu.RUnlock()
		return
	}
	imageCacheMu.RUnlock()

	// Generate new image
	pixels, err := h.db.GetAllPixels()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to get pixels", "DB_ERROR", "")
		return
	}

	var buf bytes.Buffer
	if err := canvas.Render(pixels, &buf); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to render image", "RENDER_ERROR", "")
		return
	}

	// Update cache
	imageCacheMu.Lock()
	imageCache = buf.Bytes()
	imageCacheTime = time.Now()
	imageCacheMu.Unlock()

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Write(buf.Bytes())
}

// GetCanvasRegion returns pixel data for a region (max 128x128).
func (h *Handler) GetCanvasRegion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	// Parse query params
	x, err := strconv.Atoi(r.URL.Query().Get("x"))
	if err != nil {
		x = 0
	}
	y, err := strconv.Atoi(r.URL.Query().Get("y"))
	if err != nil {
		y = 0
	}
	width, err := strconv.Atoi(r.URL.Query().Get("width"))
	if err != nil || width == 0 {
		width = models.MaxRegionSize
	}
	height, err := strconv.Atoi(r.URL.Query().Get("height"))
	if err != nil || height == 0 {
		height = models.MaxRegionSize
	}

	// Validate region
	if err := canvas.ValidateRegion(x, y, width, height); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error(), "INVALID_REGION", "")
		return
	}

	// Get pixels
	pixels, err := h.db.GetRegion(x, y, width, height)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to get region", "DB_ERROR", "")
		return
	}

	resp := models.RegionResponse{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
		Pixels: pixels,
	}

	w.Header().Set("Cache-Control", "public, max-age=60")
	WriteJSON(w, http.StatusOK, resp)
}

// GetPixel returns information about a single pixel.
func (h *Handler) GetPixel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	// Parse query params
	xStr := r.URL.Query().Get("x")
	yStr := r.URL.Query().Get("y")

	if xStr == "" || yStr == "" {
		WriteError(w, http.StatusBadRequest, "Missing x or y parameter", "MISSING_PARAM", "")
		return
	}

	x, err := strconv.Atoi(xStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid x coordinate", "INVALID_PARAM", "")
		return
	}
	y, err := strconv.Atoi(yStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid y coordinate", "INVALID_PARAM", "")
		return
	}

	// Validate coordinates
	if err := canvas.ValidateCoordinate(x); err != nil {
		WriteError(w, http.StatusBadRequest, "x: "+err.Error(), "INVALID_COORD", "")
		return
	}
	if err := canvas.ValidateCoordinate(y); err != nil {
		WriteError(w, http.StatusBadRequest, "y: "+err.Error(), "INVALID_COORD", "")
		return
	}

	pixel, err := h.db.GetPixel(x, y)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to get pixel", "DB_ERROR", "")
		return
	}

	WriteJSON(w, http.StatusOK, pixel)
}

// EditPixelRequest is the request body for editing a pixel.
type EditPixelRequest struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Color string `json:"color"`
}

// EditPixelResponse is the response for editing a pixel.
type EditPixelResponse struct {
	Success    bool    `json:"success"`
	NextEditAt *string `json:"next_edit_at,omitempty"`
}

// EditPixel updates a single pixel (requires auth).
func (h *Handler) EditPixel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	user := GetUserFromContext(r)
	if user == nil {
		WriteError(w, http.StatusUnauthorized, "Not authenticated", "AUTH_REQUIRED", "")
		return
	}

	// Check if user can edit
	canEdit, nextEdit, err := h.db.CanUserEditNow(user.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to check edit status", "DB_ERROR", "")
		return
	}
	if !canEdit {
		formatted := nextEdit.Format(time.RFC3339)
		WriteError(w, http.StatusTooManyRequests, "You can only edit once per day", "RATE_LIMITED", "Next edit available at "+formatted)
		return
	}

	// Parse request
	var req EditPixelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid JSON body", "INVALID_JSON", err.Error())
		return
	}

	// Validate coordinates
	if err := canvas.ValidateCoordinate(req.X); err != nil {
		WriteError(w, http.StatusBadRequest, "x: "+err.Error(), "INVALID_COORD", "")
		return
	}
	if err := canvas.ValidateCoordinate(req.Y); err != nil {
		WriteError(w, http.StatusBadRequest, "y: "+err.Error(), "INVALID_COORD", "")
		return
	}

	// Validate color
	if err := canvas.ValidateColor(req.Color); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid color format. Use #RRGGBB", "INVALID_COLOR", "")
		return
	}

	// Set pixel
	if err := h.db.SetPixel(req.X, req.Y, req.Color, user.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to edit pixel", "DB_ERROR", "")
		return
	}

	// Invalidate image cache
	imageCacheMu.Lock()
	imageCache = nil
	imageCacheMu.Unlock()

	// Calculate next edit time
	nextEditTime := time.Now().Add(24 * time.Hour).Format(time.RFC3339)

	WriteJSON(w, http.StatusOK, EditPixelResponse{
		Success:    true,
		NextEditAt: &nextEditTime,
	})
}

// GetPixelHistory returns the edit history for a pixel.
func (h *Handler) GetPixelHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	xStr := r.URL.Query().Get("x")
	yStr := r.URL.Query().Get("y")
	limitStr := r.URL.Query().Get("limit")

	if xStr == "" || yStr == "" {
		WriteError(w, http.StatusBadRequest, "Missing x or y parameter", "MISSING_PARAM", "")
		return
	}

	x, _ := strconv.Atoi(xStr)
	y, _ := strconv.Atoi(yStr)
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if err := canvas.ValidateCoordinate(x); err != nil {
		WriteError(w, http.StatusBadRequest, "x: "+err.Error(), "INVALID_COORD", "")
		return
	}
	if err := canvas.ValidateCoordinate(y); err != nil {
		WriteError(w, http.StatusBadRequest, "y: "+err.Error(), "INVALID_COORD", "")
		return
	}

	history, err := h.db.GetPixelHistory(x, y, limit)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to get history", "DB_ERROR", "")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"x":       x,
		"y":       y,
		"history": history,
	})
}

// GetStats returns canvas statistics.
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	stats, err := h.db.GetStats()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to get stats", "DB_ERROR", "")
		return
	}

	WriteJSON(w, http.StatusOK, stats)
}
