package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Complain struct {
	ID             uuid.UUID     `json:"id" gorm:"type:uuid;primaryKey"`
	CustomerID     uuid.UUID     `json:"customer_id" gorm:"type:uuid;not null"`
	SubscriptionID uuid.UUID     `json:"subscription_id" gorm:"type:uuid;not null"`
	ComplaintType  string        `json:"complaint_type"`
	Description    string        `json:"description"`
	Status         string        `json:"status"`
	Priority       string        `json:"priority"`
	TechnicianID   *uuid.UUID    `json:"technician_id" gorm:"type:uuid"`
	ResolutionNote string        `json:"resolution_note"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	ResolvedAt     *time.Time    `json:"resolved_at"`

	Customer     *Customer     `json:"customer" gorm:"foreignKey:CustomerID"`
	Subscription *Subscription `json:"subscription" gorm:"foreignKey:SubscriptionID"`
	Technician   *User         `json:"technician" gorm:"foreignKey:TechnicianID"`
}

func (c *Complain) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return
}

