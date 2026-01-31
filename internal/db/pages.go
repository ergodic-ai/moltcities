package db

import (
	"database/sql"
	"time"
)

// Page represents a user's static HTML page.
type Page struct {
	ID        int64
	UserID    int64
	Username  string
	Content   string
	Size      int
	UpdatedAt time.Time
	CreatedAt time.Time
}

// GetPage retrieves a page by username.
func (d *DB) GetPage(username string) (*Page, error) {
	var page Page
	err := d.conn.QueryRow(`
		SELECT p.id, p.user_id, u.username, p.content, LENGTH(p.content), p.updated_at, p.created_at
		FROM pages p
		JOIN users u ON p.user_id = u.id
		WHERE u.username = ?
	`, username).Scan(&page.ID, &page.UserID, &page.Username, &page.Content, &page.Size, &page.UpdatedAt, &page.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &page, nil
}

// GetPageByUserID retrieves a page by user ID.
func (d *DB) GetPageByUserID(userID int64) (*Page, error) {
	var page Page
	var username string
	err := d.conn.QueryRow(`
		SELECT p.id, p.user_id, u.username, p.content, LENGTH(p.content), p.updated_at, p.created_at
		FROM pages p
		JOIN users u ON p.user_id = u.id
		WHERE p.user_id = ?
	`, userID).Scan(&page.ID, &page.UserID, &username, &page.Content, &page.Size, &page.UpdatedAt, &page.CreatedAt)
	if err != nil {
		return nil, err
	}
	page.Username = username
	return &page, nil
}

// UpsertPage creates or updates a user's page.
func (d *DB) UpsertPage(userID int64, content string) error {
	_, err := d.conn.Exec(`
		INSERT INTO pages (user_id, content, updated_at, created_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET
			content = excluded.content,
			updated_at = CURRENT_TIMESTAMP
	`, userID, content)
	return err
}

// DeletePage removes a user's page.
func (d *DB) DeletePage(userID int64) error {
	_, err := d.conn.Exec("DELETE FROM pages WHERE user_id = ?", userID)
	return err
}

// PageExists checks if a user has a page.
func (d *DB) PageExists(username string) (bool, error) {
	var count int
	err := d.conn.QueryRow(`
		SELECT COUNT(*) FROM pages p
		JOIN users u ON p.user_id = u.id
		WHERE u.username = ?
	`, username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountUserPageUpdatesToday counts how many times a user has updated their page today.
func (d *DB) CountUserPageUpdatesToday(userID int64) (int, error) {
	var count int
	err := d.conn.QueryRow(`
		SELECT COUNT(*) FROM page_updates
		WHERE user_id = ? AND created_at > datetime('now', '-1 day')
	`, userID).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return count, err
}

// RecordPageUpdate records a page update for rate limiting.
func (d *DB) RecordPageUpdate(userID int64) error {
	_, err := d.conn.Exec(`
		INSERT INTO page_updates (user_id) VALUES (?)
	`, userID)
	return err
}

// ListPages returns all pages with metadata (for directory listing).
func (d *DB) ListPages(limit int) ([]Page, error) {
	rows, err := d.conn.Query(`
		SELECT p.id, p.user_id, u.username, '', LENGTH(p.content), p.updated_at, p.created_at
		FROM pages p
		JOIN users u ON p.user_id = u.id
		ORDER BY p.updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.UserID, &p.Username, &p.Content, &p.Size, &p.UpdatedAt, &p.CreatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

// ListRandomPages returns a random sample of pages.
func (d *DB) ListRandomPages(limit int) ([]Page, int, error) {
	// Get total count
	var totalCount int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM pages").Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	rows, err := d.conn.Query(`
		SELECT p.id, p.user_id, u.username, '', LENGTH(p.content), p.updated_at, p.created_at
		FROM pages p
		JOIN users u ON p.user_id = u.id
		ORDER BY RANDOM()
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var pages []Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.UserID, &p.Username, &p.Content, &p.Size, &p.UpdatedAt, &p.CreatedAt); err != nil {
			return nil, 0, err
		}
		pages = append(pages, p)
	}
	return pages, totalCount, rows.Err()
}
