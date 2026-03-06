package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-potensia/config"
	"go-potensia/models"
	"go-potensia/utils"

	"github.com/gin-gonic/gin"
)

// ─── DASHBOARD ────────────────────────────────────────────────────────────────

func MentorGetDashboard(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		utils.NotFound(c, "User tidak ditemukan")
		return
	}
	var mentor models.MentorProfile
	config.DB.Where("user_id = ?", userID).First(&mentor)

	// Sesi hari ini
	today := time.Now().Format("2006-01-02")
	var todaySessions []models.Session
	config.DB.Where("mentor_id = ? AND DATE(scheduled_at) = ?", mentor.ID, today).
		Preload("Order.Child").
		Preload("Order.Course").
		Order("scheduled_at ASC").
		Find(&todaySessions)

	// Statistik minggu ini
	weekStart := time.Now().AddDate(0, 0, -int(time.Now().Weekday()))
	var weekCount int64
	config.DB.Model(&models.Session{}).
		Where("mentor_id = ? AND scheduled_at >= ? AND status = ?", mentor.ID, weekStart, models.SessionCompleted).
		Count(&weekCount)

	// Siswa aktif (dari orders aktif)
	var activeOrders []models.Order
	config.DB.Where("mentor_id = ? AND status = ?", mentor.ID, models.OrderActive).
		Preload("Child").
		Preload("Course").
		Find(&activeOrders)

	c.JSON(http.StatusOK, gin.H{
		"mentor": gin.H{
			"id":             mentor.ID,
			"name":           user.Name,
			"avatar":         user.AvatarURL,
			"expertise":      mentor.Expertise,
			"rating":         mentor.Rating,
			"total_students": mentor.TotalStudents,
		},
		"stats": gin.H{
			"total_students":     mentor.TotalStudents,
			"session_today":      len(todaySessions),
			"session_this_week":  weekCount,
			"completed_sessions": mentor.TotalSessions,
			"earnings":           mentor.TotalEarnings,
		},
		"today_schedule": todaySessions,
		"active_students": activeOrders,
	})
}

// ─── PROFILE ─────────────────────────────────────────────────────────────────

func MentorGetProfile(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		utils.NotFound(c, "User tidak ditemukan")
		return
	}
	var mentor models.MentorProfile
	config.DB.Where("user_id = ?", userID).First(&mentor)

	var certs []models.MentorCertificate
	var achievements []models.MentorAchievement
	var gallery []models.MentorGallery
	var education []models.MentorEducation
	config.DB.Where("mentor_id = ?", mentor.ID).Find(&certs)
	config.DB.Where("mentor_id = ?", mentor.ID).Find(&achievements)
	config.DB.Where("mentor_id = ?", mentor.ID).Order("sort_order ASC").Find(&gallery)
	config.DB.Where("mentor_id = ?", mentor.ID).Find(&education)

	c.JSON(http.StatusOK, gin.H{
		"id":             mentor.ID,
		"user_id":        user.ID,
		"name":           user.Name,
		"email":          user.Email,
		"phone":          user.Phone,
		"avatar":         user.AvatarURL,
		"expertise":      mentor.Expertise,
		"bio":            mentor.Bio,
		"rating":         mentor.Rating,
		"total_review":   mentor.TotalReview,
		"total_students": mentor.TotalStudents,
		"total_sessions": mentor.TotalSessions,
		"province":       mentor.Province,
		"regency":        mentor.Regency,
		"district":       mentor.District,
		"address":        mentor.Address,
		"bank": gin.H{
			"bank":         mentor.BankName,
			"account":      mentor.BankAccount,
			"account_name": mentor.BankAccountName,
		},
		"certificates": certs,
		"achievements": achievements,
		"gallery":      gallery,
		"education":    education,
	})
}

func MentorUpdateProfile(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var input struct {
		Name      string `json:"name"`
		Phone     string `json:"phone"`
		Expertise string `json:"expertise"`
		Bio       string `json:"bio"`
		Province  string `json:"province"`
		Regency   string `json:"regency"`
		District  string `json:"district"`
		Address   string `json:"address"`
		BankName        string `json:"bank_name"`
		BankAccount     string `json:"bank_account"`
		BankAccountName string `json:"bank_account_name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}

	var user models.User
	config.DB.First(&user, userID)
	if input.Name != "" {
		user.Name = strings.TrimSpace(input.Name)
	}
	if input.Phone != "" {
		if !utils.IsValidPhone(input.Phone) {
			utils.BadRequest(c, "Format nomor HP tidak valid")
			return
		}
		user.Phone = input.Phone
	}
	config.DB.Save(&user)

	var mentor models.MentorProfile
	config.DB.Where("user_id = ?", userID).First(&mentor)
	if input.Expertise != "" {
		mentor.Expertise = input.Expertise
	}
	if input.Bio != "" {
		mentor.Bio = input.Bio
	}
	if input.Province != "" {
		mentor.Province = input.Province
	}
	if input.Regency != "" {
		mentor.Regency = input.Regency
	}
	if input.District != "" {
		mentor.District = input.District
	}
	if input.Address != "" {
		mentor.Address = input.Address
	}
	if input.BankName != "" {
		mentor.BankName = input.BankName
	}
	if input.BankAccount != "" {
		mentor.BankAccount = input.BankAccount
	}
	if input.BankAccountName != "" {
		mentor.BankAccountName = input.BankAccountName
	}
	config.DB.Save(&mentor)

	c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui"})
}

// ─── STUDENTS ────────────────────────────────────────────────────────────────

func MentorGetStudents(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var mentor models.MentorProfile
	config.DB.Where("user_id = ?", userID).First(&mentor)

	statusFilter := c.Query("status") // "active" | "completed" | ""
	search := c.Query("search")

	query := config.DB.Model(&models.Order{}).
		Where("mentor_id = ?", mentor.ID).
		Preload("Child").
		Preload("Course")

	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}

	var orders []models.Order
	query.Find(&orders)

	// Filter by search (nama anak / kursus)
	if search != "" {
		search = strings.ToLower(search)
		filtered := orders[:0]
		for _, o := range orders {
			if strings.Contains(strings.ToLower(o.Child.Name), search) ||
				strings.Contains(strings.ToLower(o.Course.Title), search) {
				filtered = append(filtered, o)
			}
		}
		orders = filtered
	}

	c.JSON(http.StatusOK, gin.H{"students": orders, "total": len(orders)})
}

func MentorGetStudentDetail(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	orderIDStr := c.Param("order_id")
	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "order_id tidak valid")
		return
	}

	var mentor models.MentorProfile
	config.DB.Where("user_id = ?", userID).First(&mentor)

	var order models.Order
	if err := config.DB.
		Where("id = ? AND mentor_id = ?", orderID, mentor.ID).
		Preload("Child").
		Preload("Course").
		Preload("Package").
		Preload("Parent").
		First(&order).Error; err != nil {
		utils.NotFound(c, "Data tidak ditemukan")
		return
	}

	var sessions []models.Session
	config.DB.Where("order_id = ?", order.ID).Order("scheduled_at ASC").Find(&sessions)

	var skills []models.SkillProgress
	config.DB.Where("order_id = ?", order.ID).Find(&skills)

	c.JSON(http.StatusOK, gin.H{
		"order":    order,
		"sessions": sessions,
		"skills":   skills,
	})
}

// ─── SCHEDULE ────────────────────────────────────────────────────────────────

func MentorGetSchedule(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var mentor models.MentorProfile
	config.DB.Where("user_id = ?", userID).First(&mentor)

	dateFilter := c.Query("date") // "YYYY-MM-DD", optional

	query := config.DB.Model(&models.Session{}).
		Where("mentor_id = ?", mentor.ID).
		Where("status IN ?", []models.SessionStatus{models.SessionUpcoming, models.SessionOngoing, models.SessionCompleted}).
		Preload("Order.Child").
		Preload("Order.Course").
		Order("scheduled_at ASC")

	if dateFilter != "" {
		query = query.Where("DATE(scheduled_at) = ?", dateFilter)
	} else {
		// Default: 7 hari ke depan
		query = query.Where("scheduled_at >= ?", time.Now().Add(-24*time.Hour))
	}

	var sessions []models.Session
	query.Find(&sessions)

	c.JSON(http.StatusOK, gin.H{"schedule": sessions, "total": len(sessions)})
}

func MentorUpdateSession(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	sessionIDStr := c.Param("session_id")
	sessionID, err := strconv.ParseUint(sessionIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "session_id tidak valid")
		return
	}

	var input struct {
		Topic  string `json:"topic"`
		Notes  string `json:"notes"`
		Stars  int    `json:"stars"`
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}

	var mentor models.MentorProfile
	config.DB.Where("user_id = ?", userID).First(&mentor)

	var session models.Session
	if err := config.DB.
		Where("id = ? AND mentor_id = ?", sessionID, mentor.ID).
		First(&session).Error; err != nil {
		utils.NotFound(c, "Sesi tidak ditemukan")
		return
	}

	if input.Topic != "" {
		session.Topic = input.Topic
	}
	if input.Notes != "" {
		session.Notes = input.Notes
	}
	if input.Stars >= 1 && input.Stars <= 5 {
		session.Stars = input.Stars
	}
	if input.Status != "" {
		status := models.SessionStatus(input.Status)
		validStatuses := map[models.SessionStatus]bool{
			models.SessionUpcoming:  true,
			models.SessionOngoing:   true,
			models.SessionCompleted: true,
			models.SessionCancelled: true,
		}
		if !validStatuses[status] {
			utils.BadRequest(c, "Status tidak valid")
			return
		}
		session.Status = status
		if status == models.SessionCompleted {
			now := time.Now()
			session.CompletedAt = &now
			// Update counter di Order
			config.DB.Model(&models.Order{}).
				Where("id = ?", session.OrderID).
				Updates(map[string]interface{}{
					"completed_sessions":  config.DB.Raw("completed_sessions + 1"),
					"remaining_sessions":  config.DB.Raw("remaining_sessions - 1"),
				})
		}
	}
	config.DB.Save(&session)

	c.JSON(http.StatusOK, gin.H{"message": "Sesi berhasil diperbarui", "session": session})
}

// ─── REVIEWS ─────────────────────────────────────────────────────────────────

func MentorGetReviews(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var mentor models.MentorProfile
	config.DB.Where("user_id = ?", userID).First(&mentor)

	courseFilter := c.Query("course_id")
	query := config.DB.Model(&models.Review{}).Where("mentor_id = ?", mentor.ID)
	if courseFilter != "" {
		query = query.Where("course_id = ?", courseFilter)
	}

	var reviews []models.Review
	query.Order("created_at DESC").Find(&reviews)

	// Hitung rata-rata
	var totalRating float64
	for _, r := range reviews {
		totalRating += float64(r.Rating)
	}
	avg := 0.0
	if len(reviews) > 0 {
		avg = totalRating / float64(len(reviews))
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews":    reviews,
		"total":      len(reviews),
		"avg_rating": avg,
	})
}