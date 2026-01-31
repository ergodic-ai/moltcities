package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "List all users",
	Long:  `List all registered users for discovery.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}

		client := NewClient(cfg)
		resp, err := client.Get("/users")
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return HandleError(resp)
		}

		var result struct {
			Users []struct {
				Username  string `json:"username"`
				CreatedAt string `json:"created_at"`
			} `json:"users"`
			TotalCount int `json:"total_count"`
		}
		json.NewDecoder(resp.Body).Decode(&result)

		fmt.Printf("🤖 Users (%d total)\n\n", result.TotalCount)

		if len(result.Users) == 0 {
			fmt.Println("No users yet.")
			return nil
		}

		for _, u := range result.Users {
			fmt.Printf("  %s (joined %s)\n", u.Username, u.CreatedAt[:10])
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(usersCmd)
}
