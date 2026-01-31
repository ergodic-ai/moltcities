package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ergodic/moltcities/internal/db"
)

// NewRouter creates a new HTTP router with all routes configured.
func NewRouter(database *db.DB) http.Handler {
	return NewRouterWithStaticDir(database, "web")
}

// NewRouterWithStaticDir creates a router with a custom static directory.
func NewRouterWithStaticDir(database *db.DB, staticDir string) http.Handler {
	h := NewHandler(database)

	mux := http.NewServeMux()

	// Health check (no auth)
	mux.HandleFunc("/health", h.Health)

	// Registration (no auth)
	mux.HandleFunc("/register", h.Register)

	// Whoami (requires auth)
	mux.HandleFunc("/whoami", withAuth(database, h.Whoami))

	// Canvas endpoints (no auth for reading)
	mux.HandleFunc("/canvas/image", h.GetCanvasImage)
	mux.HandleFunc("/canvas/region", h.GetCanvasRegion)
	mux.HandleFunc("/pixel", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// POST /pixel requires auth
			withAuth(database, h.EditPixel)(w, r)
		} else {
			// GET /pixel is public
			h.GetPixel(w, r)
		}
	})
	mux.HandleFunc("/pixel/history", h.GetPixelHistory)
	mux.HandleFunc("/stats", h.GetStats)

	// Channel endpoints
	mux.HandleFunc("/channels", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// POST /channels requires auth
			withAuth(database, h.CreateChannel)(w, r)
		} else {
			// GET /channels is public
			h.ListChannels(w, r)
		}
	})

	// Individual channel and messages - need path routing
	mux.HandleFunc("/channels/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/messages") {
			if r.Method == http.MethodPost {
				withAuth(database, h.PostMessage)(w, r)
			} else {
				h.GetMessages(w, r)
			}
		} else {
			h.GetChannel(w, r)
		}
	})

	// Page API endpoints
	mux.HandleFunc("/page", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			withAuth(database, h.GetMyPage)(w, r)
		case http.MethodPut, http.MethodPost:
			withAuth(database, h.UpdatePage)(w, r)
		case http.MethodDelete:
			withAuth(database, h.DeletePageHandler)(w, r)
		default:
			WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		}
	})

	// Serve user pages at /m/{username}
	mux.HandleFunc("/m/", h.ServePage)

	// API to get random pages (for homepage preview)
	mux.HandleFunc("/pages/random", h.GetRandomPages)

	// User directory
	mux.HandleFunc("/users", h.ListUsers)

	// Mail endpoints
	mux.HandleFunc("/mail", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			withAuth(database, h.GetInbox)(w, r)
		case http.MethodPost:
			withAuth(database, h.SendMail)(w, r)
		default:
			WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		}
	})

	// Individual mail messages
	mux.HandleFunc("/mail/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			withAuth(database, h.GetMessage)(w, r)
		case http.MethodDelete:
			withAuth(database, h.DeleteMail)(w, r)
		default:
			WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		}
	})

	// Serve skills documentation
	mux.HandleFunc("/moltcities.md", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		http.ServeFile(w, r, "web/moltcities.md")
	})

	// Serve CLI install script
	mux.HandleFunc("/cli/install.sh", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.ServeFile(w, r, "web/cli/install.sh")
	})

	// Serve static files for the web frontend
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Serve index.html for root path
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Try to serve static file
		filePath := filepath.Join(staticDir, path)
		if _, err := os.Stat(filePath); err == nil {
			http.ServeFile(w, r, filePath)
			return
		}

		// Fallback to index.html for SPA routing
		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
	})

	// Wrap everything with CORS
	return CORSMiddleware(mux)
}

// withAuth wraps a handler with authentication middleware.
func withAuth(database *db.DB, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		AuthMiddleware(database)(http.HandlerFunc(handler)).ServeHTTP(w, r)
	}
}
