package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var screenshotCmd = &cobra.Command{
	Use:   "screenshot [output.png]",
	Short: "Download the canvas as a PNG image",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		output := "canvas.png"
		if len(args) > 0 {
			output = args[0]
		}

		cfg, err := LoadConfig()
		if err != nil {
			return err
		}

		client := NewClient(cfg)
		resp, err := client.Get("/canvas/image")
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return HandleError(resp)
		}

		file, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer file.Close()

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to save image: %w", err)
		}

		fmt.Printf("✓ Saved canvas to %s (1024x1024)\n", output)
		return nil
	},
}

var regionCmd = &cobra.Command{
	Use:   "region <x> <y> <width> <height>",
	Short: "Get pixel data for a region (max 128x128)",
	Long: `Get pixel data for a rectangular region of the canvas.
Maximum region size is 128x128 pixels.

Use --output to save as a PNG file instead of printing JSON.`,
	Args: cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		x, _ := strconv.Atoi(args[0])
		y, _ := strconv.Atoi(args[1])
		width, _ := strconv.Atoi(args[2])
		height, _ := strconv.Atoi(args[3])

		cfg, err := LoadConfig()
		if err != nil {
			return err
		}

		client := NewClient(cfg)
		path := fmt.Sprintf("/canvas/region?x=%d&y=%d&width=%d&height=%d", x, y, width, height)
		resp, err := client.Get(path)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return HandleError(resp)
		}

		var result struct {
			X      int        `json:"x"`
			Y      int        `json:"y"`
			Width  int        `json:"width"`
			Height int        `json:"height"`
			Pixels [][]string `json:"pixels"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		output, _ := cmd.Flags().GetString("output")
		if output != "" {
			// Render to PNG
			img := image.NewRGBA(image.Rect(0, 0, result.Width, result.Height))
			for row, pixelRow := range result.Pixels {
				for col, hex := range pixelRow {
					c := hexToColor(hex)
					img.Set(col, row, c)
				}
			}

			file, err := os.Create(output)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			defer file.Close()

			if err := png.Encode(file, img); err != nil {
				return fmt.Errorf("failed to encode PNG: %w", err)
			}

			fmt.Printf("✓ Saved region to %s (%dx%d)\n", output, result.Width, result.Height)
		} else {
			// Print as JSON
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(result)
		}

		return nil
	},
}

func init() {
	regionCmd.Flags().StringP("output", "o", "", "Save region as PNG file")
}

var getCmd = &cobra.Command{
	Use:   "get <x> <y>",
	Short: "Get information about a single pixel",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		x, _ := strconv.Atoi(args[0])
		y, _ := strconv.Atoi(args[1])

		cfg, err := LoadConfig()
		if err != nil {
			return err
		}

		client := NewClient(cfg)
		path := fmt.Sprintf("/pixel?x=%d&y=%d", x, y)
		resp, err := client.Get(path)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return HandleError(resp)
		}

		var result struct {
			X        int     `json:"x"`
			Y        int     `json:"y"`
			Color    string  `json:"color"`
			EditedBy *string `json:"edited_by"`
			EditedAt *string `json:"edited_at"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if result.EditedBy != nil {
			fmt.Printf("(%d, %d): %s (edited by %s at %s)\n", result.X, result.Y, result.Color, *result.EditedBy, *result.EditedAt)
		} else {
			fmt.Printf("(%d, %d): %s (never edited)\n", result.X, result.Y, result.Color)
		}
		return nil
	},
}

var editCmd = &cobra.Command{
	Use:   "edit <x> <y> <color>",
	Short: "Edit a pixel (requires auth, once per day)",
	Long: `Edit a single pixel on the canvas.
Color must be in hex format: #RRGGBB

You can only edit one pixel per day.`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		x, _ := strconv.Atoi(args[0])
		y, _ := strconv.Atoi(args[1])
		colorHex := args[2]

		cfg, err := LoadConfig()
		if err != nil {
			return err
		}

		if err := RequireAuth(cfg); err != nil {
			return err
		}

		client := NewClient(cfg)
		resp, err := client.Post("/pixel", map[string]interface{}{
			"x":     x,
			"y":     y,
			"color": colorHex,
		})
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return HandleError(resp)
		}

		var result struct {
			Success    bool    `json:"success"`
			NextEditAt *string `json:"next_edit_at"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Printf("✓ Edited (%d, %d) to %s\n", x, y, colorHex)
		if result.NextEditAt != nil {
			fmt.Printf("  Next edit available at: %s\n", *result.NextEditAt)
		}
		return nil
	},
}

func hexToColor(hex string) color.RGBA {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return color.RGBA{255, 255, 255, 255}
	}
	var r, g, b uint8
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return color.RGBA{r, g, b, 255}
}
