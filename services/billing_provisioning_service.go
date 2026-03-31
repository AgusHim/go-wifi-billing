package services

import (
	"log"
	"os"
	"strings"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

type BillingProvisioningService interface {
	HandlePaymentConfirmed(payment *models.Payment, bill *models.Bill, subscription *models.Subscription)
	HandleBillOverdue(bill *models.Bill, subscription *models.Subscription)
}

type billingProvisioningService struct {
	serviceAccountRepo repositories.ServiceAccountRepository
	serviceAccountSvc  ServiceAccountService
	subscriptionRepo   repositories.SubscriptionRepository
	historyRepo        repositories.ServiceStatusHistoryRepository
}

func NewBillingProvisioningService(
	serviceAccountRepo repositories.ServiceAccountRepository,
	serviceAccountSvc ServiceAccountService,
	subscriptionRepo repositories.SubscriptionRepository,
	historyRepo repositories.ServiceStatusHistoryRepository,
) BillingProvisioningService {
	return &billingProvisioningService{
		serviceAccountRepo: serviceAccountRepo,
		serviceAccountSvc:  serviceAccountSvc,
		subscriptionRepo:   subscriptionRepo,
		historyRepo:        historyRepo,
	}
}

func (s *billingProvisioningService) HandlePaymentConfirmed(payment *models.Payment, bill *models.Bill, subscription *models.Subscription) {
	if !billingProvisioningEnabled() || subscription == nil {
		return
	}

	if strings.TrimSpace(strings.ToLower(subscription.Status)) != "active" {
		subscription.Status = "active"
		if err := s.subscriptionRepo.Update(subscription); err != nil {
			log.Printf("[billing-provisioning] failed to mark subscription %s active: %v", subscription.ID, err)
		}
	}

	accounts, err := s.serviceAccountRepo.FindBySubscriptionID(subscription.ID.String())
	if err != nil {
		log.Printf("[billing-provisioning] failed to load service accounts for subscription %s: %v", subscription.ID, err)
		return
	}

	for _, account := range accounts {
		action := selectPaymentRecoveryAction(&account)
		if action == "" {
			continue
		}

		if action == "create_account" || action == "unsuspend_account" {
			previousStatus := account.Status
			account.Status = "reactivation_pending"
			if err := s.serviceAccountRepo.Update(&account); err != nil {
				log.Printf("[billing-provisioning] failed to mark service account %s reactivation_pending: %v", account.ID, err)
			} else {
				s.recordStatusHistory(account.ID, previousStatus, account.Status, action, "billing", "payment confirmed, waiting router reactivation", &bill.ID, &payment.ID)
			}
		}

		if _, err := s.serviceAccountSvc.EnqueueAction(account.ID, action); err != nil {
			log.Printf(
				"[billing-provisioning] failed to enqueue %s for payment %s bill %s service account %s: %v",
				action,
				payment.ID,
				bill.ID,
				account.ID,
				err,
			)
		}
	}
}

func (s *billingProvisioningService) recordStatusHistory(
	serviceAccountID uuid.UUID,
	previousStatus string,
	newStatus string,
	action string,
	source string,
	note string,
	billID *uuid.UUID,
	paymentID *uuid.UUID,
) {
	if s.historyRepo == nil {
		return
	}
	if strings.TrimSpace(previousStatus) == strings.TrimSpace(newStatus) {
		return
	}
	_ = s.historyRepo.Create(&models.ServiceStatusHistory{
		ServiceAccountID: serviceAccountID,
		PreviousStatus:   previousStatus,
		NewStatus:        newStatus,
		Action:           action,
		Source:           source,
		Note:             note,
		BillID:           billID,
		PaymentID:        paymentID,
	})
}

func (s *billingProvisioningService) HandleBillOverdue(bill *models.Bill, subscription *models.Subscription) {
	if !billingProvisioningEnabled() || subscription == nil {
		return
	}

	if strings.TrimSpace(strings.ToLower(subscription.Status)) != "suspended" {
		subscription.Status = "suspended"
		if err := s.subscriptionRepo.Update(subscription); err != nil {
			log.Printf("[billing-provisioning] failed to mark subscription %s suspended: %v", subscription.ID, err)
		}
	}

	accounts, err := s.serviceAccountRepo.FindBySubscriptionID(subscription.ID.String())
	if err != nil {
		log.Printf("[billing-provisioning] failed to load service accounts for overdue bill %s: %v", bill.ID, err)
		return
	}

	for _, account := range accounts {
		status := strings.TrimSpace(strings.ToLower(account.Status))
		if status == "suspended" || status == "terminated" {
			continue
		}

		if _, err := s.serviceAccountSvc.EnqueueAction(account.ID, "suspend_account"); err != nil {
			log.Printf(
				"[billing-provisioning] failed to enqueue suspend for overdue bill %s service account %s: %v",
				bill.ID,
				account.ID,
				err,
			)
		}
	}
}

func selectPaymentRecoveryAction(account *models.ServiceAccount) string {
	if account == nil {
		return ""
	}

	status := strings.TrimSpace(strings.ToLower(account.Status))
	switch status {
	case "", "pending":
		return "create_account"
	case "suspended", "reactivation_pending":
		return "unsuspend_account"
	case "terminated":
		if strings.TrimSpace(account.RemoteName) == "" && strings.TrimSpace(account.RemoteID) == "" {
			return "create_account"
		}
		return "unsuspend_account"
	default:
		return ""
	}
}

func billingProvisioningEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("FEATURE_BILLING_PROVISIONING_ENABLED")))
	return value == "1" || value == "true" || value == "yes"
}
