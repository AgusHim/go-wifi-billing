package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProvisioningLog struct {
	ID                uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	RouterID          *uuid.UUID `json:"router_id" gorm:"type:uuid"`
	ProvisioningJobID *uuid.UUID `json:"provisioning_job_id" gorm:"type:uuid"`
	Level             string     `json:"level"`
	Action            string     `json:"action"`
	RequestPayload    string     `json:"request_payload"`
	ResponsePayload   string     `json:"response_payload"`
	Message           string     `json:"message"`
	CreatedAt         time.Time  `json:"created_at"`

	Router          *Router          `json:"router,omitempty" gorm:"foreignKey:RouterID"`
	ProvisioningJob *ProvisioningJob `json:"provisioning_job,omitempty" gorm:"foreignKey:ProvisioningJobID"`
}

func (p *ProvisioningLog) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
