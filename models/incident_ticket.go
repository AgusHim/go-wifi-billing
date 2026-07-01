package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IncidentTicket struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	AlertEventID *uuid.UUID     `json:"alert_event_id" gorm:"type:uuid;index"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Severity     string         `json:"severity" gorm:"index"`
	Status       string         `json:"status" gorm:"default:open;index"`
	OpenedAt     time.Time      `json:"opened_at" gorm:"index"`
	ClosedAt     *time.Time     `json:"closed_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	AlertEvent *AlertEvent `json:"alert_event,omitempty" gorm:"foreignKey:AlertEventID"`
}

func (i *IncidentTicket) BeforeCreate(tx *gorm.DB) (err error) {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	if i.OpenedAt.IsZero() {
		i.OpenedAt = time.Now()
	}
	return nil
}
