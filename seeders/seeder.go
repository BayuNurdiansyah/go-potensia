package seeders

import (
	"fmt"
	"log"
	"math"
	"time"

	"go-potensia/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedAll menjalankan semua seeder secara berurutan.
// Idempotent: cek apakah data sudah ada sebelum insert.
func SeedAll(db *gorm.DB) {
	log.Println("🌱 Starting database seeding...")

	users, mentorProfiles, parentIDs := seedUsers(db)
	_ = users
	courses, packages := seedCourses(db, mentorProfiles)
	children := seedChildren(db, parentIDs)
	orders, _ := seedOrders(db, parentIDs, children, mentorProfiles, courses, packages)
	sessions := seedSessions(db, orders, mentorProfiles)
	seedEarnings(db, sessions, orders, courses, children, mentorProfiles)
	seedReviews(db, orders, mentorProfiles, parentIDs, children, courses, packages)
	seedSkillProgress(db, orders)
	seedNotifications(db, users)
	seedBankAccounts(db, mentorProfiles)

	log.Println("✅ Seeding completed successfully!")
}

// ─── HELPERS ──────────────────────────────────────────────────────────────────

func hashPassword(pw string) string {
	b, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b)
}

func ptr[T any](v T) *T { return &v }

func addDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

// nextWeekdayFrom returns the next occurrence of weekday >= from
func nextWeekdayFrom(from time.Time, wd time.Weekday) time.Time {
	d := from
	for d.Weekday() != wd {
		d = d.AddDate(0, 0, 1)
	}
	return d
}

// ─── 1. USERS ─────────────────────────────────────────────────────────────────

func seedUsers(db *gorm.DB) ([]models.User, []models.MentorProfile, []uint) {
	log.Println("  → Seeding users...")

	password := hashPassword("Password123!")

	// 3 mentors, 3 parents
	userDefs := []struct {
		name  string
		email string
		phone string
		role  models.Role
	}{
		{"Budi Santoso", "budi.mentor@potensia.id", "628111111111", models.RoleMentor},
		{"Aisyah Putri", "aisyah.mentor@potensia.id", "628222222222", models.RoleMentor},
		{"Rizky Prasetyo", "rizky.mentor@potensia.id", "628333333333", models.RoleMentor},
		{"Siti Aminah", "siti.parent@potensia.id", "628444444444", models.RoleParent},
		{"Ahmad Fauzi", "ahmad.parent@potensia.id", "628555555555", models.RoleParent},
		{"Dewi Lestari", "dewi.parent@potensia.id", "628666666666", models.RoleParent},
	}

	var createdUsers []models.User
	for _, u := range userDefs {
		var existing models.User
		if db.Where("email = ?", u.email).First(&existing).Error == nil {
			createdUsers = append(createdUsers, existing)
			continue
		}
		user := models.User{
			Name:       u.name,
			Email:      u.email,
			Phone:      u.phone,
			Password:   password,
			Role:       u.role,
			IsVerified: true,
			IsActive:   true,
		}
		db.Create(&user)
		createdUsers = append(createdUsers, user)
	}

	// Mentor profiles
	mentorDefs := []struct {
		idx       int
		expertise string
		bio       string
		province  string
		regency   string
	}{
		{0, "Matematika, IPA", "Lulusan Matematika UGM dengan pengalaman mengajar 5 tahun.", "DI Yogyakarta", "Kota Yogyakarta"},
		{1, "Bahasa Inggris, TOEFL", "Native speaker level, berpengalaman mengajar anak-anak usia dini.", "DI Yogyakarta", "Sleman"},
		{2, "Pemrograman, Robotika", "Software engineer 7 tahun, kini fokus edukasi anak di bidang teknologi.", "Jawa Tengah", "Kota Semarang"},
	}

	var mentorProfiles []models.MentorProfile
	for _, m := range mentorDefs {
		user := createdUsers[m.idx]
		var existing models.MentorProfile
		if db.Where("user_id = ?", user.ID).First(&existing).Error == nil {
			mentorProfiles = append(mentorProfiles, existing)
			continue
		}
		mp := models.MentorProfile{
			UserID:            user.ID,
			Expertise:         m.expertise,
			Bio:               m.bio,
			Province:          m.province,
			Regency:           m.regency,
			Rating:            4.8,
			TotalReview:       12,
			TotalStudents:     5,
			TotalSessions:     40,
			CompletedSessions: 38,
			TotalEarnings:     2700000,
		}
		db.Create(&mp)

		// Certificates
		db.Create(&models.MentorCertificate{
			MentorID: mp.ID,
			Title:    fmt.Sprintf("Sertifikasi Pendidik %s", m.expertise),
			Issuer:   "Kemendikbud RI",
			Year:     "2022",
		})
		// Education
		db.Create(&models.MentorEducation{
			MentorID:    mp.ID,
			Degree:      "S1",
			Institution: "Universitas Gadjah Mada",
			Year:        "2019",
		})
		// Achievement
		db.Create(&models.MentorAchievement{
			MentorID: mp.ID,
			Text:     "Top 10 Mentor Terbaik Potensia 2024",
		})

		mentorProfiles = append(mentorProfiles, mp)
	}

	// Parent profiles
	for i := 3; i < 6; i++ {
		user := createdUsers[i]
		var existing models.ParentProfile
		if db.Where("user_id = ?", user.ID).First(&existing).Error != nil {
			db.Create(&models.ParentProfile{
				UserID:  user.ID,
				Address: "Jl. Kaliurang No. 10, Yogyakarta",
			})
		}
	}

	parentIDs := []uint{createdUsers[3].ID, createdUsers[4].ID, createdUsers[5].ID}
	return createdUsers, mentorProfiles, parentIDs
}

// ─── 2. COURSES ───────────────────────────────────────────────────────────────

func seedCourses(db *gorm.DB, mentors []models.MentorProfile) ([]models.Course, []models.CoursePackage) {
	log.Println("  → Seeding courses...")

	courseDefs := []struct {
		mentorIdx   int
		title       string
		category    string
		description string
		competencies []string
		packages    []struct {
			name     string
			duration int
			sessions int
			price    int64
			orig     *int64
			highlight bool
		}
	}{
		{
			mentorIdx:   0,
			title:       "Matematika Dasar SD",
			category:    "Matematika",
			description: "Kursus matematika dasar untuk siswa SD kelas 1–6. Materi mencakup operasi hitung, pecahan, geometri dasar, dan pemecahan masalah.",
			competencies: []string{"Operasi Hitung Dasar", "Pecahan & Desimal", "Geometri Dasar", "Pemecahan Masalah"},
			packages: []struct {
				name     string
				duration int
				sessions int
				price    int64
				orig     *int64
				highlight bool
			}{
				{"Starter", 45, 4, 240000, ptr[int64](300000), false},
				{"Reguler", 60, 8, 440000, ptr[int64](560000), true},
				{"Intensif", 90, 12, 600000, ptr[int64](780000), false},
			},
		},
		{
			mentorIdx:   0,
			title:       "IPA Terpadu SMP",
			category:    "IPA",
			description: "Kursus IPA untuk SMP, mencakup fisika, kimia, dan biologi dasar.",
			competencies: []string{"Fisika Dasar", "Kimia Dasar", "Biologi Sel", "Ekosistem"},
			packages: []struct {
				name     string
				duration int
				sessions int
				price    int64
				orig     *int64
				highlight bool
			}{
				{"Reguler", 60, 8, 480000, ptr[int64](600000), true},
				{"Intensif", 90, 12, 660000, ptr[int64](840000), false},
			},
		},
		{
			mentorIdx:   1,
			title:       "English Kids",
			category:    "Bahasa Inggris",
			description: "Kursus bahasa Inggris interaktif untuk anak usia 4–12 tahun. Belajar lewat lagu, cerita, dan permainan.",
			competencies: []string{"Vocabulary Building", "Basic Conversation", "Reading & Writing", "Listening Skills"},
			packages: []struct {
				name     string
				duration int
				sessions int
				price    int64
				orig     *int64
				highlight bool
			}{
				{"Starter", 45, 4, 200000, ptr[int64](260000), false},
				{"Reguler", 60, 8, 380000, ptr[int64](480000), true},
				{"Intensif", 90, 12, 540000, ptr[int64](720000), false},
			},
		},
		{
			mentorIdx:   2,
			title:       "Coding Anak",
			category:    "Teknologi",
			description: "Pengenalan pemrograman dan logika komputasi untuk anak usia 8–15 tahun menggunakan Scratch dan Python dasar.",
			competencies: []string{"Logika Pemrograman", "Scratch Dasar", "Python Fundamentals", "Mini Project"},
			packages: []struct {
				name     string
				duration int
				sessions int
				price    int64
				orig     *int64
				highlight bool
			}{
				{"Reguler", 60, 8, 560000, ptr[int64](700000), true},
				{"Intensif", 90, 16, 960000, ptr[int64](1280000), false},
			},
		},
	}

	var allCourses []models.Course
	var allPackages []models.CoursePackage

	for _, cd := range courseDefs {
		mentor := mentors[cd.mentorIdx]
		var existing models.Course
		if db.Where("mentor_id = ? AND title = ?", mentor.ID, cd.title).First(&existing).Error == nil {
			allCourses = append(allCourses, existing)
			var pkgs []models.CoursePackage
			db.Where("course_id = ?", existing.ID).Find(&pkgs)
			allPackages = append(allPackages, pkgs...)
			continue
		}

		course := models.Course{
			MentorID:    mentor.ID,
			Title:       cd.title,
			Category:    cd.category,
			Description: cd.description,
			Status:      models.CourseStatusActive,
			Rating:      4.7,
			TotalReview: 8,
		}
		db.Create(&course)

		for i, comp := range cd.competencies {
			db.Create(&models.CourseCompetency{CourseID: course.ID, Text: comp, SortOrder: i})
		}

		for _, pkg := range cd.packages {
			p := models.CoursePackage{
				CourseID:           course.ID,
				Name:               pkg.name,
				DurationPerSession: pkg.duration,
				TotalSessions:      pkg.sessions,
				Price:              pkg.price,
				OriginalPrice:      pkg.orig,
				IsHighlight:        pkg.highlight,
				IsActive:           true,
			}
			db.Create(&p)
			allPackages = append(allPackages, p)
		}

		allCourses = append(allCourses, course)
	}

	return allCourses, allPackages
}

// ─── 3. CHILDREN ──────────────────────────────────────────────────────────────

func seedChildren(db *gorm.DB, parentIDs []uint) []models.Child {
	log.Println("  → Seeding children...")

	childDefs := []struct {
		parentIdx int
		name      string
		education string
		gender    models.Gender
		birthYear int
	}{
		{0, "Alvaro Septian", "SD Kelas 4", models.GenderMale, 2015},
		{0, "Siti Aminah Junior", "PAUD", models.GenderFemale, 2020},
		{1, "Bima Nugraha", "SMP Kelas 1", models.GenderMale, 2012},
		{1, "Rara Setiawan", "SD Kelas 2", models.GenderFemale, 2017},
		{2, "Zara Putri", "TK B", models.GenderFemale, 2019},
	}

	var children []models.Child
	for _, cd := range childDefs {
		parentID := parentIDs[cd.parentIdx]
		var existing models.Child
		if db.Where("parent_id = ? AND name = ?", parentID, cd.name).First(&existing).Error == nil {
			children = append(children, existing)
			continue
		}
		child := models.Child{
			ParentID:  parentID,
			Name:      cd.name,
			BirthDate: time.Date(cd.birthYear, time.January, 15, 0, 0, 0, 0, time.UTC),
			Gender:    cd.gender,
			Education: cd.education,
		}
		db.Create(&child)
		children = append(children, child)
	}
	return children
}

// ─── 4. ORDERS + INVOICES ─────────────────────────────────────────────────────

func seedOrders(db *gorm.DB, parentIDs []uint, children []models.Child,
	mentors []models.MentorProfile, courses []models.Course, packages []models.CoursePackage,
) ([]models.Order, []models.Invoice) {

	log.Println("  → Seeding orders & invoices...")

	// Cari package by course title and package name
	findPkg := func(courseID uint, pkgName string) models.CoursePackage {
		for _, p := range packages {
			if p.CourseID == courseID && p.Name == pkgName {
				return p
			}
		}
		return models.CoursePackage{}
	}
	findCourse := func(title string) models.Course {
		for _, c := range courses {
			if c.Title == title {
				return c
			}
		}
		return models.Course{}
	}

	type orderDef struct {
		parentID   uint
		childID    uint
		courseName string
		pkgName    string
		mentorIdx  int
		status     models.OrderStatus
		daysAgo    int // order dibuat N hari lalu
		days       string
		prefTime   string
	}

	orderDefs := []orderDef{
		// Alvaro (child[0]) — Matematika Dasar, Reguler (8 sesi), ACTIVE
		{parentIDs[0], children[0].ID, "Matematika Dasar SD", "Reguler", 0, models.OrderActive, 30, "1,3", "09:00"},
		// Siti Aminah Junior (child[1]) — English Kids, Starter (4 sesi), COMPLETED
		{parentIDs[0], children[1].ID, "English Kids", "Starter", 1, models.OrderCompleted, 60, "2,4", "10:00"},
		// Bima (child[2]) — IPA Terpadu, Intensif (12 sesi), ACTIVE
		{parentIDs[1], children[2].ID, "IPA Terpadu SMP", "Intensif", 0, models.OrderActive, 20, "1,4", "14:00"},
		// Rara (child[3]) — English Kids, Reguler (8 sesi), ACTIVE
		{parentIDs[1], children[3].ID, "English Kids", "Reguler", 1, models.OrderActive, 15, "3,5", "08:00"},
		// Zara (child[4]) — Coding Anak, Reguler (8 sesi), PENDING (belum bayar)
		{parentIDs[2], children[4].ID, "Coding Anak", "Reguler", 2, models.OrderPending, 5, "2,5", "15:00"},
	}

	var allOrders []models.Order
	var allInvoices []models.Invoice

	for _, od := range orderDefs {
		course := findCourse(od.courseName)
		pkg := findPkg(course.ID, od.pkgName)
		if pkg.ID == 0 {
			log.Printf("    ⚠ Package not found: %s / %s", od.courseName, od.pkgName)
			continue
		}
		mentor := mentors[od.mentorIdx]

		var existingOrder models.Order
		if db.Where("parent_id = ? AND child_id = ? AND course_id = ?", od.parentID, od.childID, course.ID).
			First(&existingOrder).Error == nil {
			allOrders = append(allOrders, existingOrder)
			var inv models.Invoice
			db.Where("order_id = ?", existingOrder.ID).First(&inv)
			if inv.ID != 0 {
				allInvoices = append(allInvoices, inv)
			}
			continue
		}

		completedSess := 0
		remainSess := pkg.TotalSessions
		if od.status == models.OrderActive {
			completedSess = pkg.TotalSessions / 2
			remainSess = pkg.TotalSessions - completedSess
		} else if od.status == models.OrderCompleted {
			completedSess = pkg.TotalSessions
			remainSess = 0
		}

		order := models.Order{
			ParentID:           od.parentID,
			ChildID:            od.childID,
			CourseID:           course.ID,
			PackageID:          pkg.ID,
			MentorID:           mentor.ID,
			TotalSessions:      pkg.TotalSessions,
			CompletedSessions:  completedSess,
			RemainingSessions:  remainSess,
			DurationPerSession: pkg.DurationPerSession,
			TotalPrice:         pkg.Price,
			Status:             od.status,
			PreferredDays:      od.days,
			PreferredTime:      od.prefTime,
			MeetLink:           "https://meet.google.com/abc-defg-hij",
			Notes:              pkg.Name,
		}
		order.CreatedAt = time.Now().AddDate(0, 0, -od.daysAgo)
		db.Create(&order)
		allOrders = append(allOrders, order)

		// Invoice
		invoiceStatus := models.PaymentPaid
		var paidAt *time.Time
		if od.status == models.OrderPending {
			invoiceStatus = models.PaymentUnpaid
		} else {
			t := order.CreatedAt.Add(time.Hour * 2)
			paidAt = &t
		}

		invoice := models.Invoice{
			OrderID:     order.ID,
			ParentID:    od.parentID,
			Amount:      pkg.Price,
			Description: fmt.Sprintf("Pembayaran kursus %s – %s", course.Title, pkg.Name),
			Period:      time.Now().Format("January 2006"),
			Status:      invoiceStatus,
			DueDate:     order.CreatedAt.AddDate(0, 0, 3),
			PaidAt:      paidAt,
			Method:      models.PaymentMethodBankTransfer,
		}
		db.Create(&invoice)
		allInvoices = append(allInvoices, invoice)
	}

	return allOrders, allInvoices
}

// ─── 5. SESSIONS ──────────────────────────────────────────────────────────────

func seedSessions(db *gorm.DB, orders []models.Order, mentors []models.MentorProfile) []models.Session {
	log.Println("  → Seeding sessions...")

	var allSessions []models.Session

	for _, order := range orders {
		if order.Status == models.OrderPending {
			continue // sesi belum dibuat karena belum bayar
		}

		// Cek apakah sessions sudah ada
		var count int64
		db.Model(&models.Session{}).Where("order_id = ?", order.ID).Count(&count)
		if count > 0 {
			var existing []models.Session
			db.Where("order_id = ?", order.ID).Order("session_number ASC").Find(&existing)
			allSessions = append(allSessions, existing...)
			continue
		}

		// Generate sesi dari preferred_days + preferred_time
		weekdays := parseWeekdays(order.PreferredDays)
		hour, min := parseTime(order.PreferredTime)

		startFrom := order.CreatedAt.AddDate(0, 0, 1)
		dates := buildDates(startFrom, weekdays, order.TotalSessions)

		for i, d := range dates {
			scheduledAt := time.Date(d.Year(), d.Month(), d.Day(), hour, min, 0, 0, time.UTC)

			status := models.SessionUpcoming
			var completedAt *time.Time
			var startedAt *time.Time
			topic := ""
			notes := ""

			if i < order.CompletedSessions {
				status = models.SessionCompleted
				t1 := scheduledAt
				t2 := scheduledAt.Add(time.Duration(order.DurationPerSession) * time.Minute)
				startedAt = &t1
				completedAt = &t2
				topic = fmt.Sprintf("Materi sesi ke-%d", i+1)
				notes = "Siswa menunjukkan perkembangan yang baik."
			}

			sess := models.Session{
				OrderID:       order.ID,
				MentorID:      order.MentorID,
				ChildID:       order.ChildID,
				ScheduledAt:   scheduledAt,
				Duration:      order.DurationPerSession,
				Status:        status,
				MeetLink:      order.MeetLink,
				SessionNumber: i + 1,
				Topic:         topic,
				Notes:         notes,
				StartedAt:     startedAt,
				CompletedAt:   completedAt,
			}
			db.Create(&sess)
			allSessions = append(allSessions, sess)
		}
	}

	return allSessions
}

// ─── 6. EARNINGS ──────────────────────────────────────────────────────────────

func seedEarnings(db *gorm.DB, sessions []models.Session, orders []models.Order,
	courses []models.Course, children []models.Child, mentors []models.MentorProfile,
) {
	log.Println("  → Seeding earnings...")

	// Build lookup maps
	orderMap := map[uint]models.Order{}
	for _, o := range orders {
		orderMap[o.ID] = o
	}
	courseMap := map[uint]models.Course{}
	for _, c := range courses {
		courseMap[c.ID] = c
	}
	childMap := map[uint]models.Child{}
	for _, ch := range children {
		childMap[ch.ID] = ch
	}

	const feeRate = 0.10

	for _, sess := range sessions {
		if sess.Status != models.SessionCompleted {
			continue
		}

		var existing models.MentorEarning
		if db.Where("session_id = ?", sess.ID).First(&existing).Error == nil {
			continue // already seeded
		}

		order := orderMap[sess.OrderID]
		course := courseMap[order.CourseID]
		child := childMap[sess.ChildID]

		gross := int64(0)
		if order.TotalSessions > 0 {
			gross = order.TotalPrice / int64(order.TotalSessions)
		}
		fee := int64(math.Round(float64(gross) * feeRate))
		net := gross - fee

		earnedAt := sess.ScheduledAt.Add(time.Duration(order.DurationPerSession) * time.Minute)
		if sess.CompletedAt != nil {
			earnedAt = *sess.CompletedAt
		}

		// Determine which withdrawal week this belongs to
		weekday := int(earnedAt.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		weekStart := time.Date(earnedAt.Year(), earnedAt.Month(), earnedAt.Day()-weekday+1, 0, 0, 0, 0, time.UTC)
		weekEnd := weekStart.AddDate(0, 0, 7).Add(-time.Second)

		// Upsert weekly withdrawal record
		var withdrawal models.MentorWithdrawal
		if db.Where("mentor_id = ? AND period_start = ?", sess.MentorID, weekStart).
			First(&withdrawal).Error != nil {
			// Get bank info
			var bank models.MentorBankAccount
			db.Where("mentor_id = ?", sess.MentorID).First(&bank)
			bankName := "BCA"
			accNum := "1234567890"
			accHolder := "MENTOR"
			if bank.ID != 0 {
				bankName = bank.BankName
				accNum = bank.AccountNumber
				accHolder = bank.AccountHolder
			}
			processedAt := weekEnd.Add(24 * time.Hour)
			withdrawal = models.MentorWithdrawal{
				MentorID:      sess.MentorID,
				PeriodStart:   weekStart,
				PeriodEnd:     weekEnd,
				GrossAmount:   0,
				FeeAmount:     0,
				NetAmount:     0,
				BankName:      bankName,
				AccountNumber: accNum,
				AccountHolder: accHolder,
				Status:        models.WithdrawalProcessed,
				ProcessedAt:   &processedAt,
			}
			db.Create(&withdrawal)
		}

		earning := models.MentorEarning{
			MentorID:      sess.MentorID,
			SessionID:     sess.ID,
			OrderID:       order.ID,
			ChildID:       child.ID,
			CourseID:      course.ID,
			CourseName:    course.Title,
			StudentName:   child.Name,
			PackageName:   order.Notes,
			SessionNumber: sess.SessionNumber,
			GrossAmount:   gross,
			FeeRate:       feeRate,
			FeeAmount:     fee,
			NetAmount:     net,
			EarnedAt:      earnedAt,
			WithdrawalID:  &withdrawal.ID,
		}
		db.Create(&earning)

		// Update withdrawal totals
		db.Model(&withdrawal).Updates(map[string]interface{}{
			"gross_amount": gorm.Expr("gross_amount + ?", gross),
			"fee_amount":   gorm.Expr("fee_amount + ?", fee),
			"net_amount":   gorm.Expr("net_amount + ?", net),
		})
	}
}

// ─── 7. REVIEWS ───────────────────────────────────────────────────────────────

func seedReviews(db *gorm.DB, orders []models.Order, mentors []models.MentorProfile,
	parentIDs []uint, children []models.Child, courses []models.Course, packages []models.CoursePackage,
) {
	log.Println("  → Seeding reviews...")

	courseMap := map[uint]models.Course{}
	for _, c := range courses {
		courseMap[c.ID] = c
	}
	pkgMap := map[uint]models.CoursePackage{}
	for _, p := range packages {
		pkgMap[p.ID] = p
	}
	childMap := map[uint]models.Child{}
	for _, ch := range children {
		childMap[ch.ID] = ch
	}

	comments := []string{
		"Mentor sangat sabar dan mudah dipahami. Anak saya jadi lebih percaya diri.",
		"Penjelasannya detail dan menyenangkan. Highly recommended!",
		"Materi terstruktur dengan baik, anak saya bisa mengikuti dengan baik.",
		"Sangat profesional dan responsif. Terima kasih kak!",
	}

	ci := 0
	for _, order := range orders {
		if order.Status != models.OrderCompleted {
			continue
		}
		var existing models.Review
		if db.Where("order_id = ?", order.ID).First(&existing).Error == nil {
			continue
		}

		course := courseMap[order.CourseID]
		pkg := pkgMap[order.PackageID]
		child := childMap[order.ChildID]

		// Find parent name
		var parentUser models.User
		db.First(&parentUser, order.ParentID)

		review := models.Review{
			OrderID:      order.ID,
			CourseID:     order.CourseID,
			MentorID:     order.MentorID,
			ParentID:     order.ParentID,
			ChildID:      order.ChildID,
			PackageID:    order.PackageID,
			Rating:       5,
			Comment:      comments[ci%len(comments)],
			ReviewerName: parentUser.Name,
			CourseName:   course.Title,
			PackageName:  pkg.Name,
		}
		_ = child
		db.Create(&review)

		// Update mentor rating cache
		var avg struct{ Avg float64; Count int64 }
		db.Model(&models.Review{}).
			Select("COALESCE(AVG(rating),0) as avg, COUNT(*) as count").
			Where("mentor_id = ?", order.MentorID).
			Scan(&avg)
		db.Model(&models.MentorProfile{}).
			Where("id = ?", order.MentorID).
			Updates(map[string]interface{}{
				"rating":       avg.Avg,
				"total_review": avg.Count,
			})
		// Update course rating
		db.Model(&models.Course{}).
			Where("id = ?", order.CourseID).
			Updates(map[string]interface{}{
				"rating":       avg.Avg,
				"total_review": avg.Count,
			})

		ci++
	}
}

// ─── 8. SKILL PROGRESS ────────────────────────────────────────────────────────

func seedSkillProgress(db *gorm.DB, orders []models.Order) {
	log.Println("  → Seeding skill progress...")

	skillsByCourse := map[string][]string{
		"Matematika Dasar SD": {"Operasi Hitung Dasar", "Pecahan & Desimal", "Geometri Dasar", "Pemecahan Masalah"},
		"IPA Terpadu SMP":     {"Fisika Dasar", "Kimia Dasar", "Biologi Sel", "Ekosistem"},
		"English Kids":        {"Vocabulary Building", "Basic Conversation", "Reading & Writing", "Listening Skills"},
		"Coding Anak":         {"Logika Pemrograman", "Scratch Dasar", "Python Fundamentals", "Mini Project"},
	}

	for _, order := range orders {
		if order.Status == models.OrderPending {
			continue
		}

		var course models.Course
		db.First(&course, order.CourseID)

		skills, ok := skillsByCourse[course.Title]
		if !ok {
			continue
		}

		progressPct := 0
		if order.TotalSessions > 0 {
			progressPct = int(math.Round(float64(order.CompletedSessions) / float64(order.TotalSessions) * 100))
		}

		for i, skill := range skills {
			var existing models.SkillProgress
			if db.Where("order_id = ? AND skill_name = ?", order.ID, skill).First(&existing).Error == nil {
				continue
			}

			// Skills have slightly different progress (stagger them)
			pct := progressPct
			if i == 0 {
				pct = min(100, progressPct+10)
			} else if i == len(skills)-1 {
				pct = max(0, progressPct-20)
			}

			db.Create(&models.SkillProgress{
				OrderID:   order.ID,
				ChildID:   order.ChildID,
				CourseID:  order.CourseID,
				SkillName: skill,
				Progress:  pct,
			})
		}
	}
}

// ─── 9. NOTIFICATIONS ─────────────────────────────────────────────────────────

func seedNotifications(db *gorm.DB, users []models.User) {
	log.Println("  → Seeding notifications...")

	notifDefs := []struct {
		userIdx int
		ntype   string
		title   string
		body    string
	}{
		{0, "reminder", "Sesi Hari Ini", "Kamu punya sesi bersama Alvaro Septian jam 09:00. Jangan lupa bergabung!"},
		{0, "info", "Pembayaran Diterima", "Pembayaran untuk kursus Matematika Dasar SD telah dikonfirmasi."},
		{3, "reminder", "Jadwal Belajar Hari Ini", "Alvaro punya sesi Matematika Dasar jam 09:00 hari ini."},
		{3, "info", "Invoice Berhasil Dibayar", "Terima kasih! Pembayaran kursus Matematika Dasar telah berhasil."},
		{4, "reminder", "Sesi Bima Hari Ini", "Bima punya sesi IPA Terpadu jam 14:00 hari ini."},
		{1, "info", "Review Baru", "Kamu mendapat ulasan bintang 5 dari Siti Aminah."},
	}

	for _, nd := range notifDefs {
		if nd.userIdx >= len(users) {
			continue
		}
		user := users[nd.userIdx]
		var count int64
		db.Model(&models.Notification{}).
			Where("user_id = ? AND title = ?", user.ID, nd.title).
			Count(&count)
		if count > 0 {
			continue
		}
		db.Create(&models.Notification{
			UserID: user.ID,
			Type:   nd.ntype,
			Title:  nd.title,
			Body:   nd.body,
			IsRead: false,
		})
	}
}

// ─── 10. BANK ACCOUNTS ────────────────────────────────────────────────────────

func seedBankAccounts(db *gorm.DB, mentors []models.MentorProfile) {
	log.Println("  → Seeding bank accounts...")

	bankDefs := []struct {
		mentorIdx     int
		bankName      string
		accountNumber string
		accountHolder string
		day           models.WithdrawalDay
	}{
		{0, "BCA", "1234567890", "BUDI SANTOSO", models.WithdrawalDayJumat},
		{1, "BNI", "9876543210", "AISYAH PUTRI", models.WithdrawalDayKamis},
		{2, "Mandiri", "1122334455", "RIZKY PRASETYO", models.WithdrawalDayJumat},
	}

	for _, bd := range bankDefs {
		if bd.mentorIdx >= len(mentors) {
			continue
		}
		mentor := mentors[bd.mentorIdx]
		var existing models.MentorBankAccount
		if db.Where("mentor_id = ?", mentor.ID).First(&existing).Error == nil {
			continue
		}
		db.Create(&models.MentorBankAccount{
			MentorID:      mentor.ID,
			BankName:      bd.bankName,
			AccountNumber: bd.accountNumber,
			AccountHolder: bd.accountHolder,
			WithdrawalDay: bd.day,
			IsVerified:    true,
		})
		// Sync ke mentor profile
		db.Model(&models.MentorProfile{}).Where("id = ?", mentor.ID).Updates(map[string]interface{}{
			"bank_name":         bd.bankName,
			"bank_account":      bd.accountNumber,
			"bank_account_name": bd.accountHolder,
		})
	}
}

// ─── UTILS ────────────────────────────────────────────────────────────────────

func parseWeekdays(s string) []time.Weekday {
	days := map[string]time.Weekday{
		"0": time.Sunday, "1": time.Monday, "2": time.Tuesday,
		"3": time.Wednesday, "4": time.Thursday, "5": time.Friday, "6": time.Saturday,
	}
	var result []time.Weekday
	for i := 0; i < len(s); i++ {
		c := string(s[i])
		if wd, ok := days[c]; ok {
			result = append(result, wd)
		}
	}
	return result
}

func parseTime(s string) (hour, min int) {
	if len(s) >= 5 {
		fmt.Sscanf(s, "%d:%d", &hour, &min)
	}
	return
}

func buildDates(startFrom time.Time, weekdays []time.Weekday, total int) []time.Time {
	if len(weekdays) == 0 {
		weekdays = []time.Weekday{time.Monday}
	}
	var result []time.Time
	d := startFrom
	for len(result) < total {
		for _, wd := range weekdays {
			if len(result) >= total {
				break
			}
			next := nextWeekdayFrom(d, wd)
			result = append(result, next)
			d = next.AddDate(0, 0, 1)
		}
	}
	// Sort by date
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Before(result[i]) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result[:total]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}