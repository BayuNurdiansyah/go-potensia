package models

import "time"

// Review adalah ulasan yang diberikan parent terhadap mentor/kursus setelah sesi selesai.
type Review struct {
	ID        uint  `json:"id" gorm:"primaryKey;autoIncrement"`
	OrderID   uint  `json:"order_id" gorm:"not null;index"`
	CourseID  uint  `json:"course_id" gorm:"not null;index"`
	MentorID  uint  `json:"mentor_id" gorm:"not null;index"`
	ParentID  uint  `json:"parent_id" gorm:"not null;index"`
	ChildID   uint  `json:"child_id" gorm:"not null"`
	PackageID uint  `json:"package_id"`
	Rating    int   `json:"rating" gorm:"not null;check:rating >= 1 AND rating <= 5"`
	Comment   string `json:"comment" gorm:"type:text"`

	// Snapshot nama saat review dibuat (agar tidak berubah jika user ubah nama)
	ReviewerName string `json:"reviewer_name"`
	CourseName   string `json:"course_name"`
	PackageName  string `json:"package_name"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}