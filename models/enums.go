package models

type Role string
type CourseStatus string
type SessionStatus string
type OrderStatus string
type PaymentStatus string
type PaymentMethod string
type Gender string
type WithdrawalStatus string
type WithdrawalDay string

const (
	RoleParent Role = "parent"
	RoleMentor Role = "mentor"
	RoleAdmin  Role = "admin"

	CourseStatusActive CourseStatus = "active"
	CourseStatusDraft  CourseStatus = "draft"

	SessionUpcoming  SessionStatus = "upcoming"
	SessionOngoing   SessionStatus = "ongoing"
	SessionCompleted SessionStatus = "completed"
	SessionCancelled SessionStatus = "cancelled"

	OrderPending   OrderStatus = "pending"
	OrderActive    OrderStatus = "active"
	OrderCompleted OrderStatus = "completed"
	OrderCancelled OrderStatus = "cancelled"

	PaymentUnpaid  PaymentStatus = "unpaid"
	PaymentPaid    PaymentStatus = "paid"
	PaymentExpired PaymentStatus = "expired"

	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
	PaymentMethodEWallet      PaymentMethod = "e_wallet"
	PaymentMethodVA           PaymentMethod = "virtual_account"

	GenderMale   Gender = "Laki-laki"
	GenderFemale Gender = "Perempuan"

	// Withdrawal / pencairan
	WithdrawalPending   WithdrawalStatus = "pending"
	WithdrawalProcessed WithdrawalStatus = "processed"
	WithdrawalFailed    WithdrawalStatus = "failed"

	WithdrawalDaySenin  WithdrawalDay = "Senin"
	WithdrawalDaySelasa WithdrawalDay = "Selasa"
	WithdrawalDayRabu   WithdrawalDay = "Rabu"
	WithdrawalDayKamis  WithdrawalDay = "Kamis"
	WithdrawalDayJumat  WithdrawalDay = "Jumat"
	WithdrawalDaySabtu  WithdrawalDay = "Sabtu"
	WithdrawalDayMinggu WithdrawalDay = "Minggu"
)