package controllers

import (
	"net/http"
	"strconv"

	"go-potensia/config"
	"go-potensia/models"
	"go-potensia/utils"

	"github.com/gin-gonic/gin"
)

// ─── PUBLIC: Cari mentor & kursus ────────────────────────────────────────────

func SearchMentors(c *gin.Context) {
	province  := c.Query("province")
	regency   := c.Query("regency")
	district  := c.Query("district")
	category  := c.Query("category")
	search    := c.Query("search")
	pageStr   := c.DefaultQuery("page", "1")
	limitStr  := c.DefaultQuery("limit", "20")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 { page = 1 }
	if limit < 1 || limit > 100 { limit = 20 }
	offset := (page - 1) * limit

	// Join mentor_profiles → users → courses
	query := config.DB.Model(&models.MentorProfile{}).
		Joins("JOIN users ON users.id = mentor_profiles.user_id").
		Where("users.is_verified = ? AND users.is_active = ?", true, true).
		Preload("User")

	if province != "" {
		query = query.Where("mentor_profiles.province = ?", province)
	}
	if regency != "" {
		query = query.Where("mentor_profiles.regency = ?", regency)
	}
	if district != "" {
		query = query.Where("mentor_profiles.district = ?", district)
	}
	if search != "" {
		query = query.Where("users.name ILIKE ?", "%"+search+"%")
	}

	// Filter by category: mentor harus punya course aktif dengan kategori tersebut
	if category != "" {
		query = query.
			Joins("JOIN courses ON courses.mentor_id = mentor_profiles.id AND courses.status = 'active' AND courses.category = ?", category).
			Group("mentor_profiles.id")
	}

	var total int64
	query.Count(&total)

	var mentors []models.MentorProfile
	query.Offset(offset).Limit(limit).Order("mentor_profiles.rating DESC").Find(&mentors)

	c.JSON(http.StatusOK, gin.H{
		"mentors": mentors,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

func GetMentorPublicProfile(c *gin.Context) {
	mentorIDStr := c.Param("mentor_id")
	mentorID, err := strconv.ParseUint(mentorIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "mentor_id tidak valid")
		return
	}

	var mentor models.MentorProfile
	if err := config.DB.
		Where("id = ?", mentorID).
		Preload("User").
		First(&mentor).Error; err != nil {
		utils.NotFound(c, "Mentor tidak ditemukan")
		return
	}

	var certs []models.MentorCertificate
	var achievements []models.MentorAchievement
	var gallery []models.MentorGallery
	var education []models.MentorEducation
	config.DB.Where("mentor_id = ?", mentor.ID).Find(&certs)
	config.DB.Where("mentor_id = ?", mentor.ID).Find(&achievements)
	config.DB.Where("mentor_id = ?", mentor.ID).Order("sort_order ASC").Find(&gallery)
	config.DB.Where("mentor_id = ?", mentor.ID).Find(&education)

	var courses []models.Course
	config.DB.Where("mentor_id = ? AND status = ?", mentor.ID, models.CourseStatusActive).
		Preload("Competencies").
		Preload("Packages").
		Find(&courses)

	var reviews []models.Review
	config.DB.Where("mentor_id = ?", mentor.ID).
		Order("created_at DESC").
		Limit(10).
		Find(&reviews)

	c.JSON(http.StatusOK, gin.H{
		"mentor": gin.H{
			"id":             mentor.ID,
			"name":           mentor.User.Name,
			"avatar":         mentor.User.AvatarURL,
			"expertise":      mentor.Expertise,
			"bio":            mentor.Bio,
			"rating":         mentor.Rating,
			"total_review":   mentor.TotalReview,
			"total_students": mentor.TotalStudents,
			"total_sessions": mentor.TotalSessions,
			"province":       mentor.Province,
			"regency":        mentor.Regency,
			"district":       mentor.District,
		},
		"certificates": certs,
		"achievements": achievements,
		"gallery":      gallery,
		"education":    education,
		"courses":      courses,
		"reviews":      reviews,
	})
}

// ─── MENTOR: Kelola kursus ────────────────────────────────────────────────────

func MentorGetCourses(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var mentor models.MentorProfile
	config.DB.Where("user_id = ?", userID).First(&mentor)

	var courses []models.Course
	config.DB.Where("mentor_id = ?", mentor.ID).
		Preload("Packages").
		Order("created_at DESC").
		Find(&courses)

	c.JSON(http.StatusOK, gin.H{"courses": courses, "total": len(courses)})
}

func MentorGetCourse(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	courseIDStr := c.Param("course_id")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "course_id tidak valid")
		return
	}

	var mentor models.MentorProfile
	config.DB.Where("user_id = ?", userID).First(&mentor)

	var course models.Course
	if err := config.DB.
		Where("id = ? AND mentor_id = ?", courseID, mentor.ID).
		Preload("Competencies").
		Preload("Packages").
		First(&course).Error; err != nil {
		utils.NotFound(c, "Kursus tidak ditemukan")
		return
	}

	var certs []models.MentorCertificate
	var achievements []models.MentorAchievement
	var gallery []models.MentorGallery
	config.DB.Where("mentor_id = ?", mentor.ID).Find(&certs)
	config.DB.Where("mentor_id = ?", mentor.ID).Find(&achievements)
	config.DB.Where("mentor_id = ?", mentor.ID).Order("sort_order ASC").Find(&gallery)

	c.JSON(http.StatusOK, gin.H{
		"course":       course,
		"certificates": certs,
		"achievements": achievements,
		"gallery":      gallery,
	})
}

func MentorCreateCourse(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var input struct {
		Title        string   `json:"title"`
		Category     string   `json:"category"`
		Description  string   `json:"description"`
		Status       string   `json:"status"`
		Competencies []string `json:"competencies"`
		Packages     []struct {
			Name               string `json:"name"`
			DurationPerSession int    `json:"duration_per_session"`
			TotalSessions      int    `json:"total_sessions"`
			Price              int64  `json:"price"`
			OriginalPrice      *int64 `json:"original_price"`
			IsHighlight        bool   `json:"is_highlight"`
		} `json:"packages"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}
	if input.Title == "" || input.Category == "" {
		utils.BadRequest(c, "Judul dan kategori wajib diisi")
		return
	}

	var mentor models.MentorProfile
	config.DB.Where("user_id = ?", userID).First(&mentor)

	status := models.CourseStatusDraft
	if input.Status == string(models.CourseStatusActive) {
		status = models.CourseStatusActive
	}

	course := models.Course{
		MentorID:    mentor.ID,
		Title:       input.Title,
		Category:    input.Category,
		Description: input.Description,
		Status:      status,
	}
	if err := config.DB.Create(&course).Error; err != nil {
		utils.InternalError(c, "Gagal menyimpan kursus")
		return
	}

	for i, comp := range input.Competencies {
		config.DB.Create(&models.CourseCompetency{CourseID: course.ID, Text: comp, SortOrder: i})
	}
	for _, pkg := range input.Packages {
		config.DB.Create(&models.CoursePackage{
			CourseID:           course.ID,
			Name:               pkg.Name,
			DurationPerSession: pkg.DurationPerSession,
			TotalSessions:      pkg.TotalSessions,
			Price:              pkg.Price,
			OriginalPrice:      pkg.OriginalPrice,
			IsHighlight:        pkg.IsHighlight,
		})
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Kursus berhasil dibuat", "course_id": course.ID})
}

func MentorUpdateCourse(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	courseIDStr := c.Param("course_id")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "course_id tidak valid")
		return
	}

	var input struct {
		Title        string   `json:"title"`
		Category     string   `json:"category"`
		Description  string   `json:"description"`
		Status       string   `json:"status"`
		Competencies []string `json:"competencies"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Format request tidak valid")
		return
	}

	var mentor models.MentorProfile
	config.DB.Where("user_id = ?", userID).First(&mentor)

	var course models.Course
	if err := config.DB.Where("id = ? AND mentor_id = ?", courseID, mentor.ID).First(&course).Error; err != nil {
		utils.NotFound(c, "Kursus tidak ditemukan")
		return
	}

	if input.Title != "" { course.Title = input.Title }
	if input.Category != "" { course.Category = input.Category }
	if input.Description != "" { course.Description = input.Description }
	if input.Status == string(models.CourseStatusActive) || input.Status == string(models.CourseStatusDraft) {
		course.Status = models.CourseStatus(input.Status)
	}
	config.DB.Save(&course)

	if len(input.Competencies) > 0 {
		config.DB.Where("course_id = ?", course.ID).Delete(&models.CourseCompetency{})
		for i, comp := range input.Competencies {
			config.DB.Create(&models.CourseCompetency{CourseID: course.ID, Text: comp, SortOrder: i})
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kursus berhasil diperbarui"})
}

func MentorDeleteCourse(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	courseIDStr := c.Param("course_id")
	courseID, err := strconv.ParseUint(courseIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "course_id tidak valid")
		return
	}

	var mentor models.MentorProfile
	config.DB.Where("user_id = ?", userID).First(&mentor)

	var course models.Course
	if err := config.DB.Where("id = ? AND mentor_id = ?", courseID, mentor.ID).First(&course).Error; err != nil {
		utils.NotFound(c, "Kursus tidak ditemukan")
		return
	}

	// Jangan hapus kalau ada order aktif
	var activeCount int64
	config.DB.Model(&models.Order{}).
		Where("course_id = ? AND status IN ?", course.ID, []models.OrderStatus{models.OrderPending, models.OrderActive}).
		Count(&activeCount)
	if activeCount > 0 {
		utils.BadRequest(c, "Kursus tidak dapat dihapus karena masih ada order aktif")
		return
	}

	config.DB.Where("course_id = ?", course.ID).Delete(&models.CourseCompetency{})
	config.DB.Where("course_id = ?", course.ID).Delete(&models.CoursePackage{})
	config.DB.Delete(&course)

	c.JSON(http.StatusOK, gin.H{"message": "Kursus berhasil dihapus"})
}