package services

import (
	"errors"

	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type CoverageService interface {
	Create(input dto.CoverageCreateDTO) (*models.Coverage, error)
	GetByID(id string) (*models.Coverage, error)
	GetAll() ([]models.Coverage, error)
	Update(id string, input dto.CoverageUpdateDTO) (*models.Coverage, error)
	Delete(id string) error
}

type coverageService struct {
	repo repositories.CoverageRepository
}

func NewCoverageService(r repositories.CoverageRepository) CoverageService {
	return &coverageService{repo: r}
}

func (s *coverageService) Create(input dto.CoverageCreateDTO) (*models.Coverage, error) {
	c := &models.Coverage{
		CodeArea:    input.CodeArea,
		Name:        input.Name,
		Address:     input.Address,
		Description: input.Description,
		RangeArea:   input.RangeArea,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
	}
	if err := s.repo.Create(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *coverageService) GetByID(id string) (*models.Coverage, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	c, err := s.repo.GetByID(uid)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, errors.New("coverage not found")
	}
	return c, nil
}

func (s *coverageService) GetAll() ([]models.Coverage, error) {
	return s.repo.GetAll()
}

func (s *coverageService) Update(id string, input dto.CoverageUpdateDTO) (*models.Coverage, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetByID(uid)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("coverage not found")
	}

	if input.CodeArea != nil {
		existing.CodeArea = *input.CodeArea
	}
	if input.Name != nil {
		existing.Name = *input.Name
	}
	if input.Address != nil {
		existing.Address = *input.Address
	}
	if input.Description != nil {
		existing.Description = *input.Description
	}
	if input.RangeArea != nil {
		existing.RangeArea = *input.RangeArea
	}
	if input.Latitude != nil {
		existing.Latitude = *input.Latitude
	}
	if input.Longitude != nil {
		existing.Longitude = *input.Longitude
	}

	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *coverageService) Delete(id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	// check exists
	existing, err := s.repo.GetByID(uid)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("coverage not found")
	}
	return s.repo.Delete(uid)
}
