package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RouterSnapshot struct {
	ID              uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	RouterID        uuid.UUID      `json:"router_id" gorm:"type:uuid;not null;index"`
	Identity        string         `json:"identity"`
	RouterOSVersion string         `json:"routeros_version"`
	BoardName       string         `json:"board_name"`
	Architecture    string         `json:"architecture"`
	CPULoad         int            `json:"cpu_load"`
	FreeMemory      int64          `json:"free_memory"`
	TotalMemory     int64          `json:"total_memory"`
	FreeHDDSpace    int64          `json:"free_hdd_space"`
	TotalHDDSpace   int64          `json:"total_hdd_space"`
	Uptime          string         `json:"uptime"`
	ActivePPPoE     int            `json:"active_pppoe"`
	ActiveHotspot   int            `json:"active_hotspot"`
	InterfaceCount  int            `json:"interface_count"`
	CollectedAt     time.Time      `json:"collected_at" gorm:"index"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Router *Router `json:"router,omitempty" gorm:"foreignKey:RouterID"`
}

func (r *RouterSnapshot) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
