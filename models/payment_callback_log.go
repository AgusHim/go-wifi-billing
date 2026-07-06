package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PaymentCallbackLog struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	PaymentID         *uuid.UUID     `json:"payment_id" gorm:"type:uuid;index"`
	Provider          string         `json:"provider" gorm:"type:varchar(50);index"`
	OrderID           string         `json:"order_id" gorm:"type:varchar(255);index"`
	TransactionStatus string         `json:"transaction_status" gorm:"type:varchar(50);index"`
	FraudStatus       string         `json:"fraud_status" gorm:"type:varchar(50)"`
	GrossAmount       string         `json:"gross_amount" gorm:"type:varchar(50)"`
	SignatureValid    bool           `json:"signature_valid"`
	RawPayload        string         `json:"raw_payload" gorm:"type:text"`
	ReceivedAt        time.Time      `json:"received_at" gorm:"index"`
	ProcessedAt       *time.Time     `json:"processed_at"`
	ErrorMessage      string         `json:"error_message" gorm:"type:text"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (p *PaymentCallbackLog) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
