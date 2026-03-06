package models

import "time"

// Invoice adalah tagihan pembayaran yang dibuat untuk setiap order.
type Invoice struct {
	ID          uint          `json:"id" gorm:"primaryKey;autoIncrement"`
	OrderID     uint          `json:"order_id" gorm:"not null;index"`
	Order       Order         `json:"order,omitempty" gorm:"foreignKey:OrderID"`
	ParentID    uint          `json:"parent_id" gorm:"not null;index"`
	Amount      int64         `json:"amount"`
	Description string        `json:"description"`
	Period      string        `json:"period"` // contoh: "Januari 2025"
	Status      PaymentStatus `json:"status" gorm:"type:varchar(20);default:'unpaid'"`
	DueDate     time.Time     `json:"due_date"`
	PaidAt      *time.Time    `json:"paid_at"`
	Method      PaymentMethod `json:"method" gorm:"type:varchar(30)"`
	ProofURL    *string       `json:"proof_url"` // URL bukti transfer (opsional)

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}