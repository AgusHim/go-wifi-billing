package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Bill struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	SubscriptionID uuid.UUID      `gorm:"type:uuid" json:"subscription_id"`
	UserID         uuid.UUID      `gorm:"type:uuid" json:"user_id"`
	BillDate       time.Time      `json:"bill_date"`
	DueDate        time.Time      `json:"due_date"`
	TerminatedDate *time.Time     `json:"terminated_date"`
	Amount         int            `json:"amount"`
	Status         string         `json:"status"` // unpaid, paid, overdue
	AdminID        *uuid.UUID     `gorm:"type:uuid" json:"admin_id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	Subscription Subscription `gorm:"foreignKey:SubscriptionID" json:"subscription"`
	User         User         `gorm:"foreignKey:UserID" json:"user"`
}
