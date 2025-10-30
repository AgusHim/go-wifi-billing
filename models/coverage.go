package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Coverage struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CodeArea    string         `gorm:"type:varchar(255)" json:"code_area"`
	Name        string         `gorm:"type:varchar(255)" json:"name"`
	Address     string         `gorm:"type:varchar(255)" json:"address"`
	Description string         `gorm:"type:text" json:"description"`
	RangeArea   int            `json:"range_area"`
	Latitude    float64        `json:"latitude"`
	Longitude   float64        `json:"longitude"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate hook to set UUID when using SQLite or when DB doesn't generate it
func (c *Coverage) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return
}
