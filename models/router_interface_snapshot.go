package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RouterInterfaceSnapshot struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	RouterID    uuid.UUID      `json:"router_id" gorm:"type:uuid;not null;index"`
	Name        string         `json:"name" gorm:"index"`
	Type        string         `json:"type"`
	Running     bool           `json:"running"`
	Disabled    bool           `json:"disabled"`
	RXBps       int64          `json:"rx_bps"`
	TXBps       int64          `json:"tx_bps"`
	RXPacket    int64          `json:"rx_packet"`
	TXPacket    int64          `json:"tx_packet"`
	CollectedAt time.Time      `json:"collected_at" gorm:"index"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Router *Router `json:"router,omitempty" gorm:"foreignKey:RouterID"`
}

func (r *RouterInterfaceSnapshot) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
