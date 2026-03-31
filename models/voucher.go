package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Voucher struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	BatchID           uuid.UUID      `json:"batch_id" gorm:"type:uuid;not null"`
	Code              string         `json:"code" gorm:"uniqueIndex"`
	Username          string         `json:"username"`
	PasswordEncrypted string         `json:"-" gorm:"column:password_encrypted"`
	Password          string         `json:"password,omitempty" gorm:"-"`
	ServiceType       string         `json:"service_type"`
	RouterID          *uuid.UUID     `json:"router_id" gorm:"type:uuid"`
	NetworkPlanID     *uuid.UUID     `json:"network_plan_id" gorm:"type:uuid"`
	Status            string         `json:"status" gorm:"default:generated"`
	RedeemedAt        *time.Time     `json:"redeemed_at"`
	RedeemerName      string         `json:"redeemer_name"`
	RedeemerPhone     string         `json:"redeemer_phone"`
	LastProvisionedAt *time.Time     `json:"last_provisioned_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Batch       *VoucherBatch `json:"batch,omitempty" gorm:"foreignKey:BatchID"`
	Router      *Router       `json:"router,omitempty" gorm:"foreignKey:RouterID"`
	NetworkPlan *NetworkPlan  `json:"network_plan,omitempty" gorm:"foreignKey:NetworkPlanID"`
}

func (v *Voucher) BeforeCreate(tx *gorm.DB) (err error) {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}
