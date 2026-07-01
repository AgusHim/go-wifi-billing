package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AlertNotification struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	AlertEventID uuid.UUID      `json:"alert_event_id" gorm:"type:uuid;not null;index"`
	Channel      string         `json:"channel" gorm:"index"`
	Recipient    string         `json:"recipient"`
	Status       string         `json:"status" gorm:"default:pending;index"`
	ErrorMessage string         `json:"error_message"`
	SentAt       *time.Time     `json:"sent_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	AlertEvent *AlertEvent `json:"alert_event,omitempty" gorm:"foreignKey:AlertEventID"`
}

func (a *AlertNotification) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
