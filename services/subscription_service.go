package services

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type SubscriptionService interface {
	Create(subscription *models.Subscription) error
	GetAll() ([]models.Subscription, error)
	GetByID(id uuid.UUID) (*models.Subscription, error)
	Update(id uuid.UUID, input *models.Subscription) error
	Delete(id uuid.UUID) error
}

type subscriptionService struct {
	repo repositories.SubscriptionRepository
}

func NewSubscriptionService(repo repositories.SubscriptionRepository) SubscriptionService {
	return &subscriptionService{repo}
}

func (s *subscriptionService) Create(subscription *models.Subscription) error {
	return s.repo.Create(subscription)
}

func (s *subscriptionService) GetAll() ([]models.Subscription, error) {
	return s.repo.FindAll()
}

func (s *subscriptionService) GetByID(id uuid.UUID) (*models.Subscription, error) {
	return s.repo.FindByID(id)
}

func (s *subscriptionService) Update(id uuid.UUID, input *models.Subscription) error {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	existing.UserID = input.UserID
	existing.PackageID = input.PackageID
	existing.StartDate = input.StartDate
	existing.EndDate = input.EndDate
	existing.AutoRenew = input.AutoRenew
	existing.Status = input.Status
	existing.Description = input.Description

	return s.repo.Update(existing)
}

func (s *subscriptionService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}
