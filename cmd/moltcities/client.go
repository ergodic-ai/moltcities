package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the MoltCities API.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient creates a new API client.
func NewClient(cfg *Config) *Client {
	return &Client{
		baseURL: cfg.APIBaseURL,
		token:   cfg.APIToken,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get performs a GET request.
func (c *Client) Get(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	c.addHeaders(req)
	return c.http.Do(req)
}

// Post performs a POST request with JSON body.
func (c *Client) Post(path string, body interface{}) (*http.Response, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest("POST", c.baseURL+path, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addHeaders(req)
	return c.http.Do(req)
}

// addHeaders adds authentication headers.
func (c *Client) addHeaders(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

// ErrorResponse is the API error format.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details string `json:"details"`
}

// HandleError extracts and formats an API error.
func HandleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		if errResp.Details != "" {
			return fmt.Errorf("%s: %s", errResp.Error, errResp.Details)
		}
		return fmt.Errorf("%s", errResp.Error)
	}

	return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
}
