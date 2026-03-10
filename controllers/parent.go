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

func ParentGetDashboard(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		utils.NotFound(c, "User tidak ditemukan")
		return
	}

	// Anak-anak
	var children []models.Child
	config.DB.Where("parent_id = ?", userID).Find(&children)

	// Statistik
	var totalSessions, completedSessions, upcomingCount int64
	var childIDs []uint
	for _, ch := range children {
		childIDs = append(childIDs, ch.ID)
	}
	if len(childIDs) > 0 {
		config.DB.Model(&models.Session{}).Where("child_id IN ?", childIDs).Count(&totalSessions)
		config.DB.Model(&models.Session{}).Where("child_id IN ? AND status = ?", childIDs, models.SessionCompleted).Count(&completedSessions)
		config.DB.Model(&models.Session{}).Where("child_id IN ? AND status = ?", childIDs, models.SessionUpcoming).Count(&upcomingCount)
	}

	// Invoice belum bayar
	var unpaidInvoice models.Invoice
	config.DB.Where("parent_id = ? AND status = ?", userID, models.PaymentUnpaid).
		Order("due_date ASC").
		First(&unpaidInvoice)

	// Jadwal terdekat per anak
	type childWithSchedule struct {
		models.Child
		NextSession *models.Session `json:"next_session"`
		MentorName  string          `json:"mentor_name"`
		CourseName  string          `json:"course_name"`
		Progress    int             `json:"progress"`
	}

	var childDetails []childWithSchedule
	for _, ch := range children {
		var nextSession models.Session
		config.DB.Where("child_id = ? AND status = ?", ch.ID, models.SessionUpcoming).
			Order("scheduled_at ASC").
			Preload("Order.Course").
			First(&nextSession)

		var order models.Order
		config.DB.Where("child_id = ? AND status = ?", ch.ID, models.OrderActive).
			Preload("Course").
			First(&order)

		var mentorUser models.User
		var progress int
		if order.ID != 0 && order.TotalSessions > 0 {
			progress = (order.CompletedSessions * 100) / order.TotalSessions
			var mp models.MentorProfile
			config.DB.First(&mp, order.MentorID)
			config.DB.First(&mentorUser, mp.UserID)
		}

		cd := childWithSchedule{Child: ch, Progress: progress}
		if nextSession.ID != 0 {
			cd.NextSession = &nextSession
		}
		if order.ID != 0 {
			cd.CourseName = order.Course.Title
			cd.MentorName = mentorUser.Name
		}
		childDetails = append(childDetails, cd)
	}

	var totalSpent int64
	config.DB.Model(&models.Invoice{}).
		Where("parent_id = ? AND status = ?", userID, models.PaymentPaid).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalSpent)

	c.JSON(http.StatusOK, gin.H{
		"parent": gin.H{
			"id":     user.ID,
			"name":   user.Name,
			"avatar": user.AvatarURL,
		},
		"children": childDetails,
		"stats": gin.H{
			"total_sessions":     totalSessions,
			"completed_sessions": completedSessions,
			"upcoming_sessions":  upcomingCount,
			"total_spent":        totalSpent,
		},
		"upcoming_payment": unpaidInvoice,
	})
}

// ─── PROFILE ─────────────────────────────────────────────────────────────────

func ParentGetProfile(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		utils.NotFound(c, "User tidak ditemukan")
		return
	}
	var parent models.ParentProfile
	config.DB.Where("user_id = ?", userID).First(&parent)

	var children []models.Child
	config.DB.Where("parent_id = ?", userID).Find(&children)

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"name":     user.Name,
		"email":    user.Email,
		"phone":    user.Phone,
		"avatar":   user.AvatarURL,
		"address":  parent.Address,
		"children": children,
	})
}

func ParentUpdateProfile(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var input struct {
		Name    string `json:"name"`
		Phone   string `json:"phone"`
		Address string `json:"address"`
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

	var parent models.ParentProfile
	config.DB.Where("user_id = ?", userID).First(&parent)
	if input.Address != "" {
		parent.Address = input.Address
	}
	config.DB.Save(&parent)

	c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui"})
}

// ─── CHILDREN ────────────────────────────────────────────────────────────────

func ParentGetChildren(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var children []models.Child
	config.DB.Where("parent_id = ?", userID).Find(&children)

	c.JSON(http.StatusOK, gin.H{"children": children, "total": len(children)})
}

func ParentCreateChild(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var input struct {
		Name      string `json:"name"`
		BirthDate string `json:"birth_date"` // "YYYY-MM-DD"
		Gender    string `json:"gender"`
		Education string `json:"education"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}
	if input.Name == "" {
		utils.BadRequest(c, "Nama anak wajib diisi")
		return
	}
	birthDate, err := time.Parse("2006-01-02", input.BirthDate)
	if err != nil {
		utils.BadRequest(c, "Format tanggal lahir tidak valid. Gunakan YYYY-MM-DD")
		return
	}
	gender := models.Gender(input.Gender)
	if gender != models.GenderMale && gender != models.GenderFemale {
		utils.BadRequest(c, "Gender harus 'Laki-laki' atau 'Perempuan'")
		return
	}

	child := models.Child{
		ParentID:  userID,
		Name:      strings.ToUpper(strings.TrimSpace(input.Name)),
		BirthDate: birthDate,
		Gender:    gender,
		Education: input.Education,
	}
	if err := config.DB.Create(&child).Error; err != nil {
		utils.InternalError(c, "Gagal menyimpan data anak")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Data anak berhasil ditambahkan", "child": child})
}

func ParentUpdateChild(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	childIDStr := c.Param("child_id")
	childID, err := strconv.ParseUint(childIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "child_id tidak valid")
		return
	}

	var child models.Child
	if err := config.DB.Where("id = ? AND parent_id = ?", childID, userID).First(&child).Error; err != nil {
		utils.NotFound(c, "Data anak tidak ditemukan")
		return
	}

	var input struct {
		Name      string `json:"name"`
		BirthDate string `json:"birth_date"`
		Gender    string `json:"gender"`
		Education string `json:"education"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}

	if input.Name != "" {
		child.Name = strings.ToUpper(strings.TrimSpace(input.Name))
	}
	if input.BirthDate != "" {
		bd, err := time.Parse("2006-01-02", input.BirthDate)
		if err != nil {
			utils.BadRequest(c, "Format tanggal lahir tidak valid")
			return
		}
		child.BirthDate = bd
	}
	if input.Gender != "" {
		child.Gender = models.Gender(input.Gender)
	}
	if input.Education != "" {
		child.Education = input.Education
	}
	config.DB.Save(&child)

	c.JSON(http.StatusOK, gin.H{"message": "Data anak berhasil diperbarui", "child": child})
}

func ParentDeleteChild(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	childIDStr := c.Param("child_id")
	childID, err := strconv.ParseUint(childIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "child_id tidak valid")
		return
	}

	var child models.Child
	if err := config.DB.Where("id = ? AND parent_id = ?", childID, userID).First(&child).Error; err != nil {
		utils.NotFound(c, "Data anak tidak ditemukan")
		return
	}

	// Cek tidak ada order aktif
	var activeCount int64
	config.DB.Model(&models.Order{}).
		Where("child_id = ? AND status IN ?", childID, []models.OrderStatus{models.OrderPending, models.OrderActive}).
		Count(&activeCount)
	if activeCount > 0 {
		utils.BadRequest(c, "Data anak tidak dapat dihapus karena masih ada kursus aktif")
		return
	}

	config.DB.Delete(&child)
	c.JSON(http.StatusOK, gin.H{"message": "Data anak berhasil dihapus"})
}

// ─── CHILD PROGRESS ──────────────────────────────────────────────────────────

func ParentGetChildProgress(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	childIDStr := c.Param("child_id")
	childID, err := strconv.ParseUint(childIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "child_id tidak valid")
		return
	}

	var child models.Child
	if err := config.DB.Where("id = ? AND parent_id = ?", childID, userID).First(&child).Error; err != nil {
		utils.NotFound(c, "Data anak tidak ditemukan")
		return
	}

	var orders []models.Order
	config.DB.Where("child_id = ?", child.ID).
		Preload("Course").
		Preload("Package").
		Find(&orders)

	type orderWithProgress struct {
		models.Order
		Skills       []models.SkillProgress `json:"skills"`
		RecentSessions []models.Session     `json:"recent_sessions"`
	}

	var result []orderWithProgress
	for _, o := range orders {
		var skills []models.SkillProgress
		config.DB.Where("order_id = ?", o.ID).Find(&skills)

		var sessions []models.Session
		config.DB.Where("order_id = ?", o.ID).
			Order("scheduled_at DESC").
			Limit(5).
			Find(&sessions)

		result = append(result, orderWithProgress{
			Order:          o,
			Skills:         skills,
			RecentSessions: sessions,
		})
	}

	c.JSON(http.StatusOK, gin.H{"child": child, "progress": result})
}

// ─── ORDERS ──────────────────────────────────────────────────────────────────

func ParentGetOrders(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var orders []models.Order
	config.DB.Where("parent_id = ?", userID).
		Preload("Child").
		Preload("Course").
		Preload("Package").
		Order("created_at DESC").
		Find(&orders)

	c.JSON(http.StatusOK, gin.H{"orders": orders, "total": len(orders)})
}

func ParentCreateOrder(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var input struct {
		ChildID       uint   `json:"child_id"`
		CourseID      uint   `json:"course_id"`
		PackageID     uint   `json:"package_id"`
		PreferredDays string `json:"preferred_days"` // "1,3,5"
		PreferredTime string `json:"preferred_time"` // "09:00"
		Notes         string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}
	if input.ChildID == 0 || input.CourseID == 0 || input.PackageID == 0 {
		utils.BadRequest(c, "child_id, course_id, dan package_id wajib diisi")
		return
	}

	// Validasi anak milik parent ini
	var child models.Child
	if err := config.DB.Where("id = ? AND parent_id = ?", input.ChildID, userID).First(&child).Error; err != nil {
		utils.NotFound(c, "Data anak tidak ditemukan")
		return
	}

	// Validasi course & package
	var pkg models.CoursePackage
	if err := config.DB.Where("id = ? AND course_id = ? AND is_active = ?", input.PackageID, input.CourseID, true).First(&pkg).Error; err != nil {
		utils.NotFound(c, "Paket kursus tidak ditemukan atau tidak aktif")
		return
	}

	var course models.Course
	if err := config.DB.Where("id = ? AND status = ?", input.CourseID, models.CourseStatusActive).First(&course).Error; err != nil {
		utils.NotFound(c, "Kursus tidak ditemukan atau tidak aktif")
		return
	}

	order := models.Order{
		ParentID:           userID,
		ChildID:            input.ChildID,
		CourseID:           input.CourseID,
		PackageID:          input.PackageID,
		MentorID:           course.MentorID,
		TotalSessions:      pkg.TotalSessions,
		RemainingSessions:  pkg.TotalSessions,
		DurationPerSession: pkg.DurationPerSession,
		TotalPrice:         pkg.Price,
		Status:             models.OrderPending,
		PreferredDays:      input.PreferredDays,
		PreferredTime:      input.PreferredTime,
		Notes:              input.Notes,
	}
	if err := config.DB.Create(&order).Error; err != nil {
		utils.InternalError(c, "Gagal membuat order")
		return
	}

	// Buat invoice
	invoice := models.Invoice{
		OrderID:     order.ID,
		ParentID:    userID,
		Amount:      pkg.Price,
		Description: "Paket " + pkg.Name + " - " + course.Title,
		Period:      time.Now().Format("January 2006"),
		Status:      models.PaymentUnpaid,
		DueDate:     time.Now().Add(3 * 24 * time.Hour),
	}
	config.DB.Create(&invoice)

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Order berhasil dibuat",
		"order_id":   order.ID,
		"invoice_id": invoice.ID,
	})
}

// ─── PAYMENTS ────────────────────────────────────────────────────────────────

func ParentGetPayments(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var invoices []models.Invoice
	config.DB.Where("parent_id = ?", userID).
		Preload("Order.Course").
		Preload("Order.Package").
		Preload("Order.Child").
		Order("created_at DESC").
		Find(&invoices)

	c.JSON(http.StatusOK, gin.H{"payments": invoices, "total": len(invoices)})
}

// ParentMakePayment memproses pembayaran invoice dan — jika berhasil —
// mengaktifkan order lalu men-generate seluruh sesi secara otomatis.
//
// Alur:
//  1. Validasi invoice (milik parent, belum bayar, belum expired)
//  2. Ambil order + preferensi jadwal (preferred_days, preferred_time, duration)
//  3. Hitung semua tanggal sesi berdasarkan preferred_days mulai hari ini+1
//  4. Cek konflik untuk SETIAP sesi — jika ada 1 saja yang bentrok, tolak seluruh payment
//  5. Jika semua aman: tandai invoice paid, aktifkan order, simpan semua session ke DB
func ParentMakePayment(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	invoiceIDStr := c.Param("invoice_id")
	invoiceID, err := strconv.ParseUint(invoiceIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "invoice_id tidak valid")
		return
	}

	var input struct {
		Method   string `json:"method"` // "bank_transfer" | "e_wallet" | "virtual_account"
		ProofURL string `json:"proof_url"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}

	validMethods := map[string]bool{
		string(models.PaymentMethodBankTransfer): true,
		string(models.PaymentMethodEWallet):      true,
		string(models.PaymentMethodVA):           true,
	}
	if !validMethods[input.Method] {
		utils.BadRequest(c, "Metode pembayaran tidak valid. Pilihan: bank_transfer, e_wallet, virtual_account")
		return
	}

	// ── 1. Ambil & validasi invoice ───────────────────────────────────────────
	var invoice models.Invoice
	if err := config.DB.Where("id = ? AND parent_id = ?", invoiceID, userID).First(&invoice).Error; err != nil {
		utils.NotFound(c, "Invoice tidak ditemukan")
		return
	}
	if invoice.Status != models.PaymentUnpaid {
		utils.BadRequest(c, "Invoice sudah dibayar atau kadaluarsa")
		return
	}
	if time.Now().After(invoice.DueDate) {
		invoice.Status = models.PaymentExpired
		config.DB.Save(&invoice)
		utils.BadRequest(c, "Invoice sudah kadaluarsa, silakan hubungi admin")
		return
	}

	// ── 2. Ambil order + data yang dibutuhkan untuk generate sesi ────────────
	var order models.Order
	if err := config.DB.
		Where("id = ? AND status = ?", invoice.OrderID, models.OrderPending).
		First(&order).Error; err != nil {
		utils.BadRequest(c, "Order tidak ditemukan atau sudah aktif")
		return
	}

	if order.PreferredDays == "" || order.PreferredTime == "" {
		utils.BadRequest(c, "Order tidak memiliki preferensi jadwal. Hubungi admin.")
		return
	}

	// Parse preferred_days: "1,3,5" → []int{1, 3, 5} (0=Minggu, 1=Senin, ..., 6=Sabtu)
	preferredWeekdays, parseErr := parsePreferredDays(order.PreferredDays)
	if parseErr != nil || len(preferredWeekdays) == 0 {
		utils.BadRequest(c, "Format preferred_days di order tidak valid")
		return
	}

	// Parse preferred_time: "09:00" → jam=9, menit=0
	prefHour, prefMinute, parseErr := parsePreferredTime(order.PreferredTime)
	if parseErr != nil {
		utils.BadRequest(c, "Format preferred_time di order tidak valid")
		return
	}

	// ── 3. Hitung semua tanggal sesi ─────────────────────────────────────────
	// Mulai dari besok (H+1) agar tidak ada sesi di hari yang sama dengan pembayaran
	sessionDates := buildSessionDates(
		time.Now().AddDate(0, 0, 1),
		preferredWeekdays,
		order.TotalSessions,
	)

	// ── 4. Cek konflik untuk SEMUA sesi sekaligus — sebelum ada yang disimpan ─
	type conflictInfo struct {
		SessionNumber int    `json:"session_number"`
		Date          string `json:"date"`
		Time          string `json:"time"`
		ConflictingID uint   `json:"conflicting_session_id"`
		FreeFrom      string `json:"free_from"`
	}
	var conflicts []conflictInfo

	for i, date := range sessionDates {
		proposedStart := time.Date(
			date.Year(), date.Month(), date.Day(),
			prefHour, prefMinute, 0, 0, time.UTC,
		)
		ok, conflict, checkErr := CheckMentorConflict(
			order.MentorID,
			proposedStart,
			order.DurationPerSession,
			0, // tidak ada session yang di-exclude — semua harus dicek fresh
		)
		if checkErr != nil {
			utils.InternalError(c, "Gagal memvalidasi jadwal sesi")
			return
		}
		if !ok {
			conflicts = append(conflicts, conflictInfo{
				SessionNumber: i + 1,
				Date:          date.Format("2006-01-02"),
				Time:          proposedStart.Format("15:04"),
				ConflictingID: conflict.SessionID,
				FreeFrom:      conflict.Slot.End.Format("15:04"),
			})
		}
	}

	// Jika ada konflik, tolak seluruh payment — parent harus pilih jadwal lain
	if len(conflicts) > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"message": "Pembayaran ditolak: jadwal yang kamu pilih sudah diambil oleh siswa lain. Silakan pilih hari atau jam yang berbeda.",
			"conflicts": conflicts,
		})
		return
	}

	// ── 5. Semua sesi aman — proses payment + aktifkan order + generate sesi ──
	// Gunakan DB transaction agar atomik: kalau satu gagal, semua di-rollback
	tx := config.DB.Begin()
	if tx.Error != nil {
		utils.InternalError(c, "Gagal memulai transaksi")
		return
	}

	// Tandai invoice paid
	now := time.Now()
	invoice.Status = models.PaymentPaid
	invoice.Method = models.PaymentMethod(input.Method)
	invoice.PaidAt = &now
	if input.ProofURL != "" {
		invoice.ProofURL = &input.ProofURL
	}
	if err := tx.Save(&invoice).Error; err != nil {
		tx.Rollback()
		utils.InternalError(c, "Gagal menyimpan pembayaran")
		return
	}

	// Aktifkan order
	if err := tx.Model(&models.Order{}).
		Where("id = ?", order.ID).
		Update("status", models.OrderActive).Error; err != nil {
		tx.Rollback()
		utils.InternalError(c, "Gagal mengaktifkan order")
		return
	}

	// Generate semua sesi
	sessions := make([]models.Session, len(sessionDates))
	for i, date := range sessionDates {
		proposedStart := time.Date(
			date.Year(), date.Month(), date.Day(),
			prefHour, prefMinute, 0, 0, time.UTC,
		)
		sessions[i] = models.Session{
			OrderID:       order.ID,
			MentorID:      order.MentorID,
			ChildID:       order.ChildID,
			ScheduledAt:   proposedStart,
			Duration:      order.DurationPerSession,
			Status:        models.SessionUpcoming,
			MeetLink:      order.MeetLink,
			SessionNumber: i + 1,
		}
	}
	if err := tx.Create(&sessions).Error; err != nil {
		tx.Rollback()
		utils.InternalError(c, "Gagal membuat jadwal sesi")
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.InternalError(c, "Gagal menyelesaikan transaksi")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Pembayaran berhasil dikonfirmasi",
		"sessions_created": len(sessions),
		"first_session":    sessions[0].ScheduledAt.Format("2006-01-02 15:04"),
		"last_session":     sessions[len(sessions)-1].ScheduledAt.Format("2006-01-02 15:04"),
	})
}

// parsePreferredDays mem-parse string "1,3,5" menjadi slice weekday Go (time.Weekday).
// Nilai valid: 0 (Minggu) sampai 6 (Sabtu). Duplikat diabaikan. Urutan dipertahankan.
func parsePreferredDays(s string) ([]time.Weekday, error) {
	parts := strings.Split(s, ",")
	seen := make(map[time.Weekday]bool)
	var days []time.Weekday
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > 6 {
			return nil, fmt.Errorf("nilai hari tidak valid: %q", p)
		}
		wd := time.Weekday(n)
		if !seen[wd] {
			seen[wd] = true
			days = append(days, wd)
		}
	}
	return days, nil
}

// parsePreferredTime mem-parse string "HH:MM" menjadi jam dan menit.
func parsePreferredTime(s string) (hour, minute int, err error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("format waktu tidak valid")
	}
	h, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	m, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("jam atau menit tidak valid")
	}
	return h, m, nil
}

// buildSessionDates menghasilkan slice tanggal sesi sejumlah totalSessions,
// dimulai dari startFrom, hanya pada hari-hari yang ada di weekdays.
//
// Contoh: weekdays=[1,3] (Senin,Rabu), total=4, mulai 2025-01-20 (Senin)
// → [2025-01-20, 2025-01-22, 2025-01-27, 2025-01-29]
func buildSessionDates(startFrom time.Time, weekdays []time.Weekday, totalSessions int) []time.Time {
	// Normalisasi ke midnight UTC agar konsisten
	cursor := time.Date(startFrom.Year(), startFrom.Month(), startFrom.Day(), 0, 0, 0, 0, time.UTC)

	// Buat set weekday untuk O(1) lookup
	wdSet := make(map[time.Weekday]bool, len(weekdays))
	for _, wd := range weekdays {
		wdSet[wd] = true
	}

	dates := make([]time.Time, 0, totalSessions)
	for len(dates) < totalSessions {
		if wdSet[cursor.Weekday()] {
			dates = append(dates, cursor)
		}
		cursor = cursor.AddDate(0, 0, 1)
	}
	return dates
}

// ─── SCHEDULE ────────────────────────────────────────────────────────────────

func ParentGetSchedule(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var children []models.Child
	config.DB.Where("parent_id = ?", userID).Find(&children)

	var childIDs []uint
	for _, ch := range children {
		childIDs = append(childIDs, ch.ID)
	}

	var sessions []models.Session
	if len(childIDs) > 0 {
		config.DB.Where("child_id IN ? AND status IN ?", childIDs,
			[]models.SessionStatus{models.SessionUpcoming, models.SessionOngoing}).
			Preload("Order.Course").
			Preload("Order.Child").
			Order("scheduled_at ASC").
			Find(&sessions)
	}

	c.JSON(http.StatusOK, gin.H{"schedule": sessions, "total": len(sessions)})
}

// ─── REVIEWS ─────────────────────────────────────────────────────────────────

func ParentSubmitReview(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var input struct {
		OrderID  uint   `json:"order_id"`
		Rating   int    `json:"rating"`
		Comment  string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}
	if input.OrderID == 0 {
		utils.BadRequest(c, "order_id wajib diisi")
		return
	}
	if input.Rating < 1 || input.Rating > 5 {
		utils.BadRequest(c, "Rating harus antara 1 sampai 5")
		return
	}

	// Validasi order milik parent & sudah selesai/aktif
	var order models.Order
	if err := config.DB.
		Where("id = ? AND parent_id = ? AND status IN ?", input.OrderID, userID,
			[]models.OrderStatus{models.OrderActive, models.OrderCompleted}).
		Preload("Course").
		Preload("Package").
		Preload("Child").
		First(&order).Error; err != nil {
		utils.NotFound(c, "Order tidak ditemukan atau belum aktif")
		return
	}

	// Cek sudah review untuk order ini belum
	var existing models.Review
	config.DB.Where("order_id = ? AND parent_id = ?", input.OrderID, userID).First(&existing)
	if existing.ID != 0 {
		utils.Conflict(c, "Kamu sudah memberikan ulasan untuk order ini")
		return
	}

	var mentorUser models.User
	var mp models.MentorProfile
	config.DB.First(&mp, order.MentorID)
	config.DB.First(&mentorUser, mp.UserID)

	review := models.Review{
		OrderID:      order.ID,
		CourseID:     order.CourseID,
		MentorID:     order.MentorID,
		ParentID:     userID,
		ChildID:      order.ChildID,
		PackageID:    order.PackageID,
		Rating:       input.Rating,
		Comment:      input.Comment,
		ReviewerName: order.Child.Name,
		CourseName:   order.Course.Title,
		PackageName:  order.Package.Name,
	}
	if err := config.DB.Create(&review).Error; err != nil {
		utils.InternalError(c, "Gagal menyimpan ulasan")
		return
	}

	// Update rata-rata rating di course & mentor profile
	go func(courseID, mentorProfileID uint) {
		var avg float64
		var count int64
		config.DB.Model(&models.Review{}).Where("course_id = ?", courseID).Count(&count)
		config.DB.Model(&models.Review{}).Where("course_id = ?", courseID).
			Select("AVG(rating)").Scan(&avg)
		config.DB.Model(&models.Course{}).Where("id = ?", courseID).
			Updates(map[string]interface{}{"rating": avg, "total_review": count})

		var mentorAvg float64
		var mentorCount int64
		config.DB.Model(&models.Review{}).Where("mentor_id = ?", mentorProfileID).Count(&mentorCount)
		config.DB.Model(&models.Review{}).Where("mentor_id = ?", mentorProfileID).
			Select("AVG(rating)").Scan(&mentorAvg)
		config.DB.Model(&models.MentorProfile{}).Where("id = ?", mentorProfileID).
			Updates(map[string]interface{}{"rating": mentorAvg, "total_review": mentorCount})
	}(order.CourseID, order.MentorID)

	c.JSON(http.StatusCreated, gin.H{"message": "Ulasan berhasil dikirim", "review": review})
}

func ParentGetReviews(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var reviews []models.Review
	config.DB.Where("parent_id = ?", userID).
		Order("created_at DESC").
		Find(&reviews)

	c.JSON(http.StatusOK, gin.H{"reviews": reviews, "total": len(reviews)})
}