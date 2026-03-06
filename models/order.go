package models

import "time"

// Order adalah transaksi pembelian paket kursus oleh parent untuk anaknya.
type Order struct {
	ID       uint  `json:"id" gorm:"primaryKey;autoIncrement"`
	ParentID uint  `json:"parent_id" gorm:"not null;index"`
	Parent   User  `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	ChildID  uint  `json:"child_id" gorm:"not null;index"`
	Child    Child `json:"child,omitempty" gorm:"foreignKey:ChildID"`
	CourseID uint  `json:"course_id" gorm:"not null;index"`
	Course   Course `json:"course,omitempty" gorm:"foreignKey:CourseID"`
	PackageID uint         `json:"package_id" gorm:"not null"`
	Package   CoursePackage `json:"package,omitempty" gorm:"foreignKey:PackageID"`
	MentorID  uint         `json:"mentor_id" gorm:"not null;index"`

	TotalSessions      int         `json:"total_sessions"`
	CompletedSessions  int         `json:"completed_sessions" gorm:"default:0"`
	RemainingSessions  int         `json:"remaining_sessions"`
	DurationPerSession int         `json:"duration_per_session"` // menit
	TotalPrice         int64       `json:"total_price"`
	Status             OrderStatus `json:"status" gorm:"type:varchar(20);default:'pending'"`

	// Preferensi jadwal dari parent saat booking
	PreferredDays string `json:"preferred_days" gorm:"type:varchar(50)"` // "1,3,5" = Senin,Rabu,Jumat
	PreferredTime string `json:"preferred_time" gorm:"type:varchar(10)"` // "09:00"

	MeetLink string `json:"meet_link" gorm:"type:varchar(300)"`
	Notes    string `json:"notes" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Session adalah satu pertemuan belajar antara mentor dan siswa.
type Session struct {
	ID            uint          `json:"id" gorm:"primaryKey;autoIncrement"`
	OrderID       uint          `json:"order_id" gorm:"not null;index"`
	Order         Order         `json:"order,omitempty" gorm:"foreignKey:OrderID"`
	MentorID      uint          `json:"mentor_id" gorm:"not null;index"`
	ChildID       uint          `json:"child_id" gorm:"not null;index"`
	ScheduledAt   time.Time     `json:"scheduled_at"`
	Duration      int           `json:"duration"` // menit
	Status        SessionStatus `json:"status" gorm:"type:varchar(20);default:'upcoming'"`
	MeetLink      string        `json:"meet_link" gorm:"type:varchar(300)"`
	SessionNumber int           `json:"session_number"`

	// Diisi mentor setelah sesi selesai
	Topic  string `json:"topic" gorm:"type:varchar(200)"`
	Notes  string `json:"notes" gorm:"type:text"`
	Stars  int    `json:"stars" gorm:"default:0"` // bintang dari mentor untuk siswa (1–5)

	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}