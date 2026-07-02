package services

import (
	"errors"
	"strings"

	"github.com/Agushim/go_wifi_billing/lib"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionService interface {
	Create(subscription *models.Subscription) error
	GetAll(page int, limit int, search string, customerID *string, status *string, customerDeleted *string, endDateFilter *string) ([]models.Subscription, int64, error)
	GetByID(id uuid.UUID) (*models.Subscription, error)
	FindByCustomerID(customerID string) ([]models.Subscription, error)
	Update(id uuid.UUID, input *models.Subscription) (*models.Subscription, error)
	Delete(id uuid.UUID) error
	DeleteByCustomerID(customerID uuid.UUID) error
}

type subscriptionService struct {
	repo               repositories.SubscriptionRepository
	customerRepo       repositories.CustomerRepository
	networkPlanRepo    repositories.NetworkPlanRepository
	serviceAccountRepo repositories.ServiceAccountRepository
}

func NewSubscriptionService(
	repo repositories.SubscriptionRepository,
	customerRepo repositories.CustomerRepository,
	networkPlanRepo repositories.NetworkPlanRepository,
	serviceAccountRepo repositories.ServiceAccountRepository,
) SubscriptionService {
	return &subscriptionService{
		repo:               repo,
		customerRepo:       customerRepo,
		networkPlanRepo:    networkPlanRepo,
		serviceAccountRepo: serviceAccountRepo,
	}
}

func (s *subscriptionService) Create(subscription *models.Subscription) error {
	if err := s.applyInferredNetworkPlan(subscription); err != nil {
		return err
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

func (s *subscriptionService) GetAll(page int, limit int, search string, customerID *string, status *string, customerDeleted *string, endDateFilter *string) ([]models.Subscription, int64, error) {
	items, total, err := s.repo.FindAll(page, limit, search, customerID, status, customerDeleted, endDateFilter)
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
	subs, _, err := s.repo.FindAll(1, 9999, "", customerIDPtr, nil, nil, nil)
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
	if err := s.applyInferredNetworkPlan(existing); err != nil {
		return nil, err
	}
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
	if existing.NetworkPlanID != nil && s.serviceAccountRepo != nil {
		if err := s.serviceAccountRepo.ClearNetworkPlanFallbacksBySubscriptionID(existing.ID); err != nil {
			return nil, err
		}
		existing, err = s.repo.FindByID(id)
		if err != nil {
			return nil, err
		}
	}

	return sanitizeSubscription(existing), nil
}

func (s *subscriptionService) Delete(id uuid.UUID) error {
	return s.repo.Delete(id)
}

func (s *subscriptionService) DeleteByCustomerID(customerID uuid.UUID) error {
	return s.repo.SoftDeleteByCustomerID(customerID)
}

func (s *subscriptionService) applyInferredNetworkPlan(subscription *models.Subscription) error {
	if subscription == nil {
		return nil
	}

	if subscription.NetworkPlanID != nil {
		if strings.TrimSpace(subscription.ServiceType) == "" && subscription.NetworkPlan != nil {
			subscription.ServiceType = subscription.NetworkPlan.ServiceType
		}
		return nil
	}

	customer := subscription.Customer
	if customer == nil && s.customerRepo != nil && subscription.CustomerID != uuid.Nil {
		found, err := s.customerRepo.FindByID(subscription.CustomerID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			customer = found
		}
	}

	if planID, serviceType, ok := inferNetworkPlanFromCustomerRelations(customer, subscription.ServiceType); ok {
		subscription.NetworkPlanID = &planID
		if strings.TrimSpace(subscription.ServiceType) == "" {
			subscription.ServiceType = serviceType
		}
		return nil
	}

	profileName := ""
	if customer != nil {
		profileName = customer.ProfilePPOE
	}
	if strings.TrimSpace(profileName) == "" || s.networkPlanRepo == nil {
		return nil
	}

	plans, err := s.networkPlanRepo.FindAll()
	if err != nil {
		return err
	}
	if plan := findNetworkPlanByProfile(plans, profileName, subscription.ServiceType); plan != nil {
		planID := plan.ID
		subscription.NetworkPlanID = &planID
		if strings.TrimSpace(subscription.ServiceType) == "" {
			subscription.ServiceType = plan.ServiceType
		}
	}

	return nil
}

func inferNetworkPlanFromCustomerRelations(customer *models.Customer, requestedServiceType string) (uuid.UUID, string, bool) {
	if customer == nil {
		return uuid.Nil, "", false
	}

	for _, subscription := range customer.Subscriptions {
		if subscription.NetworkPlan != nil && subscription.NetworkPlan.ID != uuid.Nil && serviceTypeMatches(subscription.NetworkPlan.ServiceType, requestedServiceType) {
			return subscription.NetworkPlan.ID, subscription.NetworkPlan.ServiceType, true
		}
		if subscription.NetworkPlanID != nil && serviceTypeMatches(subscription.ServiceType, requestedServiceType) {
			return *subscription.NetworkPlanID, subscription.ServiceType, true
		}
		for _, account := range subscription.ServiceAccounts {
			if account.NetworkPlan != nil && account.NetworkPlan.ID != uuid.Nil && serviceTypeMatches(account.NetworkPlan.ServiceType, requestedServiceType) {
				return account.NetworkPlan.ID, account.NetworkPlan.ServiceType, true
			}
			if account.NetworkPlanID != nil && serviceTypeMatches(account.ServiceType, requestedServiceType) {
				return *account.NetworkPlanID, account.ServiceType, true
			}
		}
	}

	return uuid.Nil, "", false
}

func findNetworkPlanByProfile(plans []models.NetworkPlan, profileName string, requestedServiceType string) *models.NetworkPlan {
	normalizedProfile := normalizeProfileName(profileName)
	if normalizedProfile == "" {
		return nil
	}

	requested := normalizeSubscriptionServiceType(requestedServiceType)
	preferredServiceType := requested
	if preferredServiceType == "" {
		preferredServiceType = "pppoe"
	}

	for i := range plans {
		if normalizeProfileName(plans[i].MikrotikProfileName) == normalizedProfile && normalizeSubscriptionServiceType(plans[i].ServiceType) == preferredServiceType {
			return &plans[i]
		}
	}
	for i := range plans {
		if normalizeProfileName(plans[i].MikrotikProfileName) == normalizedProfile && serviceTypeMatches(plans[i].ServiceType, requestedServiceType) {
			return &plans[i]
		}
	}

	return nil
}

func serviceTypeMatches(planServiceType string, requestedServiceType string) bool {
	planType := normalizeSubscriptionServiceType(planServiceType)
	requestedType := normalizeSubscriptionServiceType(requestedServiceType)
	return requestedType == "" || planType == "" || planType == requestedType
}

func normalizeSubscriptionServiceType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "ppp" {
		return "pppoe"
	}
	return normalized
}

func normalizeProfileName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
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
