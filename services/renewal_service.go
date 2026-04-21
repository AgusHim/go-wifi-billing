package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/lib"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"gorm.io/gorm"
)

type RenewalService interface {
	StartScheduler()
	GenerateAutoRenewInvoices(referenceTime time.Time) (int, error)
	RecordPaymentConfirmed(subscriptionID uuid.UUID, billID uuid.UUID, paymentID uuid.UUID, note string) error
	ListHistory(subscriptionID uuid.UUID, limit int) ([]models.SubscriptionRenewalHistory, error)
	RunAutoGenerateNow() (int, error)
	SyncRecurringProfile(subscriptionID uuid.UUID) error
}

type renewalService struct {
	subRepo     repositories.SubscriptionRepository
	billRepo    repositories.BillRepository
	historyRepo repositories.SubscriptionRenewalHistoryRepository
}

func NewRenewalService(
	subRepo repositories.SubscriptionRepository,
	billRepo repositories.BillRepository,
	historyRepo repositories.SubscriptionRenewalHistoryRepository,
) RenewalService {
	return &renewalService{
		subRepo:     subRepo,
		billRepo:    billRepo,
		historyRepo: historyRepo,
	}
}

func (s *renewalService) StartScheduler() {
	if !renewalAutogenEnabled() {
		return
	}

	go func() {
		if _, err := s.GenerateAutoRenewInvoices(time.Now()); err != nil {
			log.Printf("[renewal] initial auto-generate run failed: %v", err)
		}

		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := s.GenerateAutoRenewInvoices(time.Now()); err != nil {
				log.Printf("[renewal] scheduled auto-generate run failed: %v", err)
			}
		}
	}()
}

func (s *renewalService) GenerateAutoRenewInvoices(referenceTime time.Time) (int, error) {
	if !renewalAutogenEnabled() {
		return 0, nil
	}

	threshold := referenceTime.AddDate(0, 0, 7)
	candidates, err := s.subRepo.FindAutoRenewCandidates(threshold, "auto_generate_invoice")
	if err != nil {
		return 0, err
	}

	createdCount := 0
	for _, sub := range candidates {
		targetMonth := int(sub.EndDate.Month())
		targetYear := sub.EndDate.Year()
		existing, err := s.billRepo.FindBillBySubscriptionAndMonth(sub.ID, targetMonth, targetYear)
		if err == nil && existing != nil {
			continue
		}
		if err != nil && !errorsIsRecordNotFound(err) {
			return createdCount, err
		}

		bill, err := buildRenewalBill(sub, referenceTime)
		if err != nil {
			return createdCount, err
		}
		if err := s.billRepo.Create(bill); err != nil {
			return createdCount, err
		}
		if err := s.historyRepo.Create(&models.SubscriptionRenewalHistory{
			SubscriptionID: sub.ID,
			BillID:         &bill.ID,
			Action:         "invoice_generated",
			Status:         "success",
			Note:           fmt.Sprintf("Auto-generated renewal invoice %s", bill.PublicID),
			ExecutedAt:     referenceTime,
		}); err != nil {
			return createdCount, err
		}
		createdCount++
	}

	return createdCount, nil
}

func (s *renewalService) RecordPaymentConfirmed(subscriptionID uuid.UUID, billID uuid.UUID, paymentID uuid.UUID, note string) error {
	return s.historyRepo.Create(&models.SubscriptionRenewalHistory{
		SubscriptionID: subscriptionID,
		BillID:         &billID,
		PaymentID:      &paymentID,
		Action:         "payment_confirmed",
		Status:         "success",
		Note:           strings.TrimSpace(note),
		ExecutedAt:     time.Now(),
	})
}

func (s *renewalService) ListHistory(subscriptionID uuid.UUID, limit int) ([]models.SubscriptionRenewalHistory, error) {
	return s.historyRepo.FindBySubscriptionID(subscriptionID, limit)
}

func (s *renewalService) RunAutoGenerateNow() (int, error) {
	return s.GenerateAutoRenewInvoices(time.Now())
}

func (s *renewalService) SyncRecurringProfile(subscriptionID uuid.UUID) error {
	subscription, err := s.subRepo.FindByID(subscriptionID)
	if err != nil {
		return err
	}

	if strings.TrimSpace(strings.ToLower(subscription.RenewalMode)) != "gateway_recurring" {
		if subscription.RecurringStatus != "disabled" {
			subscription.RecurringStatus = "disabled"
			subscription.RecurringLastSyncedAt = nil
			return s.subRepo.Update(subscription)
		}
		return nil
	}
	if !subscription.RecurringConsent {
		subscription.RecurringStatus = "consent_required"
		return s.subRepo.Update(subscription)
	}
	if !recurringPaymentsEnabled() {
		subscription.RecurringStatus = "integration_disabled"
		return s.subRepo.Update(subscription)
	}
	if strings.TrimSpace(strings.ToLower(subscription.RecurringProvider)) != "midtrans" {
		subscription.RecurringStatus = "provider_required"
		return s.subRepo.Update(subscription)
	}

	token, err := lib.DecryptSecret(subscription.RecurringTokenEncrypted)
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		subscription.RecurringStatus = "pending_token"
		return s.subRepo.Update(subscription)
	}
	if subscription.Package == nil || subscription.Customer == nil || subscription.Customer.User == nil {
		return errors.New("subscription is missing package or customer data for recurring sync")
	}

	req := s.buildMidtransSubscriptionRequest(subscription, token)
	client := &coreapi.Client{}
	client.New(os.Getenv("MIDTRANS_SERVER_KEY"), resolveMidtransEnvironment())

	var syncErr error
	if strings.TrimSpace(subscription.RecurringReferenceID) == "" {
		resp, err := client.CreateSubscription(req)
		if err != nil {
			syncErr = fmt.Errorf("midtrans create subscription failed: %w", err)
		} else {
			subscription.RecurringReferenceID = resp.ID
			subscription.RecurringStatus = strings.TrimSpace(resp.Status)
		}
	} else {
		resp, err := client.UpdateSubscription(subscription.RecurringReferenceID, req)
		if err != nil {
			syncErr = fmt.Errorf("midtrans update subscription failed: %w", err)
		} else {
			subscription.RecurringStatus = strings.TrimSpace(resp.StatusMessage)
		}
	}

	now := time.Now()
	subscription.RecurringLastSyncedAt = &now
	if syncErr != nil {
		subscription.RecurringStatus = "sync_failed"
		_ = s.subRepo.Update(subscription)
		_ = s.historyRepo.Create(&models.SubscriptionRenewalHistory{
			SubscriptionID: subscription.ID,
			Action:         "recurring_sync_failed",
			Status:         "failed",
			Note:           syncErr.Error(),
			ExecutedAt:     now,
		})
		return syncErr
	}

	if err := s.subRepo.Update(subscription); err != nil {
		return err
	}
	return s.historyRepo.Create(&models.SubscriptionRenewalHistory{
		SubscriptionID: subscription.ID,
		Action:         "recurring_profile_synced",
		Status:         "success",
		Note:           fmt.Sprintf("Recurring profile synced with provider %s", subscription.RecurringProvider),
		ExecutedAt:     now,
	})
}

func buildRenewalBill(sub models.Subscription, referenceTime time.Time) (*models.Bill, error) {
	if sub.Package == nil {
		return nil, fmt.Errorf("subscription %s has no package loaded", sub.ID)
	}

	targetMonth := sub.EndDate.Month()
	targetYear := sub.EndDate.Year()
	dueDate := time.Date(targetYear, targetMonth, sub.DueDay, 23, 59, 59, 0, time.Local)
	if dueDate.Month() != targetMonth {
		dueDate = time.Date(targetYear, targetMonth+1, 1, 23, 59, 59, 0, time.Local).AddDate(0, 0, -1)
	}

	amount := sub.Package.Price
	ppn := 0
	if sub.IsIncludePPN {
		ppn = int(float64(sub.Package.Price) * 0.11)
		amount += ppn
	}
	uniqueCode := 0
	if sub.IsActiveUniqueCode {
		n, err := rand.Int(rand.Reader, big.NewInt(500))
		if err == nil {
			uniqueCode = int(n.Int64()) + 1 // hasil 1–500
		}
		amount += uniqueCode
	}

	return &models.Bill{
		ID:             uuid.New(),
		PublicID:       fmt.Sprintf("%d%02d-%s", targetYear, targetMonth, uuid.NewString()[:6]),
		SubscriptionID: sub.ID,
		CustomerID:     sub.CustomerID,
		BillDate:       referenceTime,
		DueDate:        dueDate,
		Amount:         amount,
		PPN:            ppn,
		UniqueCode:     uniqueCode,
		Status:         "unpaid",
		CreatedAt:      referenceTime,
		UpdatedAt:      referenceTime,
	}, nil
}

func renewalAutogenEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("FEATURE_RENEWAL_AUTOGEN_ENABLED")))
	return value == "1" || value == "true" || value == "yes"
}

func recurringPaymentsEnabled() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("FEATURE_RECURRING_PAYMENT_ENABLED")))
	return value == "1" || value == "true" || value == "yes"
}

func errorsIsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func (s *renewalService) buildMidtransSubscriptionRequest(subscription *models.Subscription, token string) *coreapi.SubscriptionReq {
	amount := subscription.Package.Price
	if subscription.IsIncludePPN {
		amount += int(float64(subscription.Package.Price) * 0.11)
	}
	startTime := subscription.EndDate
	if startTime.Before(time.Now().Add(1 * time.Hour)) {
		startTime = time.Now().Add(1 * time.Hour)
	}

	return &coreapi.SubscriptionReq{
		Name:        fmt.Sprintf("sub-%s", subscription.ID.String()),
		Amount:      int64(amount),
		Currency:    "IDR",
		PaymentType: coreapi.PaymentTypeCreditCard,
		Token:       token,
		Schedule: coreapi.ScheduleDetails{
			Interval:     1,
			IntervalUnit: "month",
			MaxInterval:  120,
			StartTime:    startTime.Format("2006-01-02 15:04:05 -0700"),
		},
		Metadata: map[string]string{
			"subscription_id": subscription.ID.String(),
		},
		CustomerDetails: &midtrans.CustomerDetails{
			FName: subscription.Customer.User.Name,
			Email: subscription.Customer.User.Email,
			Phone: subscription.Customer.User.Phone,
		},
	}
}

func resolveMidtransEnvironment() midtrans.EnvironmentType {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("MIDTRANS_ENVIRONMENT")))
	if value == "" {
		value = strings.TrimSpace(strings.ToLower(os.Getenv("MIDTRANS_ENV")))
	}
	if value == "sandbox" {
		return midtrans.Sandbox
	}
	return midtrans.Production
}
