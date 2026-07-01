package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReconciliationFinding struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	RouterID          uuid.UUID      `json:"router_id" gorm:"type:uuid;not null;index"`
	ServiceAccountID  *uuid.UUID     `json:"service_account_id" gorm:"type:uuid;index"`
	FindingType       string         `json:"finding_type" gorm:"index"`
	Severity          string         `json:"severity" gorm:"index"`
	Description       string         `json:"description"`
	RecommendedAction string         `json:"recommended_action"`
	Status            string         `json:"status" gorm:"default:open;index"`
	RemoteServiceType string         `json:"remote_service_type"`
	RemoteUsername    string         `json:"remote_username"`
	RemoteID          string         `json:"remote_id"`
	RemoteProfileName string         `json:"remote_profile_name"`
	RemoteStatus      string         `json:"remote_status"`
	DetectedAt        time.Time      `json:"detected_at" gorm:"index"`
	ResolvedAt        *time.Time     `json:"resolved_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Router         *Router         `json:"router,omitempty" gorm:"foreignKey:RouterID"`
	ServiceAccount *ServiceAccount `json:"service_account,omitempty" gorm:"foreignKey:ServiceAccountID"`
}

func (r *ReconciliationFinding) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
