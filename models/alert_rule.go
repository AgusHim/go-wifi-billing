package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AlertRule struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	RuleKey       string         `json:"rule_key" gorm:"uniqueIndex;not null"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Severity      string         `json:"severity" gorm:"index"`
	Enabled       bool           `json:"enabled" gorm:"default:true;index"`
	Threshold     int            `json:"threshold"`
	WindowMinutes int            `json:"window_minutes"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (a *AlertRule) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
