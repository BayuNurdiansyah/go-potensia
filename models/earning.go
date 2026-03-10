package models

import "time"

// MentorBankAccount menyimpan rekening bank mentor untuk pencairan dana.
// Setiap mentor hanya punya satu rekening aktif (OneToOne via MentorID uniqueIndex).
type MentorBankAccount struct {
	ID              uint          `json:"id" gorm:"primaryKey;autoIncrement"`
	MentorID        uint          `json:"mentor_id" gorm:"uniqueIndex;not null"`
	BankName        string        `json:"bank_name" gorm:"not null;type:varchar(50)"`
	AccountNumber   string        `json:"account_number" gorm:"not null;type:varchar(30)"`
	AccountHolder   string        `json:"account_holder" gorm:"not null;type:varchar(100)"`
	WithdrawalDay   WithdrawalDay `json:"withdrawal_day" gorm:"type:varchar(10);default:'Jumat'"` // hari pencairan otomatis
	IsVerified      bool          `json:"is_verified" gorm:"default:false"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MentorEarning mencatat pendapatan mentor per sesi yang telah selesai.
// Dibuat otomatis saat session di-mark completed oleh mentor.
type MentorEarning struct {
	ID            uint    `json:"id" gorm:"primaryKey;autoIncrement"`
	MentorID      uint    `json:"mentor_id" gorm:"not null;index"`
	SessionID     uint    `json:"session_id" gorm:"not null;uniqueIndex"` // 1 sesi = 1 earning
	OrderID       uint    `json:"order_id" gorm:"not null;index"`
	ChildID       uint    `json:"child_id" gorm:"not null"`
	CourseID      uint    `json:"course_id" gorm:"not null"`
	CourseName    string  `json:"course_name" gorm:"type:varchar(200)"` // snapshot
	StudentName   string  `json:"student_name" gorm:"type:varchar(100)"` // snapshot
	PackageName   string  `json:"package_name" gorm:"type:varchar(100)"` // snapshot
	SessionNumber int     `json:"session_number"`
	GrossAmount   int64   `json:"gross_amount"`  // harga per sesi (full)
	FeeRate       float64 `json:"fee_rate" gorm:"type:decimal(4,3);default:0.100"` // platform fee, default 10%
	FeeAmount     int64   `json:"fee_amount"`   // potongan platform
	NetAmount     int64   `json:"net_amount"`   // yang diterima mentor
	EarnedAt      time.Time `json:"earned_at"`  // = session.completed_at

	// Apakah sudah masuk ke withdrawal
	WithdrawalID *uint `json:"withdrawal_id" gorm:"index"`

	CreatedAt time.Time `json:"created_at"`
}

// MentorWithdrawal mencatat pencairan dana (weekly payout) ke rekening mentor.
type MentorWithdrawal struct {
	ID            uint             `json:"id" gorm:"primaryKey;autoIncrement"`
	MentorID      uint             `json:"mentor_id" gorm:"not null;index"`
	PeriodStart   time.Time        `json:"period_start"`  // Senin minggu itu
	PeriodEnd     time.Time        `json:"period_end"`    // Minggu minggu itu
	GrossAmount   int64            `json:"gross_amount"`
	FeeAmount     int64            `json:"fee_amount"`
	NetAmount     int64            `json:"net_amount"`
	BankName      string           `json:"bank_name" gorm:"type:varchar(50)"` // snapshot saat pencairan
	AccountNumber string           `json:"account_number" gorm:"type:varchar(30)"`
	AccountHolder string           `json:"account_holder" gorm:"type:varchar(100)"`
	Status        WithdrawalStatus `json:"status" gorm:"type:varchar(20);default:'pending'"`
	ProcessedAt   *time.Time       `json:"processed_at"`
	Notes         string           `json:"notes" gorm:"type:text"` // catatan admin

	// Earnings yang masuk ke withdrawal ini (di-update via WithdrawalID di MentorEarning)
	Earnings []MentorEarning `json:"earnings,omitempty" gorm:"foreignKey:WithdrawalID"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}