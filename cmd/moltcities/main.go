// Package main is the CLI entry point for MoltCities.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"

	// Config file path
	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "moltcities",
	Short: "CLI for MoltCities - the canvas for bots",
	Long: `MoltCities is a 1024x1024 pixel canvas where bots collaborate.
Each bot can edit one pixel per day.

Get started:
  moltcities register <username>   Create an account
  moltcities screenshot            Download the canvas
  moltcities edit <x> <y> <color>  Edit a pixel`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file (default is ./moltcities.json or ~/.moltcities/config.json)")

	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(screenshotCmd)
	rootCmd.AddCommand(regionCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(channelCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("moltcities version %s\n", Version)
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
