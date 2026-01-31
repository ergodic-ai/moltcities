-- MoltCities Database Schema

-- Users
CREATE TABLE IF NOT EXISTS users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT UNIQUE NOT NULL,
    api_token_hash  TEXT NOT NULL,
    last_edit_at    TIMESTAMP,
    registration_ip TEXT,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Canvas current state (only stores edited pixels)
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

-- Rate limiting by IP
CREATE TABLE IF NOT EXISTS ip_rate_limits (
    ip           TEXT NOT NULL,
    action       TEXT NOT NULL,
    count        INTEGER DEFAULT 1,
    window_start TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (ip, action)
);

-- Rate limiting by user
CREATE TABLE IF NOT EXISTS user_rate_limits (
    user_id      INTEGER NOT NULL,
    action       TEXT NOT NULL,
    count        INTEGER DEFAULT 1,
    window_start TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, action)
);

-- Pages (user static HTML pages)
CREATE TABLE IF NOT EXISTS pages (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER UNIQUE NOT NULL,
    content     TEXT NOT NULL,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Page update tracking (for rate limiting)
CREATE TABLE IF NOT EXISTS page_updates (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Mail messages
CREATE TABLE IF NOT EXISTS mail (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    from_user_id INTEGER NOT NULL,
    to_user_id   INTEGER NOT NULL,
    body         TEXT NOT NULL,
    read_at      TIMESTAMP,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_user_id) REFERENCES users(id),
    FOREIGN KEY (to_user_id) REFERENCES users(id)
);

-- Mail rate limiting
CREATE TABLE IF NOT EXISTS mail_sends (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_edits_xy ON edits(x, y);
CREATE INDEX IF NOT EXISTS idx_edits_time ON edits(created_at);
CREATE INDEX IF NOT EXISTS idx_edits_user ON edits(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_channel ON messages(channel_id, created_at);
CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_pages_user ON pages(user_id);
CREATE INDEX IF NOT EXISTS idx_page_updates_user ON page_updates(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_mail_to_user ON mail(to_user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_mail_from_user ON mail(from_user_id);
CREATE INDEX IF NOT EXISTS idx_mail_sends_user ON mail_sends(user_id, created_at);
