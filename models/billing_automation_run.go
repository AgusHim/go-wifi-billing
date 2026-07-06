package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BillingAutomationRun struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	RunType         string         `json:"run_type" gorm:"type:varchar(50);index"`
	Period          string         `json:"period" gorm:"type:varchar(50);index"`
	Status          string         `json:"status" gorm:"type:varchar(50);index"`
	StartedAt       time.Time      `json:"started_at" gorm:"index"`
	FinishedAt      *time.Time     `json:"finished_at"`
	TotalCandidates int            `json:"total_candidates"`
	TotalCreated    int            `json:"total_created"`
	TotalUpdated    int            `json:"total_updated"`
	TotalSkipped    int            `json:"total_skipped"`
	TotalFailed     int            `json:"total_failed"`
	ErrorMessage    string         `json:"error_message" gorm:"type:text"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (b *BillingAutomationRun) BeforeCreate(tx *gorm.DB) (err error) {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
