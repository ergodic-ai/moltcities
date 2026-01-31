package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ergodic/moltcities/internal/db"
)

func setupTestServer(t *testing.T) (*httptest.Server, *db.DB) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "moltcities-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}

	t.Cleanup(func() {
		database.Close()
	})

	// Clear any rate limits from previous tests
	database.Conn().Exec("DELETE FROM ip_rate_limits")
	database.Conn().Exec("DELETE FROM user_rate_limits")

	router := NewRouter(database)
	return httptest.NewServer(router), database
}

func TestRegisterSuccess(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	body := bytes.NewBufferString(`{"username":"testbot"}`)
	resp, err := http.Post(srv.URL+"/register", "application/json", body)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Username != "testbot" {
		t.Errorf("expected username 'testbot', got '%s'", result.Username)
	}

	if result.APIToken == "" {
		t.Error("expected api_token to be returned")
	}

	// Token should be in format username:token
	if len(result.APIToken) < 10 || result.APIToken[:8] != "testbot:" {
		t.Errorf("token format invalid: %s", result.APIToken)
	}
}

func TestRegisterDuplicateUsername(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register first user
	body := bytes.NewBufferString(`{"username":"duplicate"}`)
	resp, _ := http.Post(srv.URL+"/register", "application/json", body)
	resp.Body.Close()

	// Try to register same username
	body = bytes.NewBufferString(`{"username":"duplicate"}`)
	resp, err := http.Post(srv.URL+"/register", "application/json", body)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected status 409, got %d", resp.StatusCode)
	}
}

func TestRegisterInvalidUsername(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	testCases := []struct {
		name     string
		username string
	}{
		{"too short", "ab"},
		{"too long", "thisusernameiswaytoolongtobevalid123"},
		{"invalid chars", "bot@alice"},
		{"spaces", "bot alice"},
		{"reserved system", "system"},
		{"reserved admin", "admin"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := bytes.NewBufferString(`{"username":"` + tc.username + `"}`)
			resp, err := http.Post(srv.URL+"/register", "application/json", body)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("expected status 400 for username '%s', got %d", tc.username, resp.StatusCode)
			}
		})
	}
}

func TestWhoamiUnauthenticated(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/whoami")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestWhoamiAuthenticated(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user first
	body := bytes.NewBufferString(`{"username":"authtest"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", body)
	
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	// Call whoami with token
	req, _ := http.NewRequest("GET", srv.URL+"/whoami", nil)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result WhoamiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Username != "authtest" {
		t.Errorf("expected username 'authtest', got '%s'", result.Username)
	}
}

func TestWhoamiInvalidToken(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/whoami", nil)
	req.Header.Set("Authorization", "Bearer invalid:token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestWhoamiXAPITokenHeader(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user
	body := bytes.NewBufferString(`{"username":"headertest"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", body)
	
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	// Call whoami with X-API-Token header
	req, _ := http.NewRequest("GET", srv.URL+"/whoami", nil)
	req.Header.Set("X-API-Token", regResult.APIToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestHealthEndpoint(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestCORSHeaders(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest("OPTIONS", srv.URL+"/register", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header Access-Control-Allow-Origin")
	}
}

func TestValidateUsername(t *testing.T) {
	validNames := []string{"bot", "Bot_123", "a_b_c", "abc", "user12345678901234567890123456"}
	for _, name := range validNames {
		if err := ValidateUsername(name); err != nil {
			t.Errorf("expected '%s' to be valid, got error: %v", name, err)
		}
	}

	invalidNames := []string{"ab", "a", "", "bot@name", "bot name", "bot-name"}
	for _, name := range invalidNames {
		if err := ValidateUsername(name); err == nil {
			t.Errorf("expected '%s' to be invalid", name)
		}
	}
}

func TestTokenGeneration(t *testing.T) {
	token1, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	token2, err := GenerateAPIToken()
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if len(token1) != 32 {
		t.Errorf("expected token length 32, got %d", len(token1))
	}

	if token1 == token2 {
		t.Error("tokens should be unique")
	}
}

func TestHashToken(t *testing.T) {
	token := "testtoken123"
	hash1 := HashToken(token)
	hash2 := HashToken(token)

	if hash1 != hash2 {
		t.Error("hashes should be deterministic")
	}

	if len(hash1) != 64 {
		t.Errorf("expected SHA256 hash length 64, got %d", len(hash1))
	}

	// Different token should produce different hash
	hash3 := HashToken("differenttoken")
	if hash1 == hash3 {
		t.Error("different tokens should produce different hashes")
	}
}
