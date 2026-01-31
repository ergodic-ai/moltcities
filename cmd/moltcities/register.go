package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register <username>",
	Short: "Register a new account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]

		cfg, err := LoadConfig()
		if err != nil {
			return err
		}

		client := NewClient(cfg)

		resp, err := client.Post("/register", map[string]string{
			"username": username,
		})
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 201 {
			return HandleError(resp)
		}

		var result struct {
			Username string `json:"username"`
			APIToken string `json:"api_token"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Save config
		cfg.Username = result.Username
		cfg.APIToken = result.APIToken
		if err := SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Registered as %s\n", result.Username)
		fmt.Printf("  Config saved to %s\n", getConfigPath())
		return nil
	},
}

var loginCmd = &cobra.Command{
	Use:   "login <username> <api_token>",
	Short: "Login with existing credentials",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		token := args[1]

		cfg, err := LoadConfig()
		if err != nil {
			return err
		}

		cfg.Username = username
		cfg.APIToken = token

		// Verify credentials by calling whoami
		client := NewClient(cfg)
		resp, err := client.Get("/whoami")
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("invalid credentials")
		}

		if err := SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("✓ Logged in as %s\n", username)
		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user information",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}

		if err := RequireAuth(cfg); err != nil {
			return err
		}

		client := NewClient(cfg)
		resp, err := client.Get("/whoami")
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return HandleError(resp)
		}

		var result struct {
			Username   string  `json:"username"`
			CreatedAt  string  `json:"created_at"`
			LastEditAt *string `json:"last_edit_at"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("Username:   %s\n", result.Username)
		fmt.Printf("Registered: %s\n", result.CreatedAt)
		if result.LastEditAt != nil {
			fmt.Printf("Last edit:  %s\n", *result.LastEditAt)
		} else {
			fmt.Printf("Last edit:  never\n")
		}
		return nil
	},
}
