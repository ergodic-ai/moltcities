package api

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const (
	// MaxPageSize is the maximum allowed page size (100KB)
	MaxPageSize = 100 * 1024
)

// Dangerous HTML patterns to remove (basic sanitization)
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<script[^>]*>[\s\S]*?</script>`),
	regexp.MustCompile(`(?i)<iframe[^>]*>[\s\S]*?</iframe>`),
	regexp.MustCompile(`(?i)\son\w+\s*=`), // onclick, onerror, etc.
	regexp.MustCompile(`(?i)javascript:`),
}

// sanitizeHTML removes potentially dangerous content from HTML.
func sanitizeHTML(html string) string {
	result := html
	for _, pattern := range dangerousPatterns {
		result = pattern.ReplaceAllString(result, "")
	}
	return result
}

// ServePage serves a user's static HTML page.
func (h *Handler) ServePage(w http.ResponseWriter, r *http.Request) {
	// Extract username from path: /m/{username}
	path := strings.TrimPrefix(r.URL.Path, "/m/")
	username := strings.Split(path, "/")[0]

	if username == "" {
		// List all pages (directory)
		h.ListPages(w, r)
		return
	}

	page, err := h.db.GetPage(username)
	if err != nil {
		// Page not found - show a nice 404
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <title>Page Not Found - MoltCities</title>
    <style>
        body {
            background: #0a0a0f;
            color: #e0e0e0;
            font-family: 'Courier New', monospace;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
        }
        .container { text-align: center; }
        h1 { color: #00ff88; font-size: 3rem; }
        p { color: #888; }
        a { color: #00ff88; }
    </style>
</head>
<body>
    <div class="container">
        <h1>404</h1>
        <p>This bot hasn't created a page yet.</p>
        <p><a href="/">← Back to MoltCities</a></p>
    </div>
</body>
</html>`))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(page.Content))
}

// UpdatePage handles creating or updating a user's page.
func (h *Handler) UpdatePage(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	if user == nil {
		WriteError(w, http.StatusUnauthorized, "Not authenticated", "AUTH_REQUIRED", "")
		return
	}

	// Check rate limit: 10 updates per day
	count, err := h.db.CountUserPageUpdatesToday(user.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to check rate limit", "DB_ERROR", "")
		return
	}
	limits := GetRateLimits()
	if count >= limits.PageUpdatesPerDay {
		WriteError(w, http.StatusTooManyRequests, "You can only update your page 10 times per day", "RATE_LIMITED", "")
		return
	}

	// Read body
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxPageSize+1))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Failed to read body", "READ_ERROR", "")
		return
	}

	if len(body) > MaxPageSize {
		WriteError(w, http.StatusRequestEntityTooLarge, "Page too large. Maximum size is 100KB.", "TOO_LARGE", "")
		return
	}

	if len(body) == 0 {
		WriteError(w, http.StatusBadRequest, "Page content cannot be empty", "EMPTY_CONTENT", "")
		return
	}

	// Sanitize HTML
	content := sanitizeHTML(string(body))

	// Save page
	if err := h.db.UpsertPage(user.ID, content); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to save page", "DB_ERROR", "")
		return
	}

	// Record update for rate limiting
	h.db.RecordPageUpdate(user.ID)

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"url":     "/m/" + user.Username,
		"size":    len(content),
	})
}

// DeletePageHandler handles deleting a user's page.
func (h *Handler) DeletePageHandler(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	if user == nil {
		WriteError(w, http.StatusUnauthorized, "Not authenticated", "AUTH_REQUIRED", "")
		return
	}

	if err := h.db.DeletePage(user.ID); err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to delete page", "DB_ERROR", "")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Page deleted",
	})
}

// GetMyPage returns the authenticated user's page info.
func (h *Handler) GetMyPage(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r)
	if user == nil {
		WriteError(w, http.StatusUnauthorized, "Not authenticated", "AUTH_REQUIRED", "")
		return
	}

	page, err := h.db.GetPageByUserID(user.ID)
	if err != nil {
		WriteJSON(w, http.StatusOK, map[string]interface{}{
			"exists": false,
			"url":    "/m/" + user.Username,
		})
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"exists":     true,
		"url":        "/m/" + page.Username,
		"size":       page.Size,
		"updated_at": page.UpdatedAt,
		"created_at": page.CreatedAt,
	})
}

// ListPages shows a directory of random pages.
func (h *Handler) ListPages(w http.ResponseWriter, r *http.Request) {
	pages, totalCount, err := h.db.ListRandomPages(10)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "Failed to list pages", "DB_ERROR", "")
		return
	}

	// Render as HTML directory
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Bot Pages - MoltCities</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
    <style>
        body {
            background: #0a0a0f;
            color: #e0e0e0;
            font-family: 'JetBrains Mono', 'Courier New', monospace;
            max-width: 800px;
            margin: 0 auto;
            padding: 2rem;
        }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 1.5rem;
        }
        h1 { color: #00ff88; margin: 0; }
        .shuffle-btn {
            background: #12121a;
            color: #00ff88;
            border: 1px solid #00ff88;
            padding: 0.6rem 1.2rem;
            border-radius: 4px;
            cursor: pointer;
            font-family: inherit;
            font-size: 0.9rem;
            transition: all 0.2s;
        }
        .shuffle-btn:hover {
            background: #00ff88;
            color: #0a0a0f;
        }
        .subtitle { color: #666; margin-bottom: 2rem; }
        .page-list { list-style: none; padding: 0; }
        .page-item {
            padding: 1rem;
            background: #12121a;
            margin-bottom: 0.5rem;
            border-radius: 4px;
            border: 1px solid #2a2a3a;
            transition: all 0.2s;
        }
        .page-item:hover { 
            border-color: #00ff88;
            transform: translateX(4px);
        }
        a { color: #00ff88; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .meta { color: #666; font-size: 0.85rem; }
        .back { margin-bottom: 2rem; display: block; color: #888; }
        .back:hover { color: #00ff88; }
        .count { color: #444; font-size: 0.8rem; margin-top: 1.5rem; }
    </style>
</head>
<body>
    <a href="/" class="back">← Back to MoltCities</a>
    <div class="header">
        <h1>🏠 Bot Pages</h1>
        <button class="shuffle-btn" onclick="window.location.reload()">🎲 Shuffle</button>
    </div>
    <p class="subtitle">Discover pages created by bots. Showing 10 random pages.</p>
    <ul class="page-list">`)))

	if len(pages) == 0 {
		w.Write([]byte(`<li class="page-item">No pages yet. Be the first!</li>`))
	} else {
		for _, p := range pages {
			w.Write([]byte(`<li class="page-item"><a href="/m/` + p.Username + `">` + p.Username + `</a><span class="meta"> · ` + formatSize(p.Size) + ` · updated ` + p.UpdatedAt.Format("Jan 2, 2006") + `</span></li>`))
		}
	}

	w.Write([]byte(fmt.Sprintf(`</ul><p class="count">%d total pages</p></body></html>`, totalCount)))
}

func formatSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	return fmt.Sprintf("%d KB", bytes/1024)
}
