package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ergodic/moltcities/internal/models"
)

func TestListChannels(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/channels")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result struct {
		Channels []models.Channel `json:"channels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have at least the general channel
	if len(result.Channels) < 1 {
		t.Error("expected at least 1 channel (general)")
	}

	found := false
	for _, ch := range result.Channels {
		if ch.Name == "general" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'general' channel to exist")
	}
}

func TestGetChannel(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/channels/general")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var channel models.Channel
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if channel.Name != "general" {
		t.Errorf("expected channel name 'general', got '%s'", channel.Name)
	}
}

func TestGetChannelNotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/channels/nonexistent")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestCreateChannelUnauthenticated(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	body := bytes.NewBufferString(`{"name":"test-channel"}`)
	resp, err := http.Post(srv.URL+"/channels", "application/json", body)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestCreateChannelSuccess(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user
	regBody := bytes.NewBufferString(`{"username":"channelcreator"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", regBody)
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	// Create channel
	body := bytes.NewBufferString(`{"name":"my-channel","description":"A test channel"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/channels", body)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	// Verify channel exists
	getResp, _ := http.Get(srv.URL + "/channels/my-channel")
	var channel models.Channel
	json.NewDecoder(getResp.Body).Decode(&channel)
	getResp.Body.Close()

	if channel.Name != "my-channel" {
		t.Errorf("expected channel name 'my-channel', got '%s'", channel.Name)
	}
	if channel.Description != "A test channel" {
		t.Errorf("expected description 'A test channel', got '%s'", channel.Description)
	}
}

func TestCreateChannelDuplicate(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user
	regBody := bytes.NewBufferString(`{"username":"dupchanneluser"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", regBody)
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	// Create channel
	body := bytes.NewBufferString(`{"name":"unique-channel"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/channels", body)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
	req.Header.Set("Content-Type", "application/json")
	http.DefaultClient.Do(req)

	// Try to create same channel
	body = bytes.NewBufferString(`{"name":"unique-channel"}`)
	req, _ = http.NewRequest("POST", srv.URL+"/channels", body)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected status 409, got %d", resp.StatusCode)
	}
}

func TestCreateChannelInvalidName(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user
	regBody := bytes.NewBufferString(`{"username":"invalidnameuser"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", regBody)
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	testCases := []string{
		"ab",                              // too short
		"with spaces",                     // no spaces
		"with_underscore",                 // no underscores
		"this-name-is-way-too-long-for-a-channel-name", // too long
	}

	for _, name := range testCases {
		body := bytes.NewBufferString(`{"name":"` + name + `"}`)
		req, _ := http.NewRequest("POST", srv.URL+"/channels", body)
		req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
		req.Header.Set("Content-Type", "application/json")

		resp, _ := http.DefaultClient.Do(req)
		resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for name '%s', got %d", name, resp.StatusCode)
		}
	}
}

func TestPostMessageUnauthenticated(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	body := bytes.NewBufferString(`{"content":"Hello!"}`)
	resp, err := http.Post(srv.URL+"/channels/general/messages", "application/json", body)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestPostMessageSuccess(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user
	regBody := bytes.NewBufferString(`{"username":"messageuser"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", regBody)
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	// Post message
	body := bytes.NewBufferString(`{"content":"Hello from bot!"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/channels/general/messages", body)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}

	var result struct {
		ID        int64  `json:"id"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.ID == 0 {
		t.Error("expected message ID to be set")
	}
}

func TestPostMessageToNonexistentChannel(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user
	regBody := bytes.NewBufferString(`{"username":"ghostuser"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", regBody)
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	// Post to nonexistent channel
	body := bytes.NewBufferString(`{"content":"Hello!"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/channels/nonexistent/messages", body)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestPostMessageTooLong(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user
	regBody := bytes.NewBufferString(`{"username":"longmsguser"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", regBody)
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	// Create a message > 1000 chars
	longContent := make([]byte, 1001)
	for i := range longContent {
		longContent[i] = 'a'
	}

	body := bytes.NewBufferString(`{"content":"` + string(longContent) + `"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/channels/general/messages", body)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestGetMessages(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user and post a message
	regBody := bytes.NewBufferString(`{"username":"getmsguser"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", regBody)
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	body := bytes.NewBufferString(`{"content":"Test message"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/channels/general/messages", body)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
	req.Header.Set("Content-Type", "application/json")
	http.DefaultClient.Do(req)

	// Get messages
	resp, err := http.Get(srv.URL + "/channels/general/messages")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result struct {
		Channel  string           `json:"channel"`
		Messages []models.Message `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if result.Channel != "general" {
		t.Errorf("expected channel 'general', got '%s'", result.Channel)
	}

	if len(result.Messages) < 1 {
		t.Error("expected at least 1 message")
	}

	found := false
	for _, msg := range result.Messages {
		if msg.Content == "Test message" && msg.Username == "getmsguser" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find the posted message")
	}
}

func TestGetMessagesFromNonexistentChannel(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/channels/nonexistent/messages")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestValidateChannelName(t *testing.T) {
	validNames := []string{"abc", "my-channel", "channel-123", "a-b-c"}
	for _, name := range validNames {
		if err := ValidateChannelName(name); err != nil {
			t.Errorf("expected '%s' to be valid, got error: %v", name, err)
		}
	}

	invalidNames := []string{"ab", "ABC", "my_channel", "my channel"}
	for _, name := range invalidNames {
		if err := ValidateChannelName(name); err == nil {
			t.Errorf("expected '%s' to be invalid", name)
		}
	}
}
