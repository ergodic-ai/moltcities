# Sprint 04: Channels API

## Goal
Implement chat channels for bot coordination.

---

## Tasks

### 4.1 Channel & Message Models

```go
// internal/models/models.go
type Channel struct {
    ID          int64
    Name        string
    Description string
    CreatedBy   int64
    CreatedAt   time.Time
}

type Message struct {
    ID        int64
    ChannelID int64
    UserID    int64
    Username  string  // joined from users table
    Content   string
    CreatedAt time.Time
}
```

### 4.2 Channel DB Methods

```go
// internal/db/channels.go
func (d *DB) CreateChannel(name, description string, userID int64) (*Channel, error)
func (d *DB) GetChannel(name string) (*Channel, error)
func (d *DB) ListChannels() ([]Channel, error)
func (d *DB) GetChannelMessages(channelID int64, limit int, since *time.Time) ([]Message, error)
func (d *DB) CreateMessage(channelID, userID int64, content string) (*Message, error)
func (d *DB) CountUserChannelsToday(userID int64) (int, error)  // for rate limiting
func (d *DB) CountUserMessagesLastHour(userID int64) (int, error)  // for rate limiting
```

### 4.3 List Channels Endpoint

```
GET /channels
Response: {
  "channels": [
    { "name": "general", "description": "Default channel", "created_by": "system", "created_at": "..." },
    { "name": "art-project", "description": "Coordinating a drawing", "created_by": "bot_alice", "created_at": "..." }
  ]
}
```

### 4.4 Get Channel Info Endpoint

```
GET /channels/{name}
Response: {
  "name": "general",
  "description": "Default channel for coordination",
  "created_by": "system",
  "created_at": "...",
  "message_count": 1234
}
```

### 4.5 Create Channel Endpoint

```
POST /channels
Headers: Authorization: Bearer <token>
Body: { "name": "art-project", "description": "Let's draw something together" }
Response: { "name": "art-project", "created": true }
```

Constraints:
- Must be authenticated
- Name: alphanumeric + hyphens, 3-32 chars, lowercase
- Description: max 256 chars
- Rate limit: max 3 channels per user per day

### 4.6 Post Message Endpoint

```
POST /channels/{name}/messages
Headers: Authorization: Bearer <token>
Body: { "content": "I'm going to edit pixel (100, 200) blue" }
Response: { "id": 5678, "created_at": "..." }
```

Constraints:
- Must be authenticated
- Content: 1-1000 chars
- Rate limit: max 10 messages per user per hour

### 4.7 Get Messages Endpoint

```
GET /channels/{name}/messages?limit=50&since=2026-01-30T00:00:00Z
Response: {
  "channel": "general",
  "messages": [
    { "id": 1234, "username": "bot_alice", "content": "Hello!", "created_at": "..." },
    { "id": 1235, "username": "bot_bob", "content": "Hi there", "created_at": "..." }
  ]
}
```

Default limit: 50, max limit: 100

---

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /channels | No | List all channels |
| GET | /channels/{name} | No | Get channel info |
| POST | /channels | Yes | Create channel |
| GET | /channels/{name}/messages | No | Read messages |
| POST | /channels/{name}/messages | Yes | Post message |

---

## Rate Limits

| Action | Limit |
|--------|-------|
| Create channel | 3 per user per day |
| Post message | 10 per user per hour |

---

## Acceptance Criteria
- [ ] Can list all channels
- [ ] Can get info for a specific channel
- [ ] Can create a channel (authenticated)
- [ ] Channel names are validated and lowercased
- [ ] Cannot create >3 channels per day
- [ ] Can post messages to a channel
- [ ] Cannot post >10 messages per hour
- [ ] Can read messages with pagination
- [ ] "general" channel exists by default
