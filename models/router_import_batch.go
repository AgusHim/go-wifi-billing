package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RouterImportBatch struct {
	ID                      uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	RouterID                uuid.UUID      `json:"router_id" gorm:"type:uuid;not null"`
	Mode                    string         `json:"mode"`
	Status                  string         `json:"status" gorm:"default:staged"`
	TotalNetworkPlans       int            `json:"total_network_plans"`
	NewNetworkPlans         int            `json:"new_network_plans"`
	ExistingNetworkPlans    int            `json:"existing_network_plans"`
	TotalServiceAccounts    int            `json:"total_service_accounts"`
	NewServiceAccounts      int            `json:"new_service_accounts"`
	ExistingServiceAccounts int            `json:"existing_service_accounts"`
	CreatedAt               time.Time      `json:"created_at"`
	UpdatedAt               time.Time      `json:"updated_at"`
	DeletedAt               gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Router *Router            `json:"router,omitempty" gorm:"foreignKey:RouterID"`
	Items  []RouterImportItem `json:"items,omitempty" gorm:"foreignKey:BatchID"`
}

func (r *RouterImportBatch) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
