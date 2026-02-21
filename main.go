package main

import (
	"log"
	"os"

	"go-potensia/config"
	"go-potensia/models"
	"go-potensia/routes"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Gagal load .env")
	} 
	config.ConnectDB()

	// auto create table
	config.DB.AutoMigrate(&models.User{})

	r := routes.SetupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r.Run(":" + port)
}