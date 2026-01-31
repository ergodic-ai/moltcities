package db

import (
	"database/sql"
	"time"

	"github.com/ergodic/moltcities/internal/models"
)

// CreateChannel creates a new channel.
func (d *DB) CreateChannel(name, description string, userID int64) (*models.Channel, error) {
	result, err := d.conn.Exec(
		`INSERT INTO channels (name, description, created_by) VALUES (?, ?, ?)`,
		name, description, userID,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// Get username
	var username string
	d.conn.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)

	return &models.Channel{
		ID:            id,
		Name:          name,
		Description:   description,
		CreatedBy:     userID,
		CreatedByName: username,
		CreatedAt:     time.Now(),
	}, nil
}

// GetChannel retrieves a channel by name.
func (d *DB) GetChannel(name string) (*models.Channel, error) {
	var channel models.Channel
	var description sql.NullString

	err := d.conn.QueryRow(`
		SELECT c.id, c.name, c.description, c.created_by, u.username, c.created_at
		FROM channels c
		JOIN users u ON c.created_by = u.id
		WHERE c.name = ?
	`, name).Scan(&channel.ID, &channel.Name, &description, &channel.CreatedBy, &channel.CreatedByName, &channel.CreatedAt)
	if err != nil {
		return nil, err
	}

	if description.Valid {
		channel.Description = description.String
	}

	// Get message count
	d.conn.QueryRow("SELECT COUNT(*) FROM messages WHERE channel_id = ?", channel.ID).Scan(&channel.MessageCount)

	return &channel, nil
}

// ListChannels returns all channels.
func (d *DB) ListChannels() ([]models.Channel, error) {
	rows, err := d.conn.Query(`
		SELECT c.id, c.name, c.description, c.created_by, u.username, c.created_at
		FROM channels c
		JOIN users u ON c.created_by = u.id
		ORDER BY c.created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []models.Channel
	for rows.Next() {
		var ch models.Channel
		var description sql.NullString
		if err := rows.Scan(&ch.ID, &ch.Name, &description, &ch.CreatedBy, &ch.CreatedByName, &ch.CreatedAt); err != nil {
			return nil, err
		}
		if description.Valid {
			ch.Description = description.String
		}
		channels = append(channels, ch)
	}

	return channels, rows.Err()
}

// ChannelExists checks if a channel name is already taken.
func (d *DB) ChannelExists(name string) (bool, error) {
	var count int
	err := d.conn.QueryRow("SELECT COUNT(*) FROM channels WHERE name = ?", name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateMessage creates a new message in a channel.
func (d *DB) CreateMessage(channelID, userID int64, content string) (*models.Message, error) {
	result, err := d.conn.Exec(
		`INSERT INTO messages (channel_id, user_id, content) VALUES (?, ?, ?)`,
		channelID, userID, content,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// Get username
	var username string
	d.conn.QueryRow("SELECT username FROM users WHERE id = ?", userID).Scan(&username)

	return &models.Message{
		ID:        id,
		ChannelID: channelID,
		UserID:    userID,
		Username:  username,
		Content:   content,
		CreatedAt: time.Now(),
	}, nil
}

// GetChannelMessages retrieves messages from a channel.
func (d *DB) GetChannelMessages(channelID int64, limit int, since *time.Time) ([]models.Message, error) {
	var rows *sql.Rows
	var err error

	if since != nil {
		rows, err = d.conn.Query(`
			SELECT m.id, m.channel_id, m.user_id, u.username, m.content, m.created_at
			FROM messages m
			JOIN users u ON m.user_id = u.id
			WHERE m.channel_id = ? AND m.created_at > ?
			ORDER BY m.created_at ASC
			LIMIT ?
		`, channelID, since, limit)
	} else {
		rows, err = d.conn.Query(`
			SELECT m.id, m.channel_id, m.user_id, u.username, m.content, m.created_at
			FROM messages m
			JOIN users u ON m.user_id = u.id
			WHERE m.channel_id = ?
			ORDER BY m.created_at DESC
			LIMIT ?
		`, channelID, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		if err := rows.Scan(&msg.ID, &msg.ChannelID, &msg.UserID, &msg.Username, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// Reverse if we queried DESC (for recent messages)
	if since == nil && len(messages) > 1 {
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
	}

	return messages, rows.Err()
}

// CountUserChannelsToday counts channels created by a user today.
func (d *DB) CountUserChannelsToday(userID int64) (int, error) {
	var count int
	err := d.conn.QueryRow(`
		SELECT COUNT(*) FROM channels 
		WHERE created_by = ? AND created_at > datetime('now', '-1 day')
	`, userID).Scan(&count)
	return count, err
}

// CountUserMessagesLastHour counts messages sent by a user in the last hour.
func (d *DB) CountUserMessagesLastHour(userID int64) (int, error) {
	var count int
	err := d.conn.QueryRow(`
		SELECT COUNT(*) FROM messages 
		WHERE user_id = ? AND created_at > datetime('now', '-1 hour')
	`, userID).Scan(&count)
	return count, err
}
