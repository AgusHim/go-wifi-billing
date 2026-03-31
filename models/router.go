package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Router struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	Name              string         `json:"name"`
	Host              string         `json:"host"`
	Port              int            `json:"port"`
	Username          string         `json:"username"`
	PasswordEncrypted string         `json:"-" gorm:"column:password_encrypted"`
	Password          string         `json:"password,omitempty" gorm:"-"`
	HasPassword       bool           `json:"has_password" gorm:"-"`
	APIType           string         `json:"api_type"`
	UseTLS            bool           `json:"use_tls"`
	Location          string         `json:"location"`
	Status            string         `json:"status" gorm:"default:unknown"`
	LastSeenAt        *time.Time     `json:"last_seen_at"`
	LastError         string         `json:"last_error"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

func (r *Router) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
