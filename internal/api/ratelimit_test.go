package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestIPRateLimitRegistration(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Should be able to register 5 users from same IP
	for i := 0; i < 5; i++ {
		body := bytes.NewBufferString(`{"username":"ratelimituser` + string(rune('a'+i)) + `"}`)
		resp, err := http.Post(srv.URL+"/register", "application/json", body)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("registration %d should succeed, got %d", i, resp.StatusCode)
		}
	}

	// 6th registration should be rate limited
	body := bytes.NewBufferString(`{"username":"ratelimituserf"}`)
	resp, err := http.Post(srv.URL+"/register", "application/json", body)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("6th registration should be rate limited, got %d", resp.StatusCode)
	}
}

func TestChannelCreationRateLimit(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user
	regBody := bytes.NewBufferString(`{"username":"channelratelimit"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", regBody)
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	// Should be able to create 3 channels
	for i := 0; i < 3; i++ {
		body := bytes.NewBufferString(`{"name":"channel-` + string(rune('a'+i)) + `"}`)
		req, _ := http.NewRequest("POST", srv.URL+"/channels", body)
		req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
		req.Header.Set("Content-Type", "application/json")
		resp, _ := http.DefaultClient.Do(req)
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("channel %d should succeed, got %d", i, resp.StatusCode)
		}
	}

	// 4th channel should be rate limited
	body := bytes.NewBufferString(`{"name":"channel-d"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/channels", body)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("4th channel should be rate limited, got %d", resp.StatusCode)
	}
}

func TestMessageRateLimit(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.Close()

	// Register user
	regBody := bytes.NewBufferString(`{"username":"msgratelimit"}`)
	regResp, _ := http.Post(srv.URL+"/register", "application/json", regBody)
	var regResult RegisterResponse
	json.NewDecoder(regResp.Body).Decode(&regResult)
	regResp.Body.Close()

	// Should be able to post 10 messages
	for i := 0; i < 10; i++ {
		body := bytes.NewBufferString(`{"content":"Message ` + string(rune('0'+i)) + `"}`)
		req, _ := http.NewRequest("POST", srv.URL+"/channels/general/messages", body)
		req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
		req.Header.Set("Content-Type", "application/json")
		resp, _ := http.DefaultClient.Do(req)
		resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("message %d should succeed, got %d", i, resp.StatusCode)
		}
	}

	// 11th message should be rate limited
	body := bytes.NewBufferString(`{"content":"Rate limited message"}`)
	req, _ := http.NewRequest("POST", srv.URL+"/channels/general/messages", body)
	req.Header.Set("Authorization", "Bearer "+regResult.APIToken)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("11th message should be rate limited, got %d", resp.StatusCode)
	}
}
