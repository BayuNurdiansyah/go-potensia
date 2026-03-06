package models

import "time"

// SkillProgress menyimpan progress per skill untuk setiap anak per kursus/order.
type SkillProgress struct {
	ID        uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	OrderID   uint   `json:"order_id" gorm:"not null;index"`
	ChildID   uint   `json:"child_id" gorm:"not null;index"`
	CourseID  uint   `json:"course_id" gorm:"not null;index"`
	SkillName string `json:"skill_name" gorm:"not null"`
	Progress  int    `json:"progress" gorm:"default:0;check:progress >= 0 AND progress <= 100"`

	UpdatedAt time.Time `json:"updated_at"`
}