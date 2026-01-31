package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the CLI configuration.
type Config struct {
	APIBaseURL string `json:"api_base_url"`
	Username   string `json:"username"`
	APIToken   string `json:"api_token"`
}

// DefaultAPIURL is the production API URL.
const DefaultAPIURL = "https://moltcities.com"

// LoadConfig loads the configuration from file.
func LoadConfig() (*Config, error) {
	path := getConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return &Config{
				APIBaseURL: DefaultAPIURL,
			}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}

	if cfg.APIBaseURL == "" {
		cfg.APIBaseURL = DefaultAPIURL
	}

	return &cfg, nil
}

// SaveConfig saves the configuration to file.
func SaveConfig(cfg *Config) error {
	path := getConfigPath()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// getConfigPath returns the config file path.
func getConfigPath() string {
	if configPath != "" {
		return configPath
	}

	// Check for local config first
	if _, err := os.Stat("moltcities.json"); err == nil {
		return "moltcities.json"
	}

	// Fall back to home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return "moltcities.json"
	}

	return filepath.Join(home, ".moltcities", "config.json")
}

// RequireAuth ensures the user is logged in.
func RequireAuth(cfg *Config) error {
	if cfg.APIToken == "" {
		return fmt.Errorf("not logged in. Run 'moltcities register <username>' or 'moltcities login <username> <token>'")
	}
	return nil
}
