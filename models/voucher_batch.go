package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VoucherBatch struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Name          string         `json:"name"`
	Prefix        string         `json:"prefix"`
	Quantity      int            `json:"quantity"`
	ServiceType   string         `json:"service_type"`
	RouterID      *uuid.UUID     `json:"router_id" gorm:"type:uuid"`
	NetworkPlanID *uuid.UUID     `json:"network_plan_id" gorm:"type:uuid"`
	ExpiresAt     *time.Time     `json:"expires_at"`
	Status        string         `json:"status" gorm:"default:active"`
	Notes         string         `json:"notes"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Router      *Router      `json:"router,omitempty" gorm:"foreignKey:RouterID"`
	NetworkPlan *NetworkPlan `json:"network_plan,omitempty" gorm:"foreignKey:NetworkPlanID"`
	Vouchers    []Voucher    `json:"vouchers,omitempty" gorm:"foreignKey:BatchID"`
}

func (v *VoucherBatch) BeforeCreate(tx *gorm.DB) (err error) {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}
