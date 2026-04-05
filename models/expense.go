package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Expense struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Title              string         `json:"title"`
	Category           string         `json:"category"`
	Amount             int            `json:"amount"`
	ExpenseDate        time.Time      `json:"expense_date"`
	Description        string         `json:"description"`
	ProofImageUrl      string         `json:"proof_image_url"`
	ProofImagePublicId string         `json:"proof_image_public_id"`
	AdminID            *uuid.UUID     `gorm:"type:uuid" json:"admin_id"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	Admin User `json:"admin" gorm:"foreignKey:AdminID"`
}
