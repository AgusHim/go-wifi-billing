package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Bill struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	PublicID       string     `gorm:"type:varchar(255);unique" json:"public_id"`
	SubscriptionID uuid.UUID  `gorm:"type:uuid" json:"subscription_id"`
	CustomerID     uuid.UUID  `gorm:"type:uuid" json:"customer_id"`
	BillDate       time.Time  `json:"bill_date"`
	DueDate        time.Time  `json:"due_date"`
	TerminatedDate *time.Time `json:"terminated_date"`
	Amount         int        `json:"amount"`
	PPN            int        `json:"ppn"`
	UniqueCode     int        `json:"unique_code"`
	Status         string     `json:"status"` // unpaid, paid, overdue
	AdminID        *uuid.UUID `gorm:"type:uuid" json:"admin_id"`
	// Snapshot fields: dipreserve walaupun customer/user/package terhapus.
	// Diisi saat bill dibuat; jangan diubah saat update kecuali memang ada perbaikan data.
	CustomerName          string         `json:"customer_name"`
	CustomerPhone         string         `json:"customer_phone"`
	CustomerEmail         string         `json:"customer_email"`
	CustomerServiceNumber string         `json:"customer_service_number"`
	CustomerAddress       string         `json:"customer_address"`
	PackageName           string         `json:"package_name"`
	PackagePrice          int            `json:"package_price"`
	CoverageName          string         `json:"coverage_name"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	Subscription Subscription `json:"subscription" gorm:"foreignKey:SubscriptionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Customer     Customer     `json:"customer" gorm:"foreignKey:CustomerID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
