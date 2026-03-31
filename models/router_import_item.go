package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RouterImportItem struct {
	ID                       uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	BatchID                  uuid.UUID      `json:"batch_id" gorm:"type:uuid;not null"`
	ItemType                 string         `json:"item_type"`
	ServiceType              string         `json:"service_type"`
	Username                 string         `json:"username"`
	RemoteID                 string         `json:"remote_id"`
	ProfileName              string         `json:"profile_name"`
	AddressPool              string         `json:"address_pool"`
	RemoteStatus             string         `json:"remote_status"`
	SuggestedName            string         `json:"suggested_name"`
	ExistingNetworkPlanID    *uuid.UUID     `json:"existing_network_plan_id" gorm:"type:uuid"`
	ExistingServiceAccountID *uuid.UUID     `json:"existing_service_account_id" gorm:"type:uuid"`
	MatchedNetworkPlanID     *uuid.UUID     `json:"matched_network_plan_id" gorm:"type:uuid"`
	Conflict                 bool           `json:"conflict"`
	RecommendedAction        string         `json:"recommended_action"`
	StageStatus              string         `json:"stage_status" gorm:"default:staged"`
	Note                     string         `json:"note"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
	DeletedAt                gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Batch                  *RouterImportBatch `json:"batch,omitempty" gorm:"foreignKey:BatchID"`
	ExistingNetworkPlan    *NetworkPlan       `json:"existing_network_plan,omitempty" gorm:"foreignKey:ExistingNetworkPlanID"`
	ExistingServiceAccount *ServiceAccount    `json:"existing_service_account,omitempty" gorm:"foreignKey:ExistingServiceAccountID"`
	MatchedNetworkPlan     *NetworkPlan       `json:"matched_network_plan,omitempty" gorm:"foreignKey:MatchedNetworkPlanID"`
}

func (r *RouterImportItem) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
