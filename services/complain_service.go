package services

import (
	"errors"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type ComplainService interface {
	Create(input *models.Complain) (*models.Complain, error)
	GetAll() ([]models.Complain, error)
	GetByID(id uuid.UUID) (*models.Complain, error)
	Update(id uuid.UUID, input *models.Complain) (*models.Complain, error)
	Delete(id uuid.UUID) error
}

type complainService struct {
	repo repositories.ComplainRepository
}

func NewComplainService(repo repositories.ComplainRepository) ComplainService {
	return &complainService{repo: repo}
}

func (s *complainService) Create(input *models.Complain) (*models.Complain, error) {
	if err := s.repo.Create(input); err != nil {
		return nil, err
	}
	created, err := s.repo.GetByID(input.ID)
	if err != nil {
		return nil, err
	}
	if created != nil {
		return created, nil
	}
	return input, nil
}

func (s *complainService) GetAll() ([]models.Complain, error) {
	return s.repo.GetAll()
}

func (s *complainService) GetByID(id uuid.UUID) (*models.Complain, error) {
	complain, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if complain == nil {
		return nil, errors.New("complain not found")
	}
	return complain, nil
}

func (s *complainService) Update(id uuid.UUID, input *models.Complain) (*models.Complain, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("complain not found")
	}

	existing.CustomerID = input.CustomerID
	existing.SubscriptionID = input.SubscriptionID
	existing.ComplaintType = input.ComplaintType
	existing.Description = input.Description
	existing.Status = input.Status
	existing.Priority = input.Priority
	existing.TechnicianID = input.TechnicianID
	existing.ResolutionNote = input.ResolutionNote
	existing.ResolvedAt = input.ResolvedAt

	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *complainService) Delete(id uuid.UUID) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("complain not found")
	}
	return s.repo.Delete(id)
}

