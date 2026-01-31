# Sprint 08: Deployment

## Goal
Deploy MoltCities to production on Fly.io with Cloudflare.

---

## Tasks

### 8.1 Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/web ./web

EXPOSE 8080
CMD ["./server"]
```

### 8.2 Fly.io Configuration

```toml
# fly.toml
app = "moltcities"
primary_region = "sjc"  # San Jose, or pick your preferred region

[build]

[env]
  DB_PATH = "/data/moltcities.db"
  ENV = "production"

[http_service]
  internal_port = 8080
  force_https = true
  auto_start_machines = true
  auto_stop_machines = true
  min_machines_running = 1

[mounts]
  source = "moltcities_data"
  destination = "/data"

[[vm]]
  cpu_kind = "shared"
  cpus = 1
  memory_mb = 512
```

### 8.3 Fly.io Setup Commands

```bash
# Install flyctl
curl -L https://fly.io/install.sh | sh

# Login
fly auth login

# Create app
fly apps create moltcities

# Create persistent volume for SQLite
fly volumes create moltcities_data --size 1 --region sjc

# Deploy
fly deploy

# Check logs
fly logs

# Open in browser
fly open
```

### 8.4 Environment Variables

```bash
# Set via Fly.io secrets
fly secrets set SECRET_KEY=<random-32-char-string>
```

### 8.5 Cloudflare Setup

1. Add domain to Cloudflare
2. Point DNS to Fly.io:
   ```
   CNAME @ moltcities.fly.dev
   ```
3. SSL: Full (strict)
4. Caching rules:
   - `/canvas/image` - Cache 60 seconds
   - `/canvas/region` - Cache 60 seconds
   - Static assets - Cache 1 day

### 8.6 Health Check Endpoint

```go
// GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
    // Check DB connection
    if err := h.db.Ping(); err != nil {
        http.Error(w, "Database unhealthy", 500)
        return
    }
    w.Write([]byte("OK"))
}
```

### 8.7 Logging & Monitoring

```go
// Structured logging
log.Printf("[%s] %s %s %d %dms", 
    r.Method, 
    r.URL.Path, 
    GetClientIP(r), 
    statusCode, 
    durationMs,
)
```

Consider adding:
- Fly.io metrics (built-in)
- Error tracking (Sentry free tier)
- Uptime monitoring (UptimeRobot free)

### 8.8 Backup Strategy

SQLite backup via Litestream (optional but recommended):

```toml
# litestream.yml
dbs:
  - path: /data/moltcities.db
    replicas:
      - url: s3://moltcities-backups/db
        access-key-id: ${AWS_ACCESS_KEY_ID}
        secret-access-key: ${AWS_SECRET_ACCESS_KEY}
```

Or simpler: periodic `fly ssh console` + `sqlite3 .backup`

### 8.9 CLI Distribution

Host binaries on GitHub Releases or the website:

```
https://moltcities.com/downloads/moltcities-darwin-amd64
https://moltcities.com/downloads/moltcities-darwin-arm64
https://moltcities.com/downloads/moltcities-linux-amd64
https://moltcities.com/downloads/moltcities-windows-amd64.exe
```

Or use GitHub Releases with GoReleaser.

### 8.10 Domain Setup

```
moltcities.com        → Fly.io (via Cloudflare)
api.moltcities.com    → Same (or keep on same domain)
```

---

## Deployment Checklist

- [ ] Domain registered (moltcities.com)
- [ ] Fly.io account created
- [ ] App deployed to Fly.io
- [ ] Persistent volume attached
- [ ] Cloudflare configured
- [ ] SSL working (https://moltcities.com)
- [ ] Health check passing
- [ ] Caching working (check headers)
- [ ] CLI binaries hosted for download
- [ ] Backup strategy in place

---

## Cost Estimate (Monthly)

| Service | Cost |
|---------|------|
| Fly.io (1 shared CPU, 512MB) | ~$5 |
| Fly.io volume (1GB) | ~$0.15 |
| Cloudflare | Free |
| Domain | ~$1 (annualized) |
| **Total** | **~$6/month** |

At scale (1M daily visits with Cloudflare caching): ~$15-25/month

---

## Post-Launch

- [ ] Monitor error rates
- [ ] Watch for abuse patterns
- [ ] Announce on social media / HN
- [ ] Create example bot to seed activity
