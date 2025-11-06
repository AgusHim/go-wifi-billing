package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Odp struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	OdcID         uuid.UUID  `json:"odc_id" gorm:"type:uuid;not null"`
	OdcPortNumber int        `json:"odc_port_number"`
	CoverageID    uuid.UUID  `json:"coverage_id" gorm:"type:uuid;not null"`
	Code          string     `json:"code"`
	FoTubeColor   string     `json:"fo_tube_color"`
	PoleNumber    string     `json:"pole_number"`
	CountPort     int        `json:"count_port"`
	Description   string     `json:"description"`
	ImageURL      string     `json:"image_url"`
	Latitude      float64    `json:"latitude"`
	Longitude     float64    `json:"longitude"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at" gorm:"index"`

	Odc      *Odc      `json:"odc" gorm:"foreignKey:OdcID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Coverage *Coverage `json:"coverage" gorm:"foreignKey:CoverageID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (o *Odp) BeforeCreate(tx *gorm.DB) (err error) {
	o.ID = uuid.New()
	return
}
