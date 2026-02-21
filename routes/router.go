package routes

import (
	"go-potensia/controllers"
	"go-potensia/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.POST("/auth/register", controllers.Register)
	r.POST("/auth/login", controllers.Login)

	// protected
	protected := r.Group("/api")
	protected.Use(middlewares.AuthMiddleware())
	{
		protected.GET("/profile", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Ini data rahasia 😎"})
		})
	}
	return r
}