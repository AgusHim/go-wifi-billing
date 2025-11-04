package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Customer struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	UserID        uuid.UUID      `json:"user_id" gorm:"type:uuid"`
	CoverageID    uuid.UUID      `json:"coverage_id" gorm:"type:uuid"`
	OdcID         uuid.UUID      `json:"odc_id" gorm:"type:uuid"`
	OdpID         uuid.UUID      `json:"odp_id" gorm:"type:uuid"`
	PortOdp       string         `json:"port_odp"`
	ServiceNumber string         `json:"service_number"`
	Card          string         `json:"card"`
	IDCard        string         `json:"id_card"`
	IsSendWa      bool           `json:"is_send_wa"`
	Status        string         `json:"status" gorm:"default:active"`
	Address       string         `json:"address"`
	Latitude      float64        `json:"latitude"`
	Longitude     float64        `json:"longitude"`
	Description   string         `json:"description"`
	Mode          string         `json:"mode"`
	IDPPOE        string         `json:"id_ppoe"`
	ProfilePPOE   string         `json:"profile_ppoe"`
	AdminID       *uuid.UUID     `json:"admin_id" gorm:"type:uuid"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Relationships (optional preload)
	User     *User     `json:"user" gorm:"foreignKey:UserID"`
	Coverage *Coverage `json:"coverage" gorm:"foreignKey:CoverageID"`
	Odc      *Odc      `json:"odc" gorm:"foreignKey:OdcID"`
	Odp      *Odp      `json:"odp" gorm:"foreignKey:OdpID"`
	Admin    *User     `json:"admin" gorm:"foreignKey:AdminID"`
}

func (c *Customer) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID = uuid.New()
	return
}
