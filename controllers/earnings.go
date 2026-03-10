package controllers

import (
	"math"
	"time"

	"go-potensia/config"
	"go-potensia/models"
	"go-potensia/utils"

	"github.com/gin-gonic/gin"
)

// ─── CONSTANTS ────────────────────────────────────────────────────────────────

const platformFeeRate = 0.10 // 10% platform fee

// ─── BANK ACCOUNT ─────────────────────────────────────────────────────────────

// MentorGetBankAccount godoc
// GET /mentor/bank
// Ambil rekening bank mentor yang sedang login.
func MentorGetBankAccount(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var mentor models.MentorProfile
	if err := config.DB.Where("user_id = ?", userID).First(&mentor).Error; err != nil {
		utils.NotFound(c, "Profil mentor tidak ditemukan")
		return
	}

	var bank models.MentorBankAccount
	if err := config.DB.Where("mentor_id = ?", mentor.ID).First(&bank).Error; err != nil {
		// Belum ada data bank — kembalikan objek kosong
		utils.OK(c, gin.H{
			"has_bank_account": false,
			"bank_account":     nil,
		})
		return
	}

	utils.OK(c, gin.H{
		"has_bank_account": true,
		"bank_account":     bank,
	})
}

// MentorUpsertBankAccount godoc
// PUT /mentor/bank
// Buat atau update rekening bank mentor.
func MentorUpsertBankAccount(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var mentor models.MentorProfile
	if err := config.DB.Where("user_id = ?", userID).First(&mentor).Error; err != nil {
		utils.NotFound(c, "Profil mentor tidak ditemukan")
		return
	}

	var req struct {
		BankName      string                `json:"bank_name" binding:"required"`
		AccountNumber string                `json:"account_number" binding:"required"`
		AccountHolder string                `json:"account_holder" binding:"required"`
		WithdrawalDay models.WithdrawalDay  `json:"withdrawal_day"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Data tidak valid: "+err.Error())
		return
	}

	if req.WithdrawalDay == "" {
		req.WithdrawalDay = models.WithdrawalDayJumat
	}

	var bank models.MentorBankAccount
	config.DB.Where("mentor_id = ?", mentor.ID).First(&bank)

	bank.MentorID      = mentor.ID
	bank.BankName      = req.BankName
	bank.AccountNumber = req.AccountNumber
	bank.AccountHolder = req.AccountHolder
	bank.WithdrawalDay = req.WithdrawalDay
	// Reset verifikasi jika nomor rekening berubah
	bank.IsVerified = false

	if bank.ID == 0 {
		config.DB.Create(&bank)
	} else {
		config.DB.Save(&bank)
	}

	// Sync ke MentorProfile juga (legacy compatibility)
	config.DB.Model(&mentor).Updates(map[string]interface{}{
		"bank_name":         req.BankName,
		"bank_account":      req.AccountNumber,
		"bank_account_name": req.AccountHolder,
	})

	utils.OK(c, bank)
}

// ─── EARNINGS ─────────────────────────────────────────────────────────────────

// MentorGetEarnings godoc
// GET /mentor/earnings?week=current|YYYY-WW
// Tampil pendapatan mentor minggu ini + breakdown per sesi.
func MentorGetEarnings(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var mentor models.MentorProfile
	if err := config.DB.Where("user_id = ?", userID).First(&mentor).Error; err != nil {
		utils.NotFound(c, "Profil mentor tidak ditemukan")
		return
	}

	// Hitung range minggu ini (Senin 00:00 – Minggu 23:59)
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7 in ISO
	}
	weekStart := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
	weekEnd   := weekStart.AddDate(0, 0, 7).Add(-time.Second)

	// Hitung auto-disbursement date (hari pencairan dari bank account)
	var bank models.MentorBankAccount
	config.DB.Where("mentor_id = ?", mentor.ID).First(&bank)
	disbursementDay := models.WithdrawalDayJumat
	if bank.ID != 0 {
		disbursementDay = bank.WithdrawalDay
	}
	disbursementDate := nextWeekday(weekStart, disbursementDay)

	// Earnings minggu ini
	var weekEarnings []models.MentorEarning
	config.DB.Where("mentor_id = ? AND earned_at BETWEEN ? AND ?", mentor.ID, weekStart, weekEnd).
		Order("earned_at ASC").
		Find(&weekEarnings)

	// Hitung total
	var grossTotal, feeTotal, netTotal int64
	for _, e := range weekEarnings {
		grossTotal += e.GrossAmount
		feeTotal  += e.FeeAmount
		netTotal  += e.NetAmount
	}

	utils.OK(c, gin.H{
		"period": gin.H{
			"week_start":        weekStart.Format("2006-01-02"),
			"week_end":          weekEnd.Format("2006-01-02"),
			"disbursement_date": disbursementDate.Format("2006-01-02"),
			"disbursement_day":  disbursementDay,
		},
		"summary": gin.H{
			"gross_total": grossTotal,
			"fee_total":   feeTotal,
			"net_total":   netTotal,
			"fee_rate":    platformFeeRate,
		},
		"earnings": weekEarnings,
	})
}

// MentorGetEarningsHistory godoc
// GET /mentor/earnings/history?page=1&limit=10
// Riwayat pencairan (weekly withdrawal history).
func MentorGetEarningsHistory(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var mentor models.MentorProfile
	if err := config.DB.Where("user_id = ?", userID).First(&mentor).Error; err != nil {
		utils.NotFound(c, "Profil mentor tidak ditemukan")
		return
	}

	page, limit := utils.Paginate(c)
	offset := (page - 1) * limit

	var total int64
	config.DB.Model(&models.MentorWithdrawal{}).
		Where("mentor_id = ?", mentor.ID).
		Count(&total)

	var withdrawals []models.MentorWithdrawal
	config.DB.Where("mentor_id = ?", mentor.ID).
		Order("period_start DESC").
		Limit(limit).
		Offset(offset).
		Find(&withdrawals)

	utils.PaginatedOK(c, withdrawals, total, page, limit)
}

// ─── STUDENT PROGRESS (untuk mentor) ─────────────────────────────────────────

// MentorGetStudentProgress godoc
// GET /mentor/students/:order_id/progress
// Detail progress skill anak dalam order tertentu.
func MentorGetStudentProgress(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var mentor models.MentorProfile
	if err := config.DB.Where("user_id = ?", userID).First(&mentor).Error; err != nil {
		utils.NotFound(c, "Profil mentor tidak ditemukan")
		return
	}

	orderID := c.Param("order_id")
	var order models.Order
	if err := config.DB.Where("id = ? AND mentor_id = ?", orderID, mentor.ID).
		Preload("Child").
		Preload("Course").
		Preload("Package").
		First(&order).Error; err != nil {
		utils.NotFound(c, "Order tidak ditemukan")
		return
	}

	var skillProgress []models.SkillProgress
	config.DB.Where("order_id = ?", order.ID).Find(&skillProgress)

	// Sesi-sesi yang sudah selesai dalam order ini
	var completedSessions []models.Session
	config.DB.Where("order_id = ? AND status = ?", order.ID, models.SessionCompleted).
		Order("session_number ASC").
		Find(&completedSessions)

	utils.OK(c, gin.H{
		"order":              order,
		"skill_progress":     skillProgress,
		"completed_sessions": completedSessions,
		"total_sessions":     order.TotalSessions,
		"done_sessions":      order.CompletedSessions,
		"overall_progress": func() int {
			if order.TotalSessions == 0 {
				return 0
			}
			return int(math.Round(float64(order.CompletedSessions) / float64(order.TotalSessions) * 100))
		}(),
	})
}

// MentorUpdateStudentProgress godoc
// PUT /mentor/students/:order_id/progress
// Update nilai skill progress anak oleh mentor.
func MentorUpdateStudentProgress(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var mentor models.MentorProfile
	if err := config.DB.Where("user_id = ?", userID).First(&mentor).Error; err != nil {
		utils.NotFound(c, "Profil mentor tidak ditemukan")
		return
	}

	orderID := c.Param("order_id")
	var order models.Order
	if err := config.DB.Where("id = ? AND mentor_id = ?", orderID, mentor.ID).First(&order).Error; err != nil {
		utils.NotFound(c, "Order tidak ditemukan atau bukan milik kamu")
		return
	}

	var req struct {
		Skills []struct {
			SkillName string `json:"skill_name" binding:"required"`
			Progress  int    `json:"progress" binding:"min=0,max=100"`
		} `json:"skills" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Data tidak valid: "+err.Error())
		return
	}

	for _, s := range req.Skills {
		var sp models.SkillProgress
		config.DB.Where("order_id = ? AND skill_name = ?", order.ID, s.SkillName).First(&sp)
		sp.OrderID   = order.ID
		sp.ChildID   = order.ChildID
		sp.CourseID  = order.CourseID
		sp.SkillName = s.SkillName
		sp.Progress  = s.Progress
		if sp.ID == 0 {
			config.DB.Create(&sp)
		} else {
			config.DB.Save(&sp)
		}
	}

	utils.OK(c, gin.H{"message": "Progress siswa berhasil diperbarui"})
}

// ─── SCHEDULE DETAIL (parent view of single order sessions) ──────────────────

// ParentGetOrderSchedule godoc
// GET /parent/orders/:order_id/schedule
// List semua sesi dalam sebuah order (untuk schedule-detail.tsx).
func ParentGetOrderSchedule(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	orderID := c.Param("order_id")
	var order models.Order
	if err := config.DB.Where("id = ? AND parent_id = ?", orderID, userID).
		Preload("Course").
		Preload("Package").
		First(&order).Error; err != nil {
		utils.NotFound(c, "Order tidak ditemukan")
		return
	}

	// Cari mentor user
	var mentorUser models.User
	config.DB.Where("id = (SELECT user_id FROM mentor_profiles WHERE id = ?)", order.MentorID).First(&mentorUser)

	var sessions []models.Session
	config.DB.Where("order_id = ?", order.ID).
		Order("session_number ASC").
		Find(&sessions)

	utils.OK(c, gin.H{
		"order":       order,
		"mentor_name": mentorUser.Name,
		"sessions":    sessions,
	})
}

// ParentGetOrderProgress godoc
// GET /parent/orders/:order_id/progress
// Detail progress skill anak dalam order (untuk progress-detail.tsx).
func ParentGetOrderProgress(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	orderID := c.Param("order_id")
	var order models.Order
	if err := config.DB.Where("id = ? AND parent_id = ?", orderID, userID).
		Preload("Child").
		Preload("Course").
		Preload("Package").
		First(&order).Error; err != nil {
		utils.NotFound(c, "Order tidak ditemukan")
		return
	}

	var skillProgress []models.SkillProgress
	config.DB.Where("order_id = ?", order.ID).Find(&skillProgress)

	var completedSessions []models.Session
	config.DB.Where("order_id = ? AND status = ?", order.ID, models.SessionCompleted).
		Order("session_number ASC").
		Find(&completedSessions)

	overallProgress := 0
	if order.TotalSessions > 0 {
		overallProgress = int(math.Round(float64(order.CompletedSessions) / float64(order.TotalSessions) * 100))
	}

	utils.OK(c, gin.H{
		"order":              order,
		"overall_progress":   overallProgress,
		"skill_progress":     skillProgress,
		"completed_sessions": completedSessions,
	})
}

// ParentGetProgressSchedule godoc
// GET /parent/progress-schedule
// List semua anak + kursus aktif mereka (untuk progress-schedule.tsx / Pantau Belajar).
func ParentGetProgressSchedule(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var children []models.Child
	config.DB.Where("parent_id = ?", userID).Find(&children)

	type ChildProgress struct {
		Child          models.Child         `json:"child"`
		ActiveSchedules []ActiveScheduleItem `json:"active_schedules"`
		ExpiredSchedules []ActiveScheduleItem `json:"expired_schedules"`
	}

	type result []ChildProgress

	var out result

	for _, child := range children {
		// Semua order untuk anak ini (active = belum expired, completed = expired)
		var orders []models.Order
		config.DB.Where("child_id = ? AND parent_id = ? AND status IN ?", child.ID, userID,
			[]models.OrderStatus{models.OrderActive, models.OrderCompleted}).
			Preload("Course").
			Preload("Package").
			Find(&orders)

		var active, expired []ActiveScheduleItem

		for _, o := range orders {
			// Ambil nama mentor
			var mentorUser models.User
			config.DB.Where("id = (SELECT user_id FROM mentor_profiles WHERE id = ?)", o.MentorID).First(&mentorUser)

			var mentorProfile models.MentorProfile
			config.DB.Where("id = ?", o.MentorID).First(&mentorProfile)

			item := ActiveScheduleItem{
				OrderID:           o.ID,
				MentorName:        mentorUser.Name,
				MentorProfileID:   o.MentorID,
				MentorPhone:       mentorUser.Phone,
				Subject:           o.Course.Title,
				PackageName:       o.Package.Name,
				TotalSessions:     o.TotalSessions,
				CompletedSessions: o.CompletedSessions,
				RemainingSessions: o.RemainingSessions,
				IsExpired:         o.Status == models.OrderCompleted || o.Status == models.OrderCancelled,
			}

			if item.IsExpired {
				expired = append(expired, item)
			} else {
				active = append(active, item)
			}
		}

		out = append(out, ChildProgress{
			Child:            child,
			ActiveSchedules:  active,
			ExpiredSchedules: expired,
		})
	}

	utils.OK(c, out)
}

// ActiveScheduleItem adalah DTO untuk item jadwal aktif anak
type ActiveScheduleItem struct {
	OrderID           uint   `json:"order_id"`
	MentorProfileID   uint   `json:"mentor_profile_id"`
	MentorName        string `json:"mentor_name"`
	MentorPhone       string `json:"mentor_phone"`
	Subject           string `json:"subject"`
	PackageName       string `json:"package_name"`
	TotalSessions     int    `json:"total_sessions"`
	CompletedSessions int    `json:"completed_sessions"`
	RemainingSessions int    `json:"remaining_sessions"`
	IsExpired         bool   `json:"is_expired"`
}

// ─── HELPERS ──────────────────────────────────────────────────────────────────

// nextWeekday mengembalikan tanggal terdekat (dari weekStart) yang hari-nya sesuai.
func nextWeekday(from time.Time, day models.WithdrawalDay) time.Time {
	dayMap := map[models.WithdrawalDay]time.Weekday{
		models.WithdrawalDaySenin:  time.Monday,
		models.WithdrawalDaySelasa: time.Tuesday,
		models.WithdrawalDayRabu:   time.Wednesday,
		models.WithdrawalDayKamis:  time.Thursday,
		models.WithdrawalDayJumat:  time.Friday,
		models.WithdrawalDaySabtu:  time.Saturday,
		models.WithdrawalDayMinggu: time.Sunday,
	}
	target, ok := dayMap[day]
	if !ok {
		target = time.Friday
	}
	d := from
	for d.Weekday() != target {
		d = d.AddDate(0, 0, 1)
	}
	return d
}

// CreateEarningFromSession membuat MentorEarning ketika sesi selesai.
// Dipanggil dari MentorUpdateSession saat status → completed.
func CreateEarningFromSession(session models.Session, order models.Order, course models.Course, child models.Child) {
	// Hitung harga per sesi: total_price / total_sessions
	pricePerSession := int64(0)
	if order.TotalSessions > 0 {
		pricePerSession = order.TotalPrice / int64(order.TotalSessions)
	}
	feeAmount := int64(math.Round(float64(pricePerSession) * platformFeeRate))
	netAmount  := pricePerSession - feeAmount

	completedAt := time.Now()
	if session.CompletedAt != nil {
		completedAt = *session.CompletedAt
	}

	earning := models.MentorEarning{
		MentorID:      session.MentorID,
		SessionID:     session.ID,
		OrderID:       order.ID,
		ChildID:       child.ID,
		CourseID:      course.ID,
		CourseName:    course.Title,
		StudentName:   child.Name,
		PackageName:   order.Notes, // will be filled properly via package lookup at call site
		SessionNumber: session.SessionNumber,
		GrossAmount:   pricePerSession,
		FeeRate:       platformFeeRate,
		FeeAmount:     feeAmount,
		NetAmount:     netAmount,
		EarnedAt:      completedAt,
	}

	// Cek apakah sudah ada earning untuk session ini (idempotent)
	var existing models.MentorEarning
	if err := config.DB.Where("session_id = ?", session.ID).First(&existing).Error; err != nil {
		config.DB.Create(&earning)

		// Update cached total_earnings di mentor profile
		config.DB.Model(&models.MentorProfile{}).
			Where("id = ?", session.MentorID).
			Update("total_earnings", config.DB.Raw("total_earnings + ?", netAmount))
	}
}

// MentorGetEarningsBySession godoc
// GET /mentor/earnings/sessions?page=1&limit=20
// List semua earning per sesi (bukan per withdrawal).
func MentorGetEarningsBySession(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var mentor models.MentorProfile
	if err := config.DB.Where("user_id = ?", userID).First(&mentor).Error; err != nil {
		utils.NotFound(c, "Profil mentor tidak ditemukan")
		return
	}

	page, limit := utils.Paginate(c)
	offset := (page - 1) * limit

	var total int64
	config.DB.Model(&models.MentorEarning{}).
		Where("mentor_id = ?", mentor.ID).
		Count(&total)

	var earnings []models.MentorEarning
	config.DB.Where("mentor_id = ?", mentor.ID).
		Order("earned_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&earnings)

	// Summary keseluruhan
	var totalGross, totalFee, totalNet int64
	config.DB.Model(&models.MentorEarning{}).
		Where("mentor_id = ?", mentor.ID).
		Select("COALESCE(SUM(gross_amount),0), COALESCE(SUM(fee_amount),0), COALESCE(SUM(net_amount),0)").
		Row().Scan(&totalGross, &totalFee, &totalNet)

	utils.OK(c, gin.H{
		"summary": gin.H{
			"total_gross": totalGross,
			"total_fee":   totalFee,
			"total_net":   totalNet,
		},
		"data":       earnings,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"total_pages": int(math.Ceil(float64(total) / float64(limit))),
	})
}