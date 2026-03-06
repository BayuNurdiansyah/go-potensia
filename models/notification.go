package models

import "time"

// Notification adalah notifikasi in-app untuk user.
type Notification struct {
	ID      uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID  uint   `json:"user_id" gorm:"not null;index"`
	Type    string `json:"type" gorm:"type:varchar(30)"` // reminder | promo | schedule | info
	Title   string `json:"title" gorm:"not null"`
	Body    string `json:"body" gorm:"type:text"`
	IsRead  bool   `json:"is_read" gorm:"default:false"`
	RefID   *uint  `json:"ref_id"`   // ID sesi/order terkait (opsional)
	RefType string `json:"ref_type"` // "session" | "order" | "invoice"

	CreatedAt time.Time `json:"created_at"`
}