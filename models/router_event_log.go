package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RouterEventLog struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	RouterID    uuid.UUID      `json:"router_id" gorm:"type:uuid;not null;index"`
	Topic       string         `json:"topic" gorm:"index"`
	Message     string         `json:"message"`
	Severity    string         `json:"severity" gorm:"index"`
	RemoteTime  string         `json:"remote_time"`
	CollectedAt time.Time      `json:"collected_at" gorm:"index"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Router *Router `json:"router,omitempty" gorm:"foreignKey:RouterID"`
}

func (r *RouterEventLog) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
