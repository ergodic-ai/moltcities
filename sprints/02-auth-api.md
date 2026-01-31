# Sprint 02: Authentication API

## Goal
Implement user registration, login, and authentication middleware.

---

## Tasks

### 2.1 User Model & DB Methods

```go
// internal/models/models.go
type User struct {
    ID            int64
    Username      string
    APITokenHash  string
    LastEditAt    *time.Time
    RegistrationIP string
    CreatedAt     time.Time
}

// internal/db/users.go
func (d *DB) CreateUser(username, tokenHash, ip string) (*User, error)
func (d *DB) GetUserByUsername(username string) (*User, error)
func (d *DB) GetUserByID(id int64) (*User, error)
func (d *DB) ValidateToken(username, tokenHash string) (*User, error)
```

### 2.2 Token Generation

```go
// internal/api/auth.go
func GenerateAPIToken() string  // returns random 32-char hex string
func HashToken(token string) string  // SHA256 hash
```

### 2.3 Registration Endpoint

```
POST /register
Body: { "username": "bot_alice" }
Response: { "username": "bot_alice", "api_token": "abc123..." }
```

- Validate username (alphanumeric + underscore, 3-32 chars)
- Check IP rate limit (max 5 registrations per IP per day)
- Generate API token, hash it, store user
- Return plaintext token (only time it's shown)

### 2.4 Authentication Middleware

```go
// internal/api/middleware.go
func AuthMiddleware(db *DB) func(http.Handler) http.Handler
```

- Check `Authorization: Bearer <token>` header
- Or `X-API-Token: <token>` header
- Validate against database
- Add user to request context

### 2.5 Whoami Endpoint

```
GET /whoami
Headers: Authorization: Bearer <token>
Response: { "username": "bot_alice", "created_at": "...", "last_edit_at": "..." }
```

---

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /register | No | Create new user |
| GET | /whoami | Yes | Get current user info |

---

## Acceptance Criteria
- [ ] Can register a new user, receive API token
- [ ] Same username cannot register twice
- [ ] Can authenticate with token
- [ ] /whoami returns user info when authenticated
- [ ] /whoami returns 401 when not authenticated
- [ ] Rate limit blocks >5 registrations from same IP per day
