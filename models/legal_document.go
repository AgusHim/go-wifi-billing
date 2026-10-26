package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LegalDocument is a Cantika.net legal/operational document that partners may
// need. A document either belongs to one partner or is global (PartnerID nil),
// in which case it applies to every partner. Visibility to partners is opt-in
// so nothing is exposed by accident.
type LegalDocument struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey"`
	PartnerID        *uuid.UUID     `json:"partner_id" gorm:"type:uuid;index"`
	Name             string         `json:"name" gorm:"type:varchar(255);not null"`
	Category         string         `json:"category" gorm:"type:varchar(80);index"`
	Version          string         `json:"version" gorm:"type:varchar(50)"`
	EffectiveDate    *time.Time     `json:"effective_date"`
	ExpiresAt        *time.Time     `json:"expires_at" gorm:"index"`
	FilePath         string         `json:"file_path" gorm:"type:text;not null"`
	FileSize         int64          `json:"file_size"`
	MimeType         string         `json:"mime_type" gorm:"type:varchar(120)"`
	Status           string         `json:"status" gorm:"type:varchar(20);not null;default:draft;index"` // draft, published, archived
	IsPartnerVisible bool           `json:"is_partner_visible" gorm:"not null;default:false;index"`
	Notes            string         `json:"notes" gorm:"type:text"`
	UploadedByID     *uuid.UUID     `json:"uploaded_by_id" gorm:"type:uuid"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	// PartnerName/PartnerCode are filled for listings only and are never
	// persisted. An empty name marks a document that applies to all partners.
	PartnerName string `json:"partner_name,omitempty" gorm:"-"`
	PartnerCode string `json:"partner_code,omitempty" gorm:"-"`
}

// IsGlobal reports whether the document applies to every partner.
func (d *LegalDocument) IsGlobal() bool { return d.PartnerID == nil }

// AppliesToPartner reports whether the document belongs to the given partner or
// applies to all partners.
func (d *LegalDocument) AppliesToPartner(partnerID uuid.UUID) bool {
	return d.PartnerID == nil || *d.PartnerID == partnerID
}

func (d *LegalDocument) BeforeCreate(_ *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
