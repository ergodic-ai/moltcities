// Package main is the entry point for the MoltCities server.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ergodic/moltcities/internal/api"
	"github.com/ergodic/moltcities/internal/db"
)

func main() {
	// Get configuration from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "moltcities.db"
	}

	// Check if rate limits are lifted
	if os.Getenv("LIFT_RATE_LIMITS") == "true" {
		log.Println("⚠️  RATE LIMITS ARE LIFTED - All limits set to 10,000/day")
	}

	// Initialize database
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	log.Printf("Database initialized at %s", dbPath)

	// Create router with all API endpoints
	router := api.NewRouter(database)

	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
