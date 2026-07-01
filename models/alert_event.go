package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AlertEvent struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	AlertRuleID      *uuid.UUID     `json:"alert_rule_id" gorm:"type:uuid;index"`
	RuleKey          string         `json:"rule_key" gorm:"index"`
	EntityType       string         `json:"entity_type" gorm:"index"`
	EntityID         *uuid.UUID     `json:"entity_id" gorm:"type:uuid;index"`
	RouterID         *uuid.UUID     `json:"router_id" gorm:"type:uuid;index"`
	ServiceAccountID *uuid.UUID     `json:"service_account_id" gorm:"type:uuid;index"`
	Severity         string         `json:"severity" gorm:"index"`
	Status           string         `json:"status" gorm:"default:open;index"`
	Title            string         `json:"title"`
	Description      string         `json:"description"`
	FirstSeenAt      time.Time      `json:"first_seen_at" gorm:"index"`
	LastSeenAt       time.Time      `json:"last_seen_at" gorm:"index"`
	AcknowledgedAt   *time.Time     `json:"acknowledged_at"`
	ResolvedAt       *time.Time     `json:"resolved_at"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	AlertRule      *AlertRule      `json:"alert_rule,omitempty" gorm:"foreignKey:AlertRuleID"`
	Router         *Router         `json:"router,omitempty" gorm:"foreignKey:RouterID"`
	ServiceAccount *ServiceAccount `json:"service_account,omitempty" gorm:"foreignKey:ServiceAccountID"`
}

func (a *AlertEvent) BeforeCreate(tx *gorm.DB) (err error) {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
