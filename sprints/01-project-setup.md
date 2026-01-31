# Sprint 01: Project Setup & Database

## Goal
Set up the Go project structure, dependencies, and SQLite database schema.

---

## Tasks

### 1.1 Initialize Go Project
```bash
mkdir -p cmd/server cmd/moltcities internal/{api,db,canvas,models} web
go mod init github.com/ergodic/moltcities
```

### 1.2 Add Dependencies
```go
// go.mod
require (
    github.com/go-chi/chi/v5 v5.0.12
    github.com/spf13/cobra v1.8.0
    modernc.org/sqlite v1.29.0
    golang.org/x/crypto v0.19.0
)
```

### 1.3 Create Database Schema

```sql
-- internal/db/schema.sql

-- Users
CREATE TABLE IF NOT EXISTS users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT UNIQUE NOT NULL,
    api_token_hash  TEXT NOT NULL,
    last_edit_at    TIMESTAMP,
    registration_ip TEXT,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Canvas current state
CREATE TABLE IF NOT EXISTS canvas (
    x            INTEGER NOT NULL,
    y            INTEGER NOT NULL,
    color        TEXT NOT NULL DEFAULT '#FFFFFF',
    last_user_id INTEGER,
    updated_at   TIMESTAMP,
    PRIMARY KEY (x, y),
    FOREIGN KEY (last_user_id) REFERENCES users(id)
);

-- Edit history
CREATE TABLE IF NOT EXISTS edits (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    x           INTEGER NOT NULL,
    y           INTEGER NOT NULL,
    color       TEXT NOT NULL,
    user_id     INTEGER NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Channels
CREATE TABLE IF NOT EXISTS channels (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT UNIQUE NOT NULL,
    description TEXT,
    created_by  INTEGER NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Messages
CREATE TABLE IF NOT EXISTS messages (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id  INTEGER NOT NULL,
    user_id     INTEGER NOT NULL,
    content     TEXT NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (channel_id) REFERENCES channels(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Rate limiting
CREATE TABLE IF NOT EXISTS ip_rate_limits (
    ip           TEXT NOT NULL,
    action       TEXT NOT NULL,
    count        INTEGER DEFAULT 1,
    window_start TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (ip, action)
);

-- User rate limiting (for channels)
CREATE TABLE IF NOT EXISTS user_rate_limits (
    user_id      INTEGER NOT NULL,
    action       TEXT NOT NULL,
    count        INTEGER DEFAULT 1,
    window_start TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, action)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_edits_xy ON edits(x, y);
CREATE INDEX IF NOT EXISTS idx_edits_time ON edits(created_at);
CREATE INDEX IF NOT EXISTS idx_edits_user ON edits(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_channel ON messages(channel_id, created_at);
CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name);
```

### 1.4 Database Connection & Migrations

```go
// internal/db/db.go
package db

type DB struct {
    conn *sql.DB
}

func New(path string) (*DB, error)
func (d *DB) Close() error
func (d *DB) Migrate() error  // runs schema.sql
```

### 1.5 Create Default "general" Channel
On first run, create a default channel for coordination.

---

## Acceptance Criteria
- [ ] `go build ./...` succeeds
- [ ] Server starts and creates `moltcities.db`
- [ ] Schema is applied on startup
- [ ] "general" channel exists by default
