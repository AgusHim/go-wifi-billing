package services

import (
	"errors"
	"strings"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type OdpService interface {
	Create(odp *models.Odp) error
	GetAll(coverageIDs []string) ([]models.Odp, error)
	GetByID(id uuid.UUID) (*models.Odp, error)
	Update(id uuid.UUID, input *models.Odp) (*models.Odp, error)
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

func (s *odpService) GetAll(coverageIDs []string) ([]models.Odp, error) {
	var parsedCoverageIDs []uuid.UUID
	for _, cid := range coverageIDs {
		cid = strings.TrimSpace(cid)
		if cid == "" {
			continue
		}
		id, err := uuid.Parse(cid)
		if err != nil {
			return nil, errors.New("invalid coverage_id")
		}
		parsedCoverageIDs = append(parsedCoverageIDs, id)
	}
	return s.repo.FindAll(parsedCoverageIDs)
}

func (s *odpService) GetByID(id uuid.UUID) (*models.Odp, error) {
	return s.repo.FindByID(id)
}

func (s *odpService) Update(id uuid.UUID, input *models.Odp) (*models.Odp, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	existing.OdcID = input.OdcID
	existing.Code = input.Code
	existing.OdcPortNumber = input.OdcPortNumber
	existing.CoverageID = input.CoverageID
	existing.FoTubeColor = input.FoTubeColor
	existing.PoleNumber = input.PoleNumber
	existing.CountPort = input.CountPort
	existing.Description = input.Description
	existing.ImageURL = input.ImageURL
	existing.Latitude = input.Latitude
	existing.Longitude = input.Longitude

	err = s.repo.Update(existing)
	if err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *odpService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}
