package services

import (
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type SubscriptionService interface {
	Create(subscription *models.Subscription) error
	GetAll(page int, limit int, search string, customerID *string, status *string) ([]models.Subscription, int64, error)
	GetByID(id uuid.UUID) (*models.Subscription, error)
	Update(id uuid.UUID, input *models.Subscription) (*models.Subscription, error)
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

func (s *subscriptionService) GetAll(page int, limit int, search string, customerID *string, status *string) ([]models.Subscription, int64, error) {
	return s.repo.FindAll(page, limit, search, customerID, status)
}

func (s *subscriptionService) GetByID(id uuid.UUID) (*models.Subscription, error) {
	return s.repo.FindByID(id)
}

func (s *subscriptionService) Update(id uuid.UUID, input *models.Subscription) (*models.Subscription, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	existing.CustomerID = input.CustomerID
	existing.PackageID = input.PackageID
	existing.StartDate = input.StartDate
	existing.EndDate = input.EndDate
	existing.AutoRenew = input.AutoRenew
	existing.IsActiveUniqueCode = input.IsActiveUniqueCode
	existing.IsIncludePPN = input.IsIncludePPN
	existing.Status = input.Status
	existing.Description = input.Description

	err = s.repo.Update(existing)
	if err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *subscriptionService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}
