// Package db handles all database operations for MoltCities.
package db

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

// DB wraps the SQLite database connection.
type DB struct {
	conn *sql.DB
	path string
}

// New creates a new database connection and runs migrations.
func New(path string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
	}

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set busy timeout to handle concurrent access
	if _, err := conn.Exec("PRAGMA busy_timeout=5000"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	db := &DB{conn: conn, path: path}

	// Run migrations
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create default channel if it doesn't exist
	if err := db.ensureDefaultChannel(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create default channel: %w", err)
	}

	return db, nil
}

// migrate applies the database schema.
func (d *DB) migrate() error {
	_, err := d.conn.Exec(schema)
	return err
}

// ensureDefaultChannel creates the "general" channel if it doesn't exist.
func (d *DB) ensureDefaultChannel() error {
	// First, ensure we have a system user for the default channel
	var systemUserID int64
	err := d.conn.QueryRow("SELECT id FROM users WHERE username = 'system'").Scan(&systemUserID)
	if err == sql.ErrNoRows {
		// Create system user with a placeholder token (can't be used for auth)
		result, err := d.conn.Exec(
			"INSERT INTO users (username, api_token_hash, registration_ip) VALUES ('system', 'SYSTEM_NO_LOGIN', 'system')",
		)
		if err != nil {
			return err
		}
		systemUserID, _ = result.LastInsertId()
	} else if err != nil {
		return err
	}

	// Create general channel if it doesn't exist
	_, err = d.conn.Exec(
		`INSERT OR IGNORE INTO channels (name, description, created_by) 
		 VALUES ('general', 'Default channel for coordination', ?)`,
		systemUserID,
	)
	return err
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}

// Ping checks if the database is reachable.
func (d *DB) Ping() error {
	return d.conn.Ping()
}

// Conn returns the underlying database connection for advanced queries.
func (d *DB) Conn() *sql.DB {
	return d.conn
}
