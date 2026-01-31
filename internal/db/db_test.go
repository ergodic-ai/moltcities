package db

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNew verifies database creation and migrations.
func TestNew(t *testing.T) {
	// Create temp directory for test database
	tmpDir, err := os.MkdirTemp("", "moltcities-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create database
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	// Verify database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}

	// Verify we can ping
	if err := db.Ping(); err != nil {
		t.Errorf("ping failed: %v", err)
	}
}

// TestMigrations verifies all tables are created.
func TestMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tables := []string{"users", "canvas", "edits", "channels", "messages", "ip_rate_limits", "user_rate_limits"}

	for _, table := range tables {
		var name string
		err := db.conn.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %s does not exist: %v", table, err)
		}
	}
}

// TestDefaultChannel verifies the "general" channel is created.
func TestDefaultChannel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	var channelName string
	var description string
	err := db.conn.QueryRow(
		"SELECT name, description FROM channels WHERE name = 'general'",
	).Scan(&channelName, &description)

	if err != nil {
		t.Fatalf("general channel not found: %v", err)
	}

	if channelName != "general" {
		t.Errorf("expected channel name 'general', got '%s'", channelName)
	}

	if description != "Default channel for coordination" {
		t.Errorf("unexpected description: %s", description)
	}
}

// TestSystemUser verifies the system user is created for the default channel.
func TestSystemUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	var username string
	err := db.conn.QueryRow(
		"SELECT username FROM users WHERE username = 'system'",
	).Scan(&username)

	if err != nil {
		t.Fatalf("system user not found: %v", err)
	}

	if username != "system" {
		t.Errorf("expected username 'system', got '%s'", username)
	}
}

// TestWALMode verifies WAL mode is enabled.
func TestWALMode(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	var journalMode string
	err := db.conn.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to get journal mode: %v", err)
	}

	if journalMode != "wal" {
		t.Errorf("expected journal_mode 'wal', got '%s'", journalMode)
	}
}

// TestIndexes verifies all indexes are created.
func TestIndexes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	indexes := []string{
		"idx_edits_xy",
		"idx_edits_time",
		"idx_edits_user",
		"idx_messages_channel",
		"idx_channels_name",
		"idx_users_username",
	}

	for _, idx := range indexes {
		var name string
		err := db.conn.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='index' AND name=?",
			idx,
		).Scan(&name)
		if err != nil {
			t.Errorf("index %s does not exist: %v", idx, err)
		}
	}
}

// TestReopen verifies database can be reopened without error.
func TestReopen(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "moltcities-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create and close database
	db1, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	db1.Close()

	// Reopen database
	db2, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to reopen database: %v", err)
	}
	defer db2.Close()

	// Verify general channel still exists
	var count int
	err = db2.conn.QueryRow("SELECT COUNT(*) FROM channels WHERE name = 'general'").Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 general channel, got %d", count)
	}
}

// setupTestDB creates a temporary test database.
func setupTestDB(t *testing.T) *DB {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "moltcities-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	return db
}
