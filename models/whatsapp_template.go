package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WhatsAppTemplate struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Name      string         `json:"name" gorm:"type:varchar(100);not null"`
	Key       string         `json:"key" gorm:"type:varchar(100);not null;uniqueIndex"`
	Content   string         `json:"content" gorm:"type:text;not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (t *WhatsAppTemplate) BeforeCreate(tx *gorm.DB) (err error) {
	t.ID = uuid.New()
	return
}
