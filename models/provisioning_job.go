package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProvisioningJob struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	EntityType   string         `json:"entity_type"`
	EntityID     *uuid.UUID     `json:"entity_id" gorm:"type:uuid"`
	Action       string         `json:"action"`
	Payload      string         `json:"payload"`
	Status       string         `json:"status" gorm:"default:pending"`
	AttemptCount int            `json:"attempt_count" gorm:"default:0"`
	ScheduledAt  *time.Time     `json:"scheduled_at"`
	ExecutedAt   *time.Time     `json:"executed_at"`
	ErrorMessage string         `json:"error_message"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (p *ProvisioningJob) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
