package models

import "time"

type Role string

const (
	RoleStudent Role = "student"
	RoleMentor  Role = "mentor"
)

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"not null"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"not null"` // never expose in JSON
	Role      Role      `json:"role" gorm:"type:varchar(20);default:'student'"`
	IsVerified bool     `json:"is_verified" gorm:"default:false"`

	// OTP for email verification
	OTP           string `json:"-" gorm:"type:varchar(6)"`
	OTPExpired    int64  `json:"-"`
	OTPAttempts   int    `json:"-" gorm:"default:0"`
	LastOTPSentAt int64  `json:"-" gorm:"default:0"`

	// Forgot password token
	ResetToken          string `json:"-" gorm:"type:varchar(64);index"`
	ResetTokenExpired   int64  `json:"-"`
	ResetTokenUsed      bool   `json:"-" gorm:"default:false"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}