package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CoverageID        *uuid.UUID     `gorm:"type:uuid" json:"coverage_id"`
	RoleID            *uuid.UUID     `gorm:"type:uuid;index" json:"role_id,omitempty"`
	Name              string         `gorm:"type:varchar(255)" json:"name"`
	Email             string         `gorm:"uniqueIndex;type:varchar(255)" json:"email"`
	Phone             string         `gorm:"type:varchar(50)" json:"phone"`
	Password          string         `gorm:"type:varchar(255)" json:"-"`
	Role              string         `gorm:"type:varchar(50)" json:"role"`
	PermissionVersion int64          `gorm:"not null;default:1" json:"permission_version"`
	IsActive          bool           `gorm:"not null;default:true;index" json:"is_active"`
	CreatedAt         time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	Coverage       Coverage `gorm:"foreignKey:CoverageID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"coverage"`
	RoleDefinition *Role    `gorm:"foreignKey:RoleID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"role_definition,omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}
