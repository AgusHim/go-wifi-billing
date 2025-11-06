package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Package struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	Category    string     `json:"category"`
	Name        string     `json:"name"`
	SpeedMbps   int        `json:"speed_mbps"`
	QuotaGB     int        `json:"quota_gb"`
	Price       int        `json:"price"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at" gorm:"index"`
}

func (p *Package) BeforeCreate(tx *gorm.DB) (err error) {
	p.ID = uuid.New()
	return
}
