package db

import (
	"database/sql"
	"time"

	"github.com/ergodic/moltcities/internal/models"
)

// CreateUser creates a new user with the given username and hashed token.
func (d *DB) CreateUser(username, tokenHash, ip string) (*models.User, error) {
	result, err := d.conn.Exec(
		`INSERT INTO users (username, api_token_hash, registration_ip) VALUES (?, ?, ?)`,
		username, tokenHash, ip,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &models.User{
		ID:             id,
		Username:       username,
		APITokenHash:   tokenHash,
		RegistrationIP: ip,
		CreatedAt:      time.Now(),
	}, nil
}

// GetUserByUsername retrieves a user by username.
func (d *DB) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	var lastEditAt sql.NullTime

	err := d.conn.QueryRow(
		`SELECT id, username, api_token_hash, last_edit_at, registration_ip, created_at 
		 FROM users WHERE username = ?`,
		username,
	).Scan(&user.ID, &user.Username, &user.APITokenHash, &lastEditAt, &user.RegistrationIP, &user.CreatedAt)

	if err != nil {
		return nil, err
	}

	if lastEditAt.Valid {
		user.LastEditAt = &lastEditAt.Time
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID.
func (d *DB) GetUserByID(id int64) (*models.User, error) {
	var user models.User
	var lastEditAt sql.NullTime

	err := d.conn.QueryRow(
		`SELECT id, username, api_token_hash, last_edit_at, registration_ip, created_at 
		 FROM users WHERE id = ?`,
		id,
	).Scan(&user.ID, &user.Username, &user.APITokenHash, &lastEditAt, &user.RegistrationIP, &user.CreatedAt)

	if err != nil {
		return nil, err
	}

	if lastEditAt.Valid {
		user.LastEditAt = &lastEditAt.Time
	}

	return &user, nil
}

// ValidateUserToken checks if the given token hash matches the user's stored hash.
func (d *DB) ValidateUserToken(username, tokenHash string) (*models.User, error) {
	user, err := d.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}

	if user.APITokenHash != tokenHash {
		return nil, sql.ErrNoRows // Token doesn't match
	}

	return user, nil
}

// UsernameExists checks if a username is already taken.
func (d *DB) UsernameExists(username string) (bool, error) {
	var count int
	err := d.conn.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username = ?",
		username,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateUserLastEdit updates the user's last edit timestamp.
func (d *DB) UpdateUserLastEdit(userID int64) error {
	_, err := d.conn.Exec(
		"UPDATE users SET last_edit_at = CURRENT_TIMESTAMP WHERE id = ?",
		userID,
	)
	return err
}

// CanUserEdit checks if the user can edit (hasn't edited in the last 24 hours).
// Returns (canEdit, nextEditTime, error).
func (d *DB) CanUserEdit(userID int64) (bool, *time.Time, error) {
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

// UserSummary is a public view of a user for the directory.
type UserSummary struct {
	Username  string
	CreatedAt time.Time
}

// ListUsers returns all users for the directory.
func (d *DB) ListUsers(limit, offset int) ([]UserSummary, int, error) {
	// Get total count
	var totalCount int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM users WHERE username != 'system'").Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	// Get users
	rows, err := d.conn.Query(`
		SELECT username, created_at FROM users
		WHERE username != 'system'
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []UserSummary
	for rows.Next() {
		var u UserSummary
		if err := rows.Scan(&u.Username, &u.CreatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	return users, totalCount, rows.Err()
}
