package services

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type WhatsAppTemplateService interface {
	Create(template *models.WhatsAppTemplate) error
	GetAll() ([]models.WhatsAppTemplate, error)
	GetByID(id uuid.UUID) (*models.WhatsAppTemplate, error)
	Update(template *models.WhatsAppTemplate) error
	Delete(id uuid.UUID) error
}

type whatsAppTemplateService struct {
	repo repositories.WhatsAppTemplateRepository
}

func NewWhatsAppTemplateService(repo repositories.WhatsAppTemplateRepository) WhatsAppTemplateService {
	return &whatsAppTemplateService{repo: repo}
}

func (s *whatsAppTemplateService) Create(template *models.WhatsAppTemplate) error {
	return s.repo.Create(template)
}

func (s *whatsAppTemplateService) GetAll() ([]models.WhatsAppTemplate, error) {
	return s.repo.GetAll()
}

func (s *whatsAppTemplateService) GetByID(id uuid.UUID) (*models.WhatsAppTemplate, error) {
	return s.repo.GetByID(id)
}

func (s *whatsAppTemplateService) Update(template *models.WhatsAppTemplate) error {
	return s.repo.Update(template)
}

func (s *whatsAppTemplateService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}
