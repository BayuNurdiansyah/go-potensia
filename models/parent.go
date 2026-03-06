package models

import "time"

// ParentProfile menyimpan data tambahan orang tua (relasi 1:1 dengan User).
type ParentProfile struct {
	ID      uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID  uint   `json:"user_id" gorm:"uniqueIndex;not null"`
	User    User   `json:"user" gorm:"foreignKey:UserID"`
	Address string `json:"address" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Child menyimpan data anak yang didaftarkan oleh parent.
type Child struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	ParentID  uint      `json:"parent_id" gorm:"not null;index"`
	Parent    User      `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Name      string    `json:"name" gorm:"not null"`
	BirthDate time.Time `json:"birth_date"`
	Gender    Gender    `json:"gender" gorm:"type:varchar(20)"`
	Education string    `json:"education" gorm:"type:varchar(50)"`
	AvatarURL *string   `json:"avatar_url"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}