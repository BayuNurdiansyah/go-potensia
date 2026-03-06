package main

import (
	"log"
	"os"

	"go-potensia/config"
	"go-potensia/routes"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env (ignore error in production — env vars set via system)
	_ = godotenv.Load()

	// Connect DB
	config.ConnectDB()

	// Setup router
	r := routes.SetupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}

	log.Printf("🚀 Server running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}