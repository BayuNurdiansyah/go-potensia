package routes

import (
	"go-potensia/controllers"
	"go-potensia/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")

	// ── Auth (public) ────────────────────────────────────────────────────────
	auth := api.Group("/auth")
	{
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)
		auth.POST("/verify-otp", controllers.VerifyOTP)
		auth.POST("/resend-otp", controllers.ResendOTP)

		// Forgot password flow
		auth.POST("/forgot-password", controllers.ForgotPassword)
		auth.GET("/verify-reset-token", controllers.VerifyResetToken) // frontend hits this to validate token
		auth.POST("/reset-password", controllers.ResetPassword)
	}

	// ── Protected routes ─────────────────────────────────────────────────────
	protected := api.Group("/")
	protected.Use(middlewares.AuthMiddleware())
	{
		protected.GET("/profile", controllers.GetProfile)
	}

	return r
}