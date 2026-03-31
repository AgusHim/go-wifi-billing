package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NetworkPlan struct {
	ID                  uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Name                string         `json:"name"`
	ServiceType         string         `json:"service_type"`
	RouterID            *uuid.UUID     `json:"router_id" gorm:"type:uuid"`
	MikrotikProfileName string         `json:"mikrotik_profile_name"`
	AddressPool         string         `json:"address_pool"`
	DownloadKbps        int            `json:"download_kbps"`
	UploadKbps          int            `json:"upload_kbps"`
	Description         string         `json:"description"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Router *Router `json:"router,omitempty" gorm:"foreignKey:RouterID"`
}

func (n *NetworkPlan) BeforeCreate(tx *gorm.DB) (err error) {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}
