package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Subscription struct {
	ID                 uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	CustomerID         uuid.UUID      `json:"customer_id" gorm:"type:uuid;not null"`
	PackageID          uuid.UUID      `json:"package_id" gorm:"type:uuid;not null"`
	PeriodType         string         `json:"period_type"`
	IsIncludePPN       bool           `json:"is_include_ppn"`
	IsActiveUniqueCode bool           `json:"is_active_unique_code" gorm:"default:false"`
	PaymentType        string         `json:"payment_type"`
	DueDay             int            `json:"due_day"`
	StartDate          time.Time      `json:"start_date"`
	EndDate            time.Time      `json:"end_date"`
	AutoRenew          bool           `json:"auto_renew"`
	Status             string         `json:"status"` // active, suspended, terminated
	Description        string         `json:"description"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Relationships (optional preload)
	Customer *Customer `json:"customer" gorm:"foreignKey:CustomerID"`
	Package  *Package  `json:"package" gorm:"foreignKey:PackageID"`
}

func (s *Subscription) BeforeCreate(tx *gorm.DB) (err error) {
	s.ID = uuid.New()
	return
}
