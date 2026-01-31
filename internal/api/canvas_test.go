package api

import (
	"bytes"
	"encoding/json"
	"image/png"
	"net/http"
	"testing"

	"github.com/ergodic/moltcities/internal/models"
)

func TestGetCanvasImage(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/canvas/image")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "image/png" {
		t.Errorf("expected Content-Type image/png, got %s", resp.Header.Get("Content-Type"))
	}

	// Verify it's a valid PNG
	img, err := png.Decode(resp.Body)
	if err != nil {
		t.Fatalf("response is not a valid PNG: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != models.CanvasSize || bounds.Dy() != models.CanvasSize {
		t.Errorf("expected %dx%d image, got %dx%d", models.CanvasSize, models.CanvasSize, bounds.Dx(), bounds.Dy())
	}
}

func TestGetCanvasRegion(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/canvas/region?x=0&y=0&width=64&height=64")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result models.RegionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Width != 64 || result.Height != 64 {
		t.Errorf("expected 64x64, got %dx%d", result.Width, result.Height)
	}

	if len(result.Pixels) != 64 {
		t.Errorf("expected 64 rows, got %d", len(result.Pixels))
	}

	if len(result.Pixels[0]) != 64 {
		t.Errorf("expected 64 columns, got %d", len(result.Pixels[0]))
	}

	// All pixels should be white (unedited)
	if result.Pixels[0][0] != "#FFFFFF" {
		t.Errorf("expected white pixel, got %s", result.Pixels[0][0])
	}
}

func TestGetCanvasRegionTooLarge(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/canvas/region?x=0&y=0&width=256&height=256")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for region > 128x128, got %d", resp.StatusCode)
	}
}

func TestGetCanvasRegionOutOfBounds(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/canvas/region?x=1000&y=1000&width=128&height=128")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400 for out of bounds, got %d", resp.StatusCode)
	}
}

func TestGetPixelUnedited(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/pixel?x=100&y=200")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var pixel models.Pixel
	if err := json.NewDecoder(resp.Body).Decode(&pixel); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if pixel.X != 100 || pixel.Y != 200 {
		t.Errorf("expected (100, 200), got (%d, %d)", pixel.X, pixel.Y)
	}

	if pixel.Color != "#FFFFFF" {
		t.Errorf("expected white, got %s", pixel.Color)
	}

	if pixel.EditedBy != nil {
		t.Error("expected nil edited_by for unedited pixel")
	}
}

func TestGetPixelInvalidCoordinate(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	testCases := []struct {
		name string
		url  string
	}{
		{"negative x", "/pixel?x=-1&y=0"},
		{"negative y", "/pixel?x=0&y=-1"},
		{"x too large", "/pixel?x=1024&y=0"},
		{"y too large", "/pixel?x=0&y=1024"},
		{"missing x", "/pixel?y=0"},
		{"missing y", "/pixel?x=0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(srv.URL + tc.url)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", resp.StatusCode)
			}
		})
	}
}

func TestEditPixelUnauthenticated(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	body := bytes.NewBufferString(`{"x":100,"y":200,"color":"#FF5733"}`)
	resp, err := http.Post(srv.URL+"/pixel", "application/json", body)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestEditPixelSuccess(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user
	regBody := bytes.NewBufferString(`{"username":"edituser"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", regBody)
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	// Edit pixel
	body := bytes.NewBufferString(`{"x":100,"y":200,"color":"#FF5733"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/pixel", body)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result EditPixelResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}

	if result.NextEditAt == nil {
		t.Error("expected next_edit_at to be set")
	}

	// Verify pixel was updated
	pixelResp, _ := http.Get(srv.URL + "/pixel?x=100&y=200")
	var pixel models.Pixel
	json.NewDecoder(pixelResp.Body).Decode(&pixel)
	pixelResp.Body.Close()

	if pixel.Color != "#FF5733" {
		t.Errorf("expected color #FF5733, got %s", pixel.Color)
	}

	if pixel.EditedBy == nil || *pixel.EditedBy != "edituser" {
		t.Error("expected edited_by to be 'edituser'")
	}
}

func TestEditPixelRateLimit(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user
	regBody := bytes.NewBufferString(`{"username":"ratelimituser"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", regBody)
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	// First edit should succeed
	body := bytes.NewBufferString(`{"x":50,"y":50,"color":"#FF0000"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/pixel", body)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("first edit should succeed, got %d", resp.StatusCode)
	}

	// Second edit should be rate limited
	body = bytes.NewBufferString(`{"x":51,"y":51,"color":"#00FF00"}`)
	req, _ = http.NewRequest("POST", srv.URL+"/pixel", body)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("second edit should be rate limited, got %d", resp.StatusCode)
	}
}

func TestEditPixelInvalidColor(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user
	regBody := bytes.NewBufferString(`{"username":"coloruser"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", regBody)
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	testCases := []string{"invalid", "#GGG", "#12345", "red", ""}

	for _, color := range testCases {
		body := bytes.NewBufferString(`{"x":100,"y":200,"color":"` + color + `"}`)
		req, _ := http.NewRequest("POST", srv.URL+"/pixel", body)
		req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
		req.Header.Set("Content-Type", "application/json")
		resp, _ := http.DefaultClient.Do(req)
		resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for color '%s', got %d", color, resp.StatusCode)
		}
	}
}

func TestGetPixelHistory(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user and edit a pixel
	regBody := bytes.NewBufferString(`{"username":"historyuser"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", regBody)
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	body := bytes.NewBufferString(`{"x":300,"y":400,"color":"#AABBCC"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/pixel", body)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
	req.Header.Set("Content-Type", "application/json")
	http.DefaultClient.Do(req)

	// Get history
	resp, err := http.Get(srv.URL + "/pixel/history?x=300&y=400")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result struct {
		X       int           `json:"x"`
		Y       int           `json:"y"`
		History []models.Edit `json:"history"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result.History) != 1 {
		t.Errorf("expected 1 edit in history, got %d", len(result.History))
	}

	if result.History[0].Color != "#AABBCC" {
		t.Errorf("expected color #AABBCC, got %s", result.History[0].Color)
	}

	if result.History[0].Username != "historyuser" {
		t.Errorf("expected username 'historyuser', got %s", result.History[0].Username)
	}
}

func TestGetStats(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/stats")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var stats models.Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have 1 channel (general)
	if stats.TotalChannels != 1 {
		t.Errorf("expected 1 channel, got %d", stats.TotalChannels)
	}
}

func TestCanvasRegionShowsEditedPixels(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user and edit a pixel
	regBody := bytes.NewBufferString(`{"username":"regionuser"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", regBody)
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	body := bytes.NewBufferString(`{"x":10,"y":20,"color":"#123456"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/pixel", body)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
	req.Header.Set("Content-Type", "application/json")
	http.DefaultClient.Do(req)

	// Get region containing the edited pixel
	resp, err := http.Get(srv.URL + "/canvas/region?x=0&y=0&width=128&height=128")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result models.RegionResponse
	json.NewDecoder(resp.Body).Decode(&result)

	// Pixel at (10, 20) should be #123456
	if result.Pixels[20][10] != "#123456" {
		t.Errorf("expected #123456 at (10,20), got %s", result.Pixels[20][10])
	}
}
