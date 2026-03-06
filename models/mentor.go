package models

import "time"

// MentorProfile menyimpan data lengkap mentor (relasi 1:1 dengan User).
type MentorProfile struct {
	ID          uint    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      uint    `json:"user_id" gorm:"uniqueIndex;not null"`
	User        User    `json:"user" gorm:"foreignKey:UserID"`
	Expertise   string  `json:"expertise" gorm:"type:varchar(200)"`
	Bio         string  `json:"bio" gorm:"type:text"`
	Rating      float64 `json:"rating" gorm:"default:0;type:decimal(3,2)"`
	TotalReview int     `json:"total_review" gorm:"default:0"`

	// Lokasi
	Province string `json:"province" gorm:"type:varchar(100)"`
	Regency  string `json:"regency" gorm:"type:varchar(100)"`
	District string `json:"district" gorm:"type:varchar(100)"`
	Address  string `json:"address" gorm:"type:text"`

	// Rekening bank untuk pencairan
	BankName        string `json:"bank_name" gorm:"type:varchar(50)"`
	BankAccount     string `json:"bank_account" gorm:"type:varchar(30)"`
	BankAccountName string `json:"bank_account_name" gorm:"type:varchar(100)"`

	// Statistik (di-cache agar tidak query ulang tiap request)
	TotalStudents int   `json:"total_students" gorm:"default:0"`
	TotalSessions int   `json:"total_sessions" gorm:"default:0"`
	TotalEarnings int64 `json:"total_earnings" gorm:"default:0"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MentorCertificate menyimpan sertifikat yang dimiliki mentor.
type MentorCertificate struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	MentorID  uint      `json:"mentor_id" gorm:"not null;index"`
	Title     string    `json:"title" gorm:"not null"`
	Issuer    string    `json:"issuer"`
	Year      string    `json:"year" gorm:"type:varchar(4)"`
	CreatedAt time.Time `json:"created_at"`
}

// MentorAchievement menyimpan prestasi mentor.
type MentorAchievement struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	MentorID  uint      `json:"mentor_id" gorm:"not null;index"`
	Text      string    `json:"text" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}

// MentorGallery menyimpan foto galeri mentor/kursus.
type MentorGallery struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	MentorID  uint      `json:"mentor_id" gorm:"not null;index"`
	ImageURL  string    `json:"image_url" gorm:"not null"`
	SortOrder int       `json:"sort_order" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
}

// MentorEducation menyimpan riwayat pendidikan mentor.
type MentorEducation struct {
	ID          uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	MentorID    uint      `json:"mentor_id" gorm:"not null;index"`
	Degree      string    `json:"degree"`
	Institution string    `json:"institution"`
	Year        string    `json:"year" gorm:"type:varchar(4)"`
	CreatedAt   time.Time `json:"created_at"`
}