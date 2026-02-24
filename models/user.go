package models

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string `json:"name"`
	Email    string `json:"email" gorm:"unique"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Phone  string `json:"phone"`

	OTP        string
	OTPExpired int64
	IsVerified bool
	LastOTPSentAt  int64
	OTPAttempts    int
}