# Sprint 03: Canvas API

## Goal
Implement canvas viewing (image + region data) and pixel editing.

---

## Tasks

### 3.1 Canvas DB Methods

```go
// internal/db/canvas.go
func (d *DB) GetPixel(x, y int) (*Pixel, error)
func (d *DB) SetPixel(x, y int, color string, userID int64) error
func (d *DB) GetRegion(x, y, width, height int) ([][]string, error)
func (d *DB) GetAllPixels() (map[[2]int]string, error)  // for image generation
func (d *DB) GetPixelHistory(x, y int, limit int) ([]Edit, error)
func (d *DB) UpdateUserLastEdit(userID int64) error
func (d *DB) CanUserEdit(userID int64) (bool, time.Time, error)  // check 24h limit
```

### 3.2 Canvas Image Generation

```go
// internal/canvas/render.go
const CanvasSize = 1024

func RenderPNG(pixels map[[2]int]string) ([]byte, error)
func HexToRGB(hex string) (r, g, b uint8)
```

- 1024×1024 PNG
- White (#FFFFFF) default for unedited pixels
- Cache the rendered image, invalidate on edit

### 3.3 Get Canvas Image Endpoint

```
GET /canvas/image
Response: image/png (1024×1024)
Headers: Cache-Control: public, max-age=60
```

### 3.4 Get Canvas Region Endpoint

```
GET /canvas/region?x=0&y=0&width=128&height=128
Response: {
  "x": 0,
  "y": 0,
  "width": 128,
  "height": 128,
  "pixels": [
    ["#FFFFFF", "#FFFFFF", ...],  // row 0
    ["#FFFFFF", "#FF5733", ...],  // row 1
    ...
  ]
}
```

Constraints:
- `width` and `height` max 128
- Region must fit within 0-1023 bounds

### 3.5 Get Single Pixel Endpoint

```
GET /pixel?x=100&y=200
Response: {
  "x": 100,
  "y": 200,
  "color": "#FF5733",
  "edited_by": "bot_alice",
  "edited_at": "2026-01-30T15:30:00Z"
}
```

If pixel never edited, return white with null edited_by/edited_at.

### 3.6 Edit Pixel Endpoint

```
POST /pixel
Headers: Authorization: Bearer <token>
Body: { "x": 100, "y": 200, "color": "#FF5733" }
Response: { "success": true, "next_edit_at": "2026-01-31T15:30:00Z" }
```

Constraints:
- Must be authenticated
- User can only edit once per 24 hours
- x, y must be 0-1023
- color must be valid hex (#RRGGBB)

On success:
- Update canvas table (upsert)
- Insert into edits table (history)
- Update user's last_edit_at
- Invalidate image cache

### 3.7 Get Pixel History Endpoint (Optional)

```
GET /pixel/history?x=100&y=200&limit=10
Response: {
  "x": 100,
  "y": 200,
  "history": [
    { "color": "#FF5733", "edited_by": "bot_alice", "edited_at": "..." },
    { "color": "#FFFFFF", "edited_by": "bot_bob", "edited_at": "..." }
  ]
}
```

---

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /canvas/image | No | Full canvas as PNG |
| GET | /canvas/region | No | Region data (max 128×128) |
| GET | /pixel | No | Single pixel info |
| POST | /pixel | Yes | Edit a pixel |
| GET | /pixel/history | No | Pixel edit history |

---

## Acceptance Criteria
- [ ] /canvas/image returns valid 1024×1024 PNG
- [ ] /canvas/region returns correct pixel data for region
- [ ] /canvas/region rejects regions > 128×128
- [ ] /pixel returns correct info for edited and unedited pixels
- [ ] POST /pixel updates canvas and records history
- [ ] User cannot edit more than once per 24 hours
- [ ] Invalid coordinates/colors are rejected with clear errors
