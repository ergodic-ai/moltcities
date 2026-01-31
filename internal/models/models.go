// Package models contains shared data structures for MoltCities.
package models

import "time"

// CanvasSize is the fixed size of the canvas (1024x1024).
const CanvasSize = 1024

// MaxRegionSize is the maximum size for region queries (128x128).
const MaxRegionSize = 128

// User represents a registered bot/user.
type User struct {
	ID             int64      `json:"id"`
	Username       string     `json:"username"`
	APITokenHash   string     `json:"-"` // Never expose in JSON
	LastEditAt     *time.Time `json:"last_edit_at,omitempty"`
	RegistrationIP string     `json:"-"` // Never expose in JSON
	CreatedAt      time.Time  `json:"created_at"`
}

// Pixel represents a single pixel on the canvas.
type Pixel struct {
	X        int        `json:"x"`
	Y        int        `json:"y"`
	Color    string     `json:"color"`
	EditedBy *string    `json:"edited_by,omitempty"` // Username
	EditedAt *time.Time `json:"edited_at,omitempty"`
}

// Edit represents a historical edit to the canvas.
type Edit struct {
	ID        int64     `json:"id"`
	X         int       `json:"x"`
	Y         int       `json:"y"`
	Color     string    `json:"color"`
	UserID    int64     `json:"-"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// Channel represents a chat channel for coordination.
type Channel struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	CreatedBy    int64     `json:"-"`
	CreatedByName string   `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	MessageCount int       `json:"message_count,omitempty"`
}

// Message represents a chat message in a channel.
type Message struct {
	ID        int64     `json:"id"`
	ChannelID int64     `json:"-"`
	UserID    int64     `json:"-"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// RegionResponse is the response for region queries.
type RegionResponse struct {
	X      int        `json:"x"`
	Y      int        `json:"y"`
	Width  int        `json:"width"`
	Height int        `json:"height"`
	Pixels [][]string `json:"pixels"` // [row][col] = "#RRGGBB"
}

// Stats holds canvas and user statistics.
type Stats struct {
	TotalEdits    int `json:"total_edits"`
	UniquePixels  int `json:"unique_pixels"`
	TotalUsers    int `json:"total_users"`
	TotalChannels int `json:"total_channels"`
	TotalMessages int `json:"total_messages"`
}

// ErrorResponse is the standard error format.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}
