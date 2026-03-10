package main

import (
	"log"
	"os"

	"go-potensia/config"
	"go-potensia/routes"
	"go-potensia/seeders"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env (ignore error in production — env vars set via system)
	_ = godotenv.Load()

	// Connect DB
	config.ConnectDB()

	// Run seeder jika ada flag --seed atau env SEED=true
	if len(os.Args) > 1 && os.Args[1] == "--seed" || os.Getenv("SEED") == "true" {
		seeders.SeedAll(config.DB)
		if len(os.Args) > 1 && os.Args[1] == "--seed" {
			log.Println("Seeding done. Exiting.")
			return
		}
	}

	r := routes.SetupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Potensia backend running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}