package controllers

import (
	"fmt"
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

// MentorUpdateSession memperbarui data sesi: topik, catatan, bintang, status, jadwal, dan meet link.
//
// Jika field scheduled_at dikirim (reschedule), sistem akan:
//  1. Menolak jika sesi sudah completed/cancelled
//  2. Menolak jika waktu baru sudah lewat
//  3. Menolak jika waktu baru bertabrakan dengan sesi lain milik mentor yang sama
//     (sesi dirinya sendiri dikecualikan agar reschedule ke waktu yang sama tetap valid)
func MentorUpdateSession(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	sessionIDStr := c.Param("session_id")
	sessionID, err := strconv.ParseUint(sessionIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "session_id tidak valid")
		return
	}

	var input struct {
		Topic       string `json:"topic"`
		Notes       string `json:"notes"`
		Stars       int    `json:"stars"`
		Status      string `json:"status"`
		MeetLink    string `json:"meet_link"`
		ScheduledAt string `json:"scheduled_at"` // RFC3339, opsional — hanya untuk reschedule
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}

	var mentor models.MentorProfile
	if err := config.DB.Where("user_id = ?", userID).First(&mentor).Error; err != nil {
		utils.NotFound(c, "Profil mentor tidak ditemukan")
		return
	}

	var session models.Session
	if err := config.DB.
		Where("id = ? AND mentor_id = ?", sessionID, mentor.ID).
		First(&session).Error; err != nil {
		utils.NotFound(c, "Sesi tidak ditemukan")
		return
	}

	// ── Reschedule: validasi dan cek konflik ──────────────────────────────────
	if input.ScheduledAt != "" {
		if session.Status == models.SessionCompleted || session.Status == models.SessionCancelled {
			utils.BadRequest(c, "Sesi yang sudah selesai atau dibatalkan tidak dapat dijadwalkan ulang")
			return
		}

		// Terima RFC3339 dengan atau tanpa timezone suffix
		newTime, parseErr := time.Parse(time.RFC3339, input.ScheduledAt)
		if parseErr != nil {
			newTime, parseErr = time.Parse("2006-01-02T15:04:05", input.ScheduledAt)
			if parseErr != nil {
				utils.BadRequest(c, "Format scheduled_at tidak valid. Gunakan RFC3339, contoh: 2025-01-20T09:00:00Z")
				return
			}
		}

		// Tolak reschedule ke masa lalu (toleransi 5 menit untuk latency jaringan)
		if newTime.Before(time.Now().Add(-5 * time.Minute)) {
			utils.BadRequest(c, "Tidak dapat menjadwalkan sesi di waktu yang sudah lewat")
			return
		}

		// Cek konflik dengan sesi lain milik mentor ini.
		// session.ID dikecualikan agar sesi ini tidak konflik dengan dirinya sendiri.
		ok, conflict, checkErr := CheckMentorConflict(mentor.ID, newTime, session.Duration, session.ID)
		if checkErr != nil {
			utils.InternalError(c, "Gagal memvalidasi jadwal, coba lagi")
			return
		}
		if !ok {
			c.JSON(http.StatusConflict, gin.H{
				"message": fmt.Sprintf(
					"Jadwal bertabrakan dengan sesi lain yang berlangsung sampai %s",
					conflict.Slot.End.Format("15:04"),
				),
				"conflict": gin.H{
					"session_id": conflict.SessionID,
					"ends_at":    conflict.Slot.End.Format(time.RFC3339),
				},
			})
			return
		}

		session.ScheduledAt = newTime
	}

	// ── Update field konten (semua opsional) ──────────────────────────────────
	if input.MeetLink != "" {
		session.MeetLink = input.MeetLink
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

	// ── Perubahan status ──────────────────────────────────────────────────────
	if input.Status != "" {
		status := models.SessionStatus(input.Status)
		validStatuses := map[models.SessionStatus]bool{
			models.SessionUpcoming:  true,
			models.SessionOngoing:   true,
			models.SessionCompleted: true,
			models.SessionCancelled: true,
		}
		if !validStatuses[status] {
			utils.BadRequest(c, "Status tidak valid. Pilihan: upcoming, ongoing, completed, cancelled")
			return
		}
		// Lindungi dari rollback status yang tidak logis
		if (session.Status == models.SessionCompleted || session.Status == models.SessionCancelled) &&
			(status == models.SessionUpcoming || status == models.SessionOngoing) {
			utils.BadRequest(c, "Tidak dapat mengubah status sesi yang sudah selesai atau dibatalkan")
			return
		}

		session.Status = status

		if status == models.SessionOngoing {
			now := time.Now()
			session.StartedAt = &now
		}
		if status == models.SessionCompleted {
			now := time.Now()
			session.CompletedAt = &now
			// Update counter secara atomic — guard agar tidak bisa negatif
			config.DB.Model(&models.Order{}).
				Where("id = ? AND completed_sessions < total_sessions", session.OrderID).
				Updates(map[string]interface{}{
					"completed_sessions": config.DB.Raw("completed_sessions + 1"),
					"remaining_sessions": config.DB.Raw("remaining_sessions - 1"),
				})
		}
	}

	if err := config.DB.Save(&session).Error; err != nil {
		utils.InternalError(c, "Gagal menyimpan perubahan sesi")
		return
	}

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