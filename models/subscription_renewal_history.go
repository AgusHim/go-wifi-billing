package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionRenewalHistory struct {
	ID             uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	SubscriptionID uuid.UUID      `json:"subscription_id" gorm:"type:uuid;not null"`
	BillID         *uuid.UUID     `json:"bill_id" gorm:"type:uuid"`
	PaymentID      *uuid.UUID     `json:"payment_id" gorm:"type:uuid"`
	Action         string         `json:"action"`
	Status         string         `json:"status"`
	Note           string         `json:"note"`
	ExecutedAt     time.Time      `json:"executed_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Bill    *Bill    `json:"bill,omitempty" gorm:"foreignKey:BillID"`
	Payment *Payment `json:"payment,omitempty" gorm:"foreignKey:PaymentID"`
}

func (s *SubscriptionRenewalHistory) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
