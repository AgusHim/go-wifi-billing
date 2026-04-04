package repositories

import (
	"errors"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WhatsAppTemplateRepository interface {
	Create(template *models.WhatsAppTemplate) error
	GetAll() ([]models.WhatsAppTemplate, error)
	GetByID(id uuid.UUID) (*models.WhatsAppTemplate, error)
	Update(template *models.WhatsAppTemplate) error
	Delete(id uuid.UUID) error
}

type whatsAppTemplateRepository struct {
	db *gorm.DB
}

func NewWhatsAppTemplateRepository(db *gorm.DB) WhatsAppTemplateRepository {
	return &whatsAppTemplateRepository{db: db}
}

func (r *whatsAppTemplateRepository) Create(template *models.WhatsAppTemplate) error {
	return r.db.Create(template).Error
}

func (r *whatsAppTemplateRepository) GetAll() ([]models.WhatsAppTemplate, error) {
	var templates []models.WhatsAppTemplate
	if err := r.db.Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *whatsAppTemplateRepository) GetByID(id uuid.UUID) (*models.WhatsAppTemplate, error) {
	var template models.WhatsAppTemplate
	if err := r.db.First(&template, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &template, nil
}

func (r *whatsAppTemplateRepository) Update(template *models.WhatsAppTemplate) error {
	return r.db.Save(template).Error
}

func (r *whatsAppTemplateRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.WhatsAppTemplate{}, "id = ?", id).Error
}
