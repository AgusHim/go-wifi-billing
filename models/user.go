package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name       string         `gorm:"type:varchar(255)" json:"name"`
	Email      string         `gorm:"uniqueIndex;type:varchar(255)" json:"email"`
	Phone      string         `gorm:"type:varchar(50)" json:"phone"`
	Password   string         `gorm:"type:varchar(255)" json:"-"`
	Role       string         `gorm:"type:varchar(50)" json:"role"`
	CustomerID *uuid.UUID     `gorm:"type:uuid; default:null" json:"customer_id"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Customer Customer `gorm:"foreignKey:CustomerID" json:"customer"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}
