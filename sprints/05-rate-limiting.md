# Sprint 05: Rate Limiting & Security

## Goal
Implement IP-based rate limiting and input validation.

---

## Tasks

### 5.1 IP Rate Limiting DB Methods

```go
// internal/db/ratelimit.go
func (d *DB) CheckIPRateLimit(ip, action string, limit int, windowSeconds int) (bool, error)
func (d *DB) CheckUserRateLimit(userID int64, action string, limit int, windowSeconds int) (bool, error)
```

### 5.2 Rate Limit Middleware

```go
// internal/api/middleware.go
func IPRateLimitMiddleware(db *DB, limit int, windowSeconds int) func(http.Handler) http.Handler
```

Apply to all endpoints:
- 100 requests per IP per minute (general API limit)

### 5.3 Registration Rate Limit

```go
// In register handler
// 5 registrations per IP per day (86400 seconds)
allowed, err := db.CheckIPRateLimit(ip, "register", 5, 86400)
```

### 5.4 Client IP Detection

```go
// internal/api/middleware.go
func GetClientIP(r *http.Request) string
```

Check in order:
1. `X-Forwarded-For` (first IP)
2. `X-Real-IP`
3. `r.RemoteAddr`

### 5.5 Input Validation

```go
// internal/api/validation.go
func ValidateUsername(s string) error      // alphanumeric + underscore, 3-32 chars
func ValidateChannelName(s string) error   // alphanumeric + hyphens, 3-32 chars, lowercase
func ValidateColor(s string) error         // #RRGGBB format
func ValidateCoordinate(n int) error       // 0-1023
func ValidateMessageContent(s string) error // 1-1000 chars
```

### 5.6 Error Response Format

```go
// Consistent error responses
type ErrorResponse struct {
    Error   string `json:"error"`
    Code    string `json:"code,omitempty"`     // machine-readable code
    Details string `json:"details,omitempty"`  // additional info
}
```

Example:
```json
{
  "error": "Rate limit exceeded",
  "code": "RATE_LIMITED",
  "details": "Max 5 registrations per IP per day. Try again in 23h 45m."
}
```

### 5.7 CORS Middleware

```go
func CORSMiddleware(next http.Handler) http.Handler
```

Allow:
- Origins: * (for now, can restrict later)
- Methods: GET, POST, OPTIONS
- Headers: Content-Type, Authorization, X-API-Token

---

## Rate Limits Summary

| Action | Scope | Limit | Window |
|--------|-------|-------|--------|
| API calls | IP | 100 | 1 minute |
| Registration | IP | 5 | 1 day |
| Pixel edit | User | 1 | 1 day |
| Channel creation | User | 3 | 1 day |
| Message posting | User | 10 | 1 hour |

---

## Acceptance Criteria
- [ ] General API rate limit (100/min per IP) works
- [ ] Registration rate limit (5/day per IP) works
- [ ] Rate limit errors return helpful messages with retry timing
- [ ] All user input is validated
- [ ] Invalid input returns clear error messages
- [ ] CORS headers allow cross-origin requests
- [ ] IP detection works behind proxies (X-Forwarded-For)
