package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Payment struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	BillID      uuid.UUID  `gorm:"type:uuid" json:"bill_id"`
	RefID       string     `json:"ref_id"`
	PaymentDate time.Time  `json:"payment_date"`
	DueDate     time.Time  `json:"due_date"`
	ExpiredDate time.Time  `json:"expired_date"`
	Method      string     `json:"method"` // bank_transfer, ewallet, cash
	PaymentUrl  *string    `json:"payment_url"`
	Amount      int        `json:"amount"`
	Status      string     `json:"status"` // pending, confirmed, failed
	AdminID     *uuid.UUID `gorm:"type:uuid" json:"admin_id"`
	// Snapshot fields: di-copy dari bill saat payment dibuat.
	// Immutable — tetap utuh meski bill/customer/package terhapus.
	CustomerName    string     `json:"customer_name"`
	CustomerPhone   string     `json:"customer_phone"`
	CustomerEmail   string     `json:"customer_email"`
	CustomerAddress string     `json:"customer_address"`
	BillPublicID    string     `json:"bill_public_id"`
	BillAmount      int        `json:"bill_amount"`
	BillPPN         int        `json:"bill_ppn"`
	BillUniqueCode  int        `json:"bill_unique_code"`
	BillDate        *time.Time `json:"bill_date"`
	BillDueDate     *time.Time `json:"bill_due_date"`
	PackageName     string     `json:"package_name"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	Bill  Bill `json:"bill" gorm:"foreignKey:BillID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" `
	Admin User `json:"admin" gorm:"foreignKey:AdminID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" `
}
