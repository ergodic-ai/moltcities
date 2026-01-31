// Package canvas handles canvas rendering operations.
package canvas

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"

	"github.com/ergodic/moltcities/internal/models"
)

// Render generates a PNG image of the canvas.
func Render(pixels map[[2]int]string, w io.Writer) error {
	img := image.NewRGBA(image.Rect(0, 0, models.CanvasSize, models.CanvasSize))

	// Fill with white
	white := color.RGBA{255, 255, 255, 255}
	for y := 0; y < models.CanvasSize; y++ {
		for x := 0; x < models.CanvasSize; x++ {
			img.Set(x, y, white)
		}
	}

	// Apply edited pixels
	for coord, hex := range pixels {
		c, err := HexToColor(hex)
		if err != nil {
			continue // Skip invalid colors
		}
		img.Set(coord[0], coord[1], c)
	}

	return png.Encode(w, img)
}

// RenderRegion generates a PNG image of a canvas region.
func RenderRegion(pixels [][]string, w io.Writer) error {
	height := len(pixels)
	if height == 0 {
		return fmt.Errorf("empty region")
	}
	width := len(pixels[0])

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y, row := range pixels {
		for x, hex := range row {
			c, err := HexToColor(hex)
			if err != nil {
				c = color.RGBA{255, 255, 255, 255} // Default to white
			}
			img.Set(x, y, c)
		}
	}

	return png.Encode(w, img)
}

// HexToColor converts a hex color string to color.RGBA.
// Accepts formats: "#RRGGBB" or "RRGGBB"
func HexToColor(hex string) (color.RGBA, error) {
	// Remove # prefix if present
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}

	if len(hex) != 6 {
		return color.RGBA{}, fmt.Errorf("invalid hex color: %s", hex)
	}

	var r, g, b uint8
	_, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid hex color: %s", hex)
	}

	return color.RGBA{r, g, b, 255}, nil
}

// ValidateColor checks if a color string is valid.
func ValidateColor(hex string) error {
	_, err := HexToColor(hex)
	return err
}

// ValidateCoordinate checks if a coordinate is within canvas bounds.
func ValidateCoordinate(n int) error {
	if n < 0 || n >= models.CanvasSize {
		return fmt.Errorf("coordinate must be between 0 and %d", models.CanvasSize-1)
	}
	return nil
}

// ValidateRegion checks if a region is valid.
func ValidateRegion(x, y, width, height int) error {
	if err := ValidateCoordinate(x); err != nil {
		return fmt.Errorf("x: %w", err)
	}
	if err := ValidateCoordinate(y); err != nil {
		return fmt.Errorf("y: %w", err)
	}
	if width < 1 || width > models.MaxRegionSize {
		return fmt.Errorf("width must be between 1 and %d", models.MaxRegionSize)
	}
	if height < 1 || height > models.MaxRegionSize {
		return fmt.Errorf("height must be between 1 and %d", models.MaxRegionSize)
	}
	if x+width > models.CanvasSize {
		return fmt.Errorf("region extends beyond canvas width")
	}
	if y+height > models.CanvasSize {
		return fmt.Errorf("region extends beyond canvas height")
	}
	return nil
}
