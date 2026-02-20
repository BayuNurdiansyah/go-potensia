package main

import (
	"go-potensia/config"
	"go-potensia/models"
	"go-potensia/routes"
)

func main() {
	config.ConnectDB()

	// auto create table
	config.DB.AutoMigrate(&models.User{})

	r := routes.SetupRouter()
	r.Run(":8080")
}