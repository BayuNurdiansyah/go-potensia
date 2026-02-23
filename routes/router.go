package routes

import (
	"go-potensia/controllers"
	"go-potensia/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	auth := r.Group("/auth")
	{
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)
		auth.POST("/verify-otp", controllers.VerifyOTP) 
	}

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