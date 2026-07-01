package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ServiceSessionSnapshot struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	RouterID         uuid.UUID      `json:"router_id" gorm:"type:uuid;not null;index"`
	ServiceAccountID *uuid.UUID     `json:"service_account_id" gorm:"type:uuid;index"`
	ServiceType      string         `json:"service_type" gorm:"index"`
	Username         string         `json:"username" gorm:"index"`
	RemoteID         string         `json:"remote_id"`
	Address          string         `json:"address"`
	Uptime           string         `json:"uptime"`
	CallerID         string         `json:"caller_id"`
	Online           bool           `json:"online" gorm:"index"`
	RXBytes          int64          `json:"rx_bytes"`
	TXBytes          int64          `json:"tx_bytes"`
	CollectedAt      time.Time      `json:"collected_at" gorm:"index"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Router         *Router         `json:"router,omitempty" gorm:"foreignKey:RouterID"`
	ServiceAccount *ServiceAccount `json:"service_account,omitempty" gorm:"foreignKey:ServiceAccountID"`
}

func (s *ServiceSessionSnapshot) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
