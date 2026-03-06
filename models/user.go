package models

import "time"

// User adalah akun login untuk semua role (mentor / parent / admin).
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"not null"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Phone      string    `json:"phone" gorm:"type:varchar(20)"`
	Password  string    `json:"-" gorm:"not null"`
	Role       Role      `json:"role" gorm:"type:varchar(20);not null"`
	AvatarURL  *string   `json:"avatar_url"`
	IsVerified bool      `json:"is_verified" gorm:"default:false"`
	IsActive   bool      `json:"is_active" gorm:"default:true"`

	// OTP untuk verifikasi email saat register
	OTP           string `json:"-" gorm:"type:varchar(6)"`
	OTPExpired    int64  `json:"-"`
	OTPAttempts   int    `json:"-" gorm:"default:0"`
	LastOTPSentAt int64  `json:"-" gorm:"default:0"`

	// Token reset password
	ResetToken        string `json:"-" gorm:"type:varchar(64);index"`
	ResetTokenExpired int64  `json:"-"`
	ResetTokenUsed    bool   `json:"-" gorm:"default:false"`

	// Preferensi notifikasi
	NotifReminder bool `json:"notif_reminder" gorm:"default:true"`
	NotifPromo    bool `json:"notif_promo" gorm:"default:true"`
	NotifSchedule bool `json:"notif_schedule" gorm:"default:true"`
	NotifInfo     bool `json:"notif_info" gorm:"default:true"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}