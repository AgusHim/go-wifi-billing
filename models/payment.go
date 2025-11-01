package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Payment struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	BillID      uuid.UUID      `gorm:"type:uuid" json:"bill_id"`
	RefID       string         `json:"ref_id"`
	PaymentDate time.Time      `json:"payment_date"`
	DueDate     time.Time      `json:"due_date"`
	Method      string         `json:"method"` // bank_transfer, ewallet, cash
	Amount      int            `json:"amount"`
	Status      string         `json:"status"` // pending, confirmed, failed
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	Bill Bill `gorm:"foreignKey:BillID" json:"bill"`
}
