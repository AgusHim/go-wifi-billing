package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ServiceAccount struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	SubscriptionID    uuid.UUID      `json:"subscription_id" gorm:"type:uuid;not null"`
	RouterID          *uuid.UUID     `json:"router_id" gorm:"type:uuid"`
	NetworkPlanID     *uuid.UUID     `json:"network_plan_id" gorm:"type:uuid"`
	ServiceType       string         `json:"service_type"`
	Username          string         `json:"username"`
	PasswordEncrypted string         `json:"-" gorm:"column:password_encrypted"`
	Password          string         `json:"password,omitempty" gorm:"-"`
	HasPassword       bool           `json:"has_password" gorm:"-"`
	RemoteName        string         `json:"remote_name"`
	RemoteID          string         `json:"remote_id"`
	Status            string         `json:"status" gorm:"default:pending"`
	OperationalStatus string         `json:"operational_status" gorm:"default:unknown;index"`
	LastSyncedAt      *time.Time     `json:"last_synced_at"`
	LastOnlineAt      *time.Time     `json:"last_online_at"`
	LastOfflineAt     *time.Time     `json:"last_offline_at"`
	LastIPAddress     string         `json:"last_ip_address"`
	LastCallerID      string         `json:"last_caller_id"`
	LastSessionUptime string         `json:"last_session_uptime"`
	LastRXBytes       int64          `json:"last_rx_bytes"`
	LastTXBytes       int64          `json:"last_tx_bytes"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	Subscription *Subscription `json:"subscription,omitempty" gorm:"foreignKey:SubscriptionID"`
	Router       *Router       `json:"router,omitempty" gorm:"foreignKey:RouterID"`
	NetworkPlan  *NetworkPlan  `json:"network_plan,omitempty" gorm:"foreignKey:NetworkPlanID"`
}

func (s *ServiceAccount) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
