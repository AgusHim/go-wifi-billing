package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Odc struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	CoverageID  uuid.UUID      `json:"coverage_id" gorm:"type:uuid;not null"`
	OdcKey      string         `json:"odc_key"`
	Code        string         `json:"code"`
	PortOlt     int            `json:"port_olt"`
	FoColor     string         `json:"fo_color"`
	PoleNumber  string         `json:"pole_number"`
	CountPort   int            `json:"count_port"`
	Description string         `json:"description"`
	ImageURL    string         `json:"image_url"`
	Latitude    float64        `json:"latitude"`
	Longitude   float64        `json:"longitude"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (o *Odc) BeforeCreate(tx *gorm.DB) (err error) {
	o.ID = uuid.New()
	return
}
