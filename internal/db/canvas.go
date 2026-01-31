package db

import (
	"database/sql"
	"time"

	"github.com/ergodic/moltcities/internal/models"
)

// GetPixel retrieves a single pixel's information.
func (d *DB) GetPixel(x, y int) (*models.Pixel, error) {
	var pixel models.Pixel
	var username sql.NullString
	var updatedAt sql.NullTime

	err := d.conn.QueryRow(`
		SELECT c.x, c.y, c.color, u.username, c.updated_at
		FROM canvas c
		LEFT JOIN users u ON c.last_user_id = u.id
		WHERE c.x = ? AND c.y = ?
	`, x, y).Scan(&pixel.X, &pixel.Y, &pixel.Color, &username, &updatedAt)

	if err == sql.ErrNoRows {
		// Pixel never edited - return white
		return &models.Pixel{
			X:     x,
			Y:     y,
			Color: "#FFFFFF",
		}, nil
	}
	if err != nil {
		return nil, err
	}

	if username.Valid {
		pixel.EditedBy = &username.String
	}
	if updatedAt.Valid {
		pixel.EditedAt = &updatedAt.Time
	}

	return &pixel, nil
}

// SetPixel updates a pixel's color and records the edit in history.
func (d *DB) SetPixel(x, y int, color string, userID int64) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Upsert into canvas table
	_, err = tx.Exec(`
		INSERT INTO canvas (x, y, color, last_user_id, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (x, y) DO UPDATE SET
			color = excluded.color,
			last_user_id = excluded.last_user_id,
			updated_at = excluded.updated_at
	`, x, y, color, userID)
	if err != nil {
		return err
	}

	// Insert into edit history
	_, err = tx.Exec(`
		INSERT INTO edits (x, y, color, user_id)
		VALUES (?, ?, ?, ?)
	`, x, y, color, userID)
	if err != nil {
		return err
	}

	// Update user's last edit time
	_, err = tx.Exec(`
		UPDATE users SET last_edit_at = CURRENT_TIMESTAMP WHERE id = ?
	`, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetRegion retrieves a rectangular region of pixels.
// Returns a 2D array of colors [row][col].
func (d *DB) GetRegion(x, y, width, height int) ([][]string, error) {
	// Initialize with white
	pixels := make([][]string, height)
	for row := 0; row < height; row++ {
		pixels[row] = make([]string, width)
		for col := 0; col < width; col++ {
			pixels[row][col] = "#FFFFFF"
		}
	}

	// Query edited pixels in the region
	rows, err := d.conn.Query(`
		SELECT x, y, color FROM canvas
		WHERE x >= ? AND x < ? AND y >= ? AND y < ?
	`, x, x+width, y, y+height)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var px, py int
		var color string
		if err := rows.Scan(&px, &py, &color); err != nil {
			return nil, err
		}
		// Convert to local coordinates
		localX := px - x
		localY := py - y
		if localY >= 0 && localY < height && localX >= 0 && localX < width {
			pixels[localY][localX] = color
		}
	}

	return pixels, rows.Err()
}

// GetAllPixels retrieves all edited pixels for image generation.
// Returns a map of (x,y) -> color.
func (d *DB) GetAllPixels() (map[[2]int]string, error) {
	pixels := make(map[[2]int]string)

	rows, err := d.conn.Query("SELECT x, y, color FROM canvas")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var x, y int
		var color string
		if err := rows.Scan(&x, &y, &color); err != nil {
			return nil, err
		}
		pixels[[2]int{x, y}] = color
	}

	return pixels, rows.Err()
}

// GetPixelHistory retrieves the edit history for a pixel.
func (d *DB) GetPixelHistory(x, y int, limit int) ([]models.Edit, error) {
	rows, err := d.conn.Query(`
		SELECT e.id, e.x, e.y, e.color, e.user_id, u.username, e.created_at
		FROM edits e
		JOIN users u ON e.user_id = u.id
		WHERE e.x = ? AND e.y = ?
		ORDER BY e.created_at DESC
		LIMIT ?
	`, x, y, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edits []models.Edit
	for rows.Next() {
		var edit models.Edit
		if err := rows.Scan(&edit.ID, &edit.X, &edit.Y, &edit.Color, &edit.UserID, &edit.Username, &edit.CreatedAt); err != nil {
			return nil, err
		}
		edits = append(edits, edit)
	}

	return edits, rows.Err()
}

// GetStats retrieves canvas and user statistics.
func (d *DB) GetStats() (*models.Stats, error) {
	var stats models.Stats

	// Total edits
	d.conn.QueryRow("SELECT COUNT(*) FROM edits").Scan(&stats.TotalEdits)

	// Unique pixels
	d.conn.QueryRow("SELECT COUNT(*) FROM canvas").Scan(&stats.UniquePixels)

	// Total users (excluding system)
	d.conn.QueryRow("SELECT COUNT(*) FROM users WHERE username != 'system'").Scan(&stats.TotalUsers)

	// Total channels
	d.conn.QueryRow("SELECT COUNT(*) FROM channels").Scan(&stats.TotalChannels)

	// Total messages
	d.conn.QueryRow("SELECT COUNT(*) FROM messages").Scan(&stats.TotalMessages)

	return &stats, nil
}

// CanUserEditNow checks if user can edit and returns time until next edit.
func (d *DB) CanUserEditNow(userID int64) (bool, *time.Time, error) {
	var lastEditAt sql.NullTime

	err := d.conn.QueryRow(
		"SELECT last_edit_at FROM users WHERE id = ?",
		userID,
	).Scan(&lastEditAt)
	if err != nil {
		return false, nil, err
	}

	if !lastEditAt.Valid {
		return true, nil, nil // Never edited
	}

	nextEdit := lastEditAt.Time.Add(24 * time.Hour)
	if time.Now().After(nextEdit) {
		return true, nil, nil
	}

	return false, &nextEdit, nil
}
