package db

import (
	"database/sql"
	"time"
)

// CheckIPRateLimit checks if an IP has exceeded the rate limit for an action.
// Returns (allowed, error).
func (d *DB) CheckIPRateLimit(ip, action string, limit int, windowSeconds int) (bool, error) {
	now := time.Now()
	windowStart := now.Add(-time.Duration(windowSeconds) * time.Second)

	var count int
	var dbWindowStart time.Time

	err := d.conn.QueryRow(`
		SELECT count, window_start FROM ip_rate_limits 
		WHERE ip = ? AND action = ?
	`, ip, action).Scan(&count, &dbWindowStart)

	if err == sql.ErrNoRows {
		// First request - create entry
		_, err = d.conn.Exec(`
			INSERT INTO ip_rate_limits (ip, action, count, window_start) 
			VALUES (?, ?, 1, ?)
		`, ip, action, now)
		return true, err
	}

	if err != nil {
		return false, err
	}

	if dbWindowStart.Before(windowStart) {
		// Window expired - reset
		_, err = d.conn.Exec(`
			UPDATE ip_rate_limits 
			SET count = 1, window_start = ? 
			WHERE ip = ? AND action = ?
		`, now, ip, action)
		return true, err
	}

	if count >= limit {
		// Rate limited
		return false, nil
	}

	// Increment
	_, err = d.conn.Exec(`
		UPDATE ip_rate_limits SET count = count + 1 
		WHERE ip = ? AND action = ?
	`, ip, action)
	return true, err
}

// CheckUserRateLimit checks if a user has exceeded the rate limit for an action.
func (d *DB) CheckUserRateLimit(userID int64, action string, limit int, windowSeconds int) (bool, error) {
	now := time.Now()
	windowStart := now.Add(-time.Duration(windowSeconds) * time.Second)

	var count int
	var dbWindowStart time.Time

	err := d.conn.QueryRow(`
		SELECT count, window_start FROM user_rate_limits 
		WHERE user_id = ? AND action = ?
	`, userID, action).Scan(&count, &dbWindowStart)

	if err == sql.ErrNoRows {
		_, err = d.conn.Exec(`
			INSERT INTO user_rate_limits (user_id, action, count, window_start) 
			VALUES (?, ?, 1, ?)
		`, userID, action, now)
		return true, err
	}

	if err != nil {
		return false, err
	}

	if dbWindowStart.Before(windowStart) {
		_, err = d.conn.Exec(`
			UPDATE user_rate_limits 
			SET count = 1, window_start = ? 
			WHERE user_id = ? AND action = ?
		`, now, userID, action)
		return true, err
	}

	if count >= limit {
		return false, nil
	}

	_, err = d.conn.Exec(`
		UPDATE user_rate_limits SET count = count + 1 
		WHERE user_id = ? AND action = ?
	`, userID, action)
	return true, err
}

// CleanupOldRateLimits removes expired rate limit entries.
func (d *DB) CleanupOldRateLimits() error {
	// Delete entries older than 24 hours
	cutoff := time.Now().Add(-24 * time.Hour)

	_, err := d.conn.Exec(`DELETE FROM ip_rate_limits WHERE window_start < ?`, cutoff)
	if err != nil {
		return err
	}

	_, err = d.conn.Exec(`DELETE FROM user_rate_limits WHERE window_start < ?`, cutoff)
	return err
}
