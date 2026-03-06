package routes

import (
	"go-potensia/controllers"
	"go-potensia/middlewares"
	"go-potensia/models"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "go-potensia"})
	})

	api := r.Group("/api")

	// ── AUTH (public) ─────────────────────────────────────────────────────────
	auth := api.Group("/auth")
	{
		auth.POST("/register", controllers.Register)
		auth.POST("/login", controllers.Login)
		auth.POST("/verify-otp", controllers.VerifyOTP)
		auth.POST("/resend-otp", controllers.ResendOTP)
		auth.POST("/forgot-password", controllers.ForgotPassword)
		auth.GET("/verify-reset-token", controllers.VerifyResetToken)
		auth.POST("/reset-password", controllers.ResetPassword)
	}

	// ── PUBLIC: search & mentor detail ───────────────────────────────────────
	api.GET("/mentors", controllers.SearchMentors)
	api.GET("/mentors/:mentor_id", controllers.GetMentorPublicProfile)

	// ── PROTECTED (all roles) ─────────────────────────────────────────────────
	protected := api.Group("/")
	protected.Use(middlewares.AuthMiddleware())
	{
		// Auth actions (authenticated)
		protected.POST("/auth/change-password", controllers.ChangePassword)
		protected.POST("/auth/delete-account", controllers.DeleteAccount)

		// Notifications (all roles)
		protected.GET("/notifications", controllers.GetNotifications)
		protected.PUT("/notifications/:notif_id/read", controllers.MarkNotificationRead)
		protected.GET("/notifications/preferences", controllers.GetNotificationPreferences)
		protected.PUT("/notifications/preferences", controllers.UpdateNotificationPreferences)
	}

	// ── MENTOR routes ─────────────────────────────────────────────────────────
	mentor := api.Group("/mentor")
	mentor.Use(middlewares.AuthMiddleware(), middlewares.RequireRole(models.RoleMentor))
	{
		// Dashboard
		mentor.GET("/dashboard", controllers.MentorGetDashboard)

		// Profile
		mentor.GET("/profile", controllers.MentorGetProfile)
		mentor.PUT("/profile", controllers.MentorUpdateProfile)

		// Students
		mentor.GET("/students", controllers.MentorGetStudents)
		mentor.GET("/students/:order_id", controllers.MentorGetStudentDetail)

		// Schedule & sessions
		mentor.GET("/schedule", controllers.MentorGetSchedule)
		mentor.PUT("/sessions/:session_id", controllers.MentorUpdateSession)

		// Courses
		mentor.GET("/courses", controllers.MentorGetCourses)
		mentor.POST("/courses", controllers.MentorCreateCourse)
		mentor.GET("/courses/:course_id", controllers.MentorGetCourse)
		mentor.PUT("/courses/:course_id", controllers.MentorUpdateCourse)
		mentor.DELETE("/courses/:course_id", controllers.MentorDeleteCourse)

		// Reviews
		mentor.GET("/reviews", controllers.MentorGetReviews)
	}

	// ── PARENT routes ─────────────────────────────────────────────────────────
	parent := api.Group("/parent")
	parent.Use(middlewares.AuthMiddleware(), middlewares.RequireRole(models.RoleParent))
	{
		// Dashboard
		parent.GET("/dashboard", controllers.ParentGetDashboard)

		// Profile
		parent.GET("/profile", controllers.ParentGetProfile)
		parent.PUT("/profile", controllers.ParentUpdateProfile)

		// Children
		parent.GET("/children", controllers.ParentGetChildren)
		parent.POST("/children", controllers.ParentCreateChild)
		parent.PUT("/children/:child_id", controllers.ParentUpdateChild)
		parent.DELETE("/children/:child_id", controllers.ParentDeleteChild)

		// Progress anak
		parent.GET("/children/:child_id/progress", controllers.ParentGetChildProgress)

		// Orders
		parent.GET("/orders", controllers.ParentGetOrders)
		parent.POST("/orders", controllers.ParentCreateOrder)

		// Payments / Invoice
		parent.GET("/payments", controllers.ParentGetPayments)
		parent.POST("/payments/:invoice_id", controllers.ParentMakePayment)

		// Schedule
		parent.GET("/schedule", controllers.ParentGetSchedule)

		// Reviews
		parent.POST("/reviews", controllers.ParentSubmitReview)
		parent.GET("/reviews", controllers.ParentGetReviews)
	}

	return r
}