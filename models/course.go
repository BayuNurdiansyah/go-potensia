package models

import "time"

// Course adalah kursus yang dibuat oleh mentor.
type Course struct {
	ID             uint         `json:"id" gorm:"primaryKey;autoIncrement"`
	MentorID       uint         `json:"mentor_id" gorm:"not null;index"`
	Mentor         MentorProfile `json:"mentor,omitempty" gorm:"foreignKey:MentorID"`
	Title          string       `json:"title" gorm:"not null"`
	Category       string       `json:"category" gorm:"type:varchar(100);index"`
	Description    string       `json:"description" gorm:"type:text"`
	Status         CourseStatus `json:"status" gorm:"type:varchar(20);default:'draft'"`
	Rating         float64      `json:"rating" gorm:"default:0;type:decimal(3,2)"`
	TotalReview    int          `json:"total_review" gorm:"default:0"`
	TotalStudents  int          `json:"total_students" gorm:"default:0"`
	ActiveStudents int          `json:"active_students" gorm:"default:0"`

	Competencies []CourseCompetency `json:"competencies,omitempty" gorm:"foreignKey:CourseID"`
	Packages     []CoursePackage    `json:"packages,omitempty" gorm:"foreignKey:CourseID"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CourseCompetency adalah satu poin kompetensi/materi dalam sebuah kursus.
type CourseCompetency struct {
	ID        uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	CourseID  uint   `json:"course_id" gorm:"not null;index"`
	Text      string `json:"text" gorm:"not null"`
	SortOrder int    `json:"sort_order" gorm:"default:0"`
}

// CoursePackage adalah paket harga dalam sebuah kursus (Starter / Reguler / Intensif).
type CoursePackage struct {
	ID                 uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	CourseID           uint   `json:"course_id" gorm:"not null;index"`
	Name               string `json:"name" gorm:"not null"`
	DurationPerSession int    `json:"duration_per_session"` // menit
	TotalSessions      int    `json:"total_sessions"`
	Price              int64  `json:"price"`
	OriginalPrice      *int64 `json:"original_price"` // nullable, harga coret
	IsHighlight        bool   `json:"is_highlight" gorm:"default:false"`
	IsActive           bool   `json:"is_active" gorm:"default:true"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}