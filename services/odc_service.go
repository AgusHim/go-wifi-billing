package services

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type OdcService interface {
	Create(odc *models.Odc) error
	GetAll() ([]models.Odc, error)
	GetByID(id uuid.UUID) (*models.Odc, error)
	Update(id uuid.UUID, input *models.Odc) error
	Delete(id uuid.UUID) error
}

type odcService struct {
	repo repositories.OdcRepository
}

func NewOdcService(repo repositories.OdcRepository) OdcService {
	return &odcService{repo}
}

func (s *odcService) Create(odc *models.Odc) error {
	return s.repo.Create(odc)
}

func (s *odcService) GetAll() ([]models.Odc, error) {
	return s.repo.FindAll()
}

func (s *odcService) GetByID(id uuid.UUID) (*models.Odc, error) {
	return s.repo.FindByID(id)
}

func (s *odcService) Update(id uuid.UUID, input *models.Odc) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	existing.CoverageID = input.CoverageID
	existing.OdcKey = input.OdcKey
	existing.Code = input.Code
	existing.PortOlt = input.PortOlt
	existing.FoColor = input.FoColor
	existing.PoleNumber = input.PoleNumber
	existing.CountPort = input.CountPort
	existing.Description = input.Description
	existing.ImageURL = input.ImageURL
	existing.Latitude = input.Latitude
	existing.Longitude = input.Longitude

	return s.repo.Update(existing)
}

func (s *odcService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}
