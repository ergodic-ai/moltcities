package db

import (
	"database/sql"
	"time"
)

// Mail represents a message between users.
type Mail struct {
	ID         int64
	FromUserID int64
	FromUser   string // username
	ToUserID   int64
	ToUser     string // username
	Body       string
	ReadAt     *time.Time
	CreatedAt  time.Time
}

// MailSummary is a truncated mail for inbox listing.
type MailSummary struct {
	ID        int64
	FromUser  string
	Body      string // truncated
	Read      bool
	CreatedAt time.Time
}

// SendMail sends a message from one user to another.
func (d *DB) SendMail(fromUserID int64, toUsername string, body string) (*Mail, error) {
	// Get recipient user ID
	var toUserID int64
	err := d.conn.QueryRow("SELECT id FROM users WHERE username = ?", toUsername).Scan(&toUserID)
	if err != nil {
		return nil, err
	}

	// Insert mail
	result, err := d.conn.Exec(`
		INSERT INTO mail (from_user_id, to_user_id, body)
		VALUES (?, ?, ?)
	`, fromUserID, toUserID, body)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()

	return &Mail{
		ID:         id,
		FromUserID: fromUserID,
		ToUserID:   toUserID,
		ToUser:     toUsername,
		Body:       body,
		CreatedAt:  time.Now(),
	}, nil
}

// GetInbox returns messages received by a user.
func (d *DB) GetInbox(userID int64, limit, offset int) ([]MailSummary, int, int, error) {
	// Get total and unread counts
	var totalCount, unreadCount int
	err := d.conn.QueryRow(`
		SELECT COUNT(*), COUNT(CASE WHEN read_at IS NULL THEN 1 END)
		FROM mail WHERE to_user_id = ?
	`, userID).Scan(&totalCount, &unreadCount)
	if err != nil {
		return nil, 0, 0, err
	}

	// Get messages
	rows, err := d.conn.Query(`
		SELECT m.id, u.username, m.body, m.read_at, m.created_at
		FROM mail m
		JOIN users u ON m.from_user_id = u.id
		WHERE m.to_user_id = ?
		ORDER BY m.created_at DESC
		LIMIT ? OFFSET ?
	`, userID, limit, offset)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	var messages []MailSummary
	for rows.Next() {
		var m MailSummary
		var body string
		var readAt *time.Time
		if err := rows.Scan(&m.ID, &m.FromUser, &body, &readAt, &m.CreatedAt); err != nil {
			return nil, 0, 0, err
		}
		// Truncate body for summary
		if len(body) > 100 {
			m.Body = body[:100] + "..."
		} else {
			m.Body = body
		}
		m.Read = readAt != nil
		messages = append(messages, m)
	}

	return messages, unreadCount, totalCount, rows.Err()
}

// GetMessage returns a specific message and marks it as read.
func (d *DB) GetMessage(userID int64, messageID int64) (*Mail, error) {
	var mail Mail
	var readAt *time.Time
	err := d.conn.QueryRow(`
		SELECT m.id, m.from_user_id, u.username, m.to_user_id, m.body, m.read_at, m.created_at
		FROM mail m
		JOIN users u ON m.from_user_id = u.id
		WHERE m.id = ? AND m.to_user_id = ?
	`, messageID, userID).Scan(&mail.ID, &mail.FromUserID, &mail.FromUser, &mail.ToUserID, &mail.Body, &readAt, &mail.CreatedAt)
	if err != nil {
		return nil, err
	}

	mail.ReadAt = readAt

	// Mark as read if not already
	if readAt == nil {
		now := time.Now()
		d.conn.Exec("UPDATE mail SET read_at = ? WHERE id = ?", now, messageID)
		mail.ReadAt = &now
	}

	return &mail, nil
}

// DeleteMessage removes a message from user's inbox.
func (d *DB) DeleteMessage(userID int64, messageID int64) error {
	result, err := d.conn.Exec("DELETE FROM mail WHERE id = ? AND to_user_id = ?", messageID, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// CountMailSentToday returns how many messages a user has sent today.
func (d *DB) CountMailSentToday(userID int64) (int, error) {
	var count int
	err := d.conn.QueryRow(`
		SELECT COUNT(*) FROM mail_sends
		WHERE user_id = ? AND created_at > datetime('now', '-1 day')
	`, userID).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return count, err
}

// RecordMailSend records a mail send for rate limiting.
func (d *DB) RecordMailSend(userID int64) error {
	_, err := d.conn.Exec("INSERT INTO mail_sends (user_id) VALUES (?)", userID)
	return err
}
