package services

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type OdpService interface {
	Create(odp *models.Odp) error
	GetAll() ([]models.Odp, error)
	GetByID(id uuid.UUID) (*models.Odp, error)
	Update(id uuid.UUID, input *models.Odp) error
	Delete(id uuid.UUID) error
}

type odpService struct {
	repo repositories.OdpRepository
}

func NewOdpService(repo repositories.OdpRepository) OdpService {
	return &odpService{repo}
}

func (s *odpService) Create(odp *models.Odp) error {
	return s.repo.Create(odp)
}

func (s *odpService) GetAll() ([]models.Odp, error) {
	return s.repo.FindAll()
}

func (s *odpService) GetByID(id uuid.UUID) (*models.Odp, error) {
	return s.repo.FindByID(id)
}

func (s *odpService) Update(id uuid.UUID, input *models.Odp) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	existing.OdcID = input.OdcID
	existing.OdcPortNumber = input.OdcPortNumber
	existing.CoverageID = input.CoverageID
	existing.FoTubeColor = input.FoTubeColor
	existing.PoleNumber = input.PoleNumber
	existing.CountPort = input.CountPort
	existing.Description = input.Description
	existing.ImageURL = input.ImageURL
	existing.Latitude = input.Latitude
	existing.Longitude = input.Longitude

	return s.repo.Update(existing)
}

func (s *odpService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}
