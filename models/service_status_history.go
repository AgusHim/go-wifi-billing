package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ServiceStatusHistory struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	ServiceAccountID  uuid.UUID      `json:"service_account_id" gorm:"type:uuid;not null"`
	PreviousStatus    string         `json:"previous_status"`
	NewStatus         string         `json:"new_status"`
	Action            string         `json:"action"`
	Source            string         `json:"source"`
	Note              string         `json:"note"`
	ProvisioningJobID *uuid.UUID     `json:"provisioning_job_id" gorm:"type:uuid"`
	BillID            *uuid.UUID     `json:"bill_id" gorm:"type:uuid"`
	PaymentID         *uuid.UUID     `json:"payment_id" gorm:"type:uuid"`
	CreatedAt         time.Time      `json:"created_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	ServiceAccount  *ServiceAccount  `json:"service_account,omitempty" gorm:"foreignKey:ServiceAccountID"`
	ProvisioningJob *ProvisioningJob `json:"provisioning_job,omitempty" gorm:"foreignKey:ProvisioningJobID"`
	Bill            *Bill            `json:"bill,omitempty" gorm:"foreignKey:BillID"`
	Payment         *Payment         `json:"payment,omitempty" gorm:"foreignKey:PaymentID"`
}

func (s *ServiceStatusHistory) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
