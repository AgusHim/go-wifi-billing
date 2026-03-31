package services

import (
	"strings"

	"github.com/Agushim/go_wifi_billing/lib"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type SubscriptionService interface {
	Create(subscription *models.Subscription) error
	GetAll(page int, limit int, search string, customerID *string, status *string) ([]models.Subscription, int64, error)
	GetByID(id uuid.UUID) (*models.Subscription, error)
	FindByCustomerID(customerID string) ([]models.Subscription, error)
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
	if subscription.NetworkPlanID != nil && subscription.ServiceType == "" && subscription.NetworkPlan != nil {
		subscription.ServiceType = subscription.NetworkPlan.ServiceType
	}
	if subscription.RenewalMode == "" {
		subscription.RenewalMode = "manual_invoice"
	}
	if strings.TrimSpace(strings.ToLower(subscription.RecurringProvider)) == "none" {
		subscription.RecurringProvider = ""
	}
	if strings.TrimSpace(subscription.RecurringToken) != "" {
		encrypted, err := lib.EncryptSecret(subscription.RecurringToken)
		if err != nil {
			return err
		}
		subscription.RecurringTokenEncrypted = encrypted
		subscription.RecurringToken = ""
	}
	return s.repo.Create(subscription)
}

func (s *subscriptionService) GetAll(page int, limit int, search string, customerID *string, status *string) ([]models.Subscription, int64, error) {
	items, total, err := s.repo.FindAll(page, limit, search, customerID, status)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		items[i] = *sanitizeSubscription(&items[i])
	}
	return items, total, nil
}

func (s *subscriptionService) GetByID(id uuid.UUID) (*models.Subscription, error) {
	item, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	return sanitizeSubscription(item), nil
}
func (s *subscriptionService) FindByCustomerID(customerID string) ([]models.Subscription, error) {
	customerIDPtr := &customerID
	subs, _, err := s.repo.FindAll(1, 9999, "", customerIDPtr, nil)
	if err != nil {
		return nil, err
	}
	for i := range subs {
		subs[i] = *sanitizeSubscription(&subs[i])
	}
	return subs, nil
}

func (s *subscriptionService) Update(id uuid.UUID, input *models.Subscription) (*models.Subscription, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	existing.CustomerID = input.CustomerID
	existing.PackageID = input.PackageID
	existing.NetworkPlanID = input.NetworkPlanID
	existing.ServiceType = input.ServiceType
	existing.PeriodType = input.PeriodType
	existing.PaymentType = input.PaymentType
	existing.StartDate = input.StartDate
	existing.EndDate = input.EndDate
	existing.AutoRenew = input.AutoRenew
	existing.RenewalMode = input.RenewalMode
	existing.RecurringConsent = input.RecurringConsent
	existing.RecurringProvider = input.RecurringProvider
	if strings.TrimSpace(strings.ToLower(existing.RecurringProvider)) == "none" {
		existing.RecurringProvider = ""
	}
	existing.IsActiveUniqueCode = input.IsActiveUniqueCode
	existing.IsIncludePPN = input.IsIncludePPN
	existing.Status = input.Status
	existing.Description = input.Description
	existing.DueDay = input.DueDay
	if strings.TrimSpace(input.RecurringToken) != "" {
		encrypted, err := lib.EncryptSecret(input.RecurringToken)
		if err != nil {
			return nil, err
		}
		existing.RecurringTokenEncrypted = encrypted
	}

	err = s.repo.Update(existing)
	if err != nil {
		return nil, err
	}

	return sanitizeSubscription(existing), nil
}

func (s *subscriptionService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}

func sanitizeSubscription(input *models.Subscription) *models.Subscription {
	if input == nil {
		return nil
	}
	copy := *input
	copy.HasRecurringToken = strings.TrimSpace(copy.RecurringTokenEncrypted) != ""
	copy.RecurringToken = ""
	return &copy
}
