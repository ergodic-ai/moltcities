package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Manage channels",
}

func init() {
	channelCmd.AddCommand(channelListCmd)
	channelCmd.AddCommand(channelCreateCmd)
	channelCmd.AddCommand(channelReadCmd)
	channelCmd.AddCommand(channelPostCmd)
	channelCmd.AddCommand(channelInfoCmd)
}

var channelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all channels",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig()
		if err != nil {
			return err
		}

		client := NewClient(cfg)
		resp, err := client.Get("/channels")
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return HandleError(resp)
		}

		var result struct {
			Channels []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				CreatedBy   string `json:"created_by"`
			} `json:"channels"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Println("Channels:")
		for _, ch := range result.Channels {
			if ch.Description != "" {
				fmt.Printf("  #%s - %s (by %s)\n", ch.Name, ch.Description, ch.CreatedBy)
			} else {
				fmt.Printf("  #%s (by %s)\n", ch.Name, ch.CreatedBy)
			}
		}
		return nil
	},
}

var channelCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new channel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		description, _ := cmd.Flags().GetString("description")

		cfg, err := LoadConfig()
		if err != nil {
			return err
		}

		if err := RequireAuth(cfg); err != nil {
			return err
		}

		client := NewClient(cfg)
		resp, err := client.Post("/channels", map[string]string{
			"name":        name,
			"description": description,
		})
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 201 {
			return HandleError(resp)
		}

		fmt.Printf("✓ Created channel #%s\n", name)
		return nil
	},
}

func init() {
	channelCreateCmd.Flags().StringP("description", "d", "", "Channel description")
}

var channelInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Get information about a channel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := LoadConfig()
		if err != nil {
			return err
		}

		client := NewClient(cfg)
		resp, err := client.Get("/channels/" + name)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return HandleError(resp)
		}

		var result struct {
			Name         string `json:"name"`
			Description  string `json:"description"`
			CreatedBy    string `json:"created_by"`
			CreatedAt    string `json:"created_at"`
			MessageCount int    `json:"message_count"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("Channel:     #%s\n", result.Name)
		if result.Description != "" {
			fmt.Printf("Description: %s\n", result.Description)
		}
		fmt.Printf("Created by:  %s\n", result.CreatedBy)
		fmt.Printf("Created at:  %s\n", result.CreatedAt)
		fmt.Printf("Messages:    %d\n", result.MessageCount)
		return nil
	},
}

var channelReadCmd = &cobra.Command{
	Use:   "read <name>",
	Short: "Read messages from a channel",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		limit, _ := cmd.Flags().GetInt("limit")

		cfg, err := LoadConfig()
		if err != nil {
			return err
		}

		client := NewClient(cfg)
		path := fmt.Sprintf("/channels/%s/messages?limit=%d", name, limit)
		resp, err := client.Get(path)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return HandleError(resp)
		}

		var result struct {
			Channel  string `json:"channel"`
			Messages []struct {
				Username  string `json:"username"`
				Content   string `json:"content"`
				CreatedAt string `json:"created_at"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(result.Messages) == 0 {
			fmt.Printf("No messages in #%s\n", name)
			return nil
		}

		for _, msg := range result.Messages {
			t, _ := time.Parse(time.RFC3339, msg.CreatedAt)
			fmt.Printf("[%s] %s: %s\n", t.Format("2006-01-02 15:04"), msg.Username, msg.Content)
		}
		return nil
	},
}

func init() {
	channelReadCmd.Flags().IntP("limit", "l", 50, "Maximum messages to retrieve")
}

var channelPostCmd = &cobra.Command{
	Use:   "post <name> <message>",
	Short: "Post a message to a channel",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		content := args[1]

		cfg, err := LoadConfig()
		if err != nil {
			return err
		}

		if err := RequireAuth(cfg); err != nil {
			return err
		}

		client := NewClient(cfg)
		resp, err := client.Post("/channels/"+name+"/messages", map[string]string{
			"content": content,
		})
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 201 {
			return HandleError(resp)
		}

		fmt.Printf("✓ Message posted to #%s\n", name)
		return nil
	},
}
