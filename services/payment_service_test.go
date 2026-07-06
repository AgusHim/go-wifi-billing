package services

import (
	"testing"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeRenewalService struct{}

func (fakeRenewalService) StartScheduler() {}
func (fakeRenewalService) GenerateAutoRenewInvoices(time.Time) (int, error) {
	return 0, nil
}
func (fakeRenewalService) RecordPaymentConfirmed(uuid.UUID, uuid.UUID, uuid.UUID, string) error {
	return nil
}
func (fakeRenewalService) ListHistory(uuid.UUID, int) ([]models.SubscriptionRenewalHistory, error) {
	return nil, nil
}
func (fakeRenewalService) RunAutoGenerateNow() (int, error) {
	return 0, nil
}
func (fakeRenewalService) SyncRecurringProfile(uuid.UUID) error {
	return nil
}

func TestCreateConfirmedPaymentUpdatesBillingStateInTransaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Subscription{},
		&models.Bill{},
		&models.Payment{},
		&models.PaymentCallbackLog{},
		&models.SubscriptionRenewalHistory{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	periodStart := time.Date(2026, time.July, 1, 0, 0, 0, 0, loc)
	periodEnd := time.Date(2026, time.July, 31, 23, 59, 59, 0, loc)
	periodYear := 2026
	periodMonth := 7

	subscription := models.Subscription{
		ID:        uuid.New(),
		Status:    "suspended",
		StartDate: time.Date(2026, time.June, 1, 0, 0, 0, 0, loc),
		EndDate:   time.Date(2026, time.June, 30, 23, 59, 59, 0, loc),
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	bill := models.Bill{
		ID:             uuid.New(),
		PublicID:       "202607-test",
		SubscriptionID: subscription.ID,
		CustomerID:     uuid.New(),
		BillDate:       periodStart,
		DueDate:        time.Date(2026, time.July, 10, 23, 59, 59, 0, loc),
		Amount:         150000,
		Status:         "unpaid",
		PeriodYear:     &periodYear,
		PeriodMonth:    &periodMonth,
		PeriodStart:    &periodStart,
		PeriodEnd:      &periodEnd,
	}
	if err := db.Create(&bill).Error; err != nil {
		t.Fatalf("create bill: %v", err)
	}

	svc := NewPaymentService(
		repositories.NewPaymentRepository(db),
		repositories.NewSubscriptionRepository(db),
		repositories.NewBillRepository(db),
		nil,
		fakeRenewalService{},
		db,
	)

	payment, err := svc.Create(models.Payment{
		BillID:      bill.ID,
		PaymentDate: time.Now(),
		DueDate:     bill.DueDate,
		Method:      "cash",
		Amount:      bill.Amount,
		Status:      "confirmed",
	})
	if err != nil {
		t.Fatalf("create confirmed payment: %v", err)
	}

	var storedPayment models.Payment
	if err := db.First(&storedPayment, "id = ?", payment.ID).Error; err != nil {
		t.Fatalf("find payment: %v", err)
	}
	if storedPayment.Status != "confirmed" {
		t.Fatalf("payment status = %q, want confirmed", storedPayment.Status)
	}

	var storedBill models.Bill
	if err := db.First(&storedBill, "id = ?", bill.ID).Error; err != nil {
		t.Fatalf("find bill: %v", err)
	}
	if storedBill.Status != "paid" {
		t.Fatalf("bill status = %q, want paid", storedBill.Status)
	}
	if storedBill.PaidAt == nil {
		t.Fatal("paid_at is nil")
	}
	if storedBill.LastPaymentID == nil || *storedBill.LastPaymentID != payment.ID {
		t.Fatalf("last_payment_id = %v, want %s", storedBill.LastPaymentID, payment.ID)
	}

	var storedSubscription models.Subscription
	if err := db.First(&storedSubscription, "id = ?", subscription.ID).Error; err != nil {
		t.Fatalf("find subscription: %v", err)
	}
	if storedSubscription.Status != "active" {
		t.Fatalf("subscription status = %q, want active", storedSubscription.Status)
	}
	if !storedSubscription.StartDate.Equal(periodStart) {
		t.Fatalf("subscription start = %v, want %v", storedSubscription.StartDate, periodStart)
	}
	if !storedSubscription.EndDate.Equal(periodEnd) {
		t.Fatalf("subscription end = %v, want %v", storedSubscription.EndDate, periodEnd)
	}

	var historyCount int64
	if err := db.Model(&models.SubscriptionRenewalHistory{}).
		Where("subscription_id = ? AND bill_id = ? AND payment_id = ?", subscription.ID, bill.ID, payment.ID).
		Count(&historyCount).Error; err != nil {
		t.Fatalf("count renewal history: %v", err)
	}
	if historyCount != 1 {
		t.Fatalf("renewal history count = %d, want 1", historyCount)
	}
}

func TestRecordPaymentCallbackLogStoresRawPayload(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.Payment{}, &models.PaymentCallbackLog{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	svc := NewPaymentService(
		repositories.NewPaymentRepository(db),
		repositories.NewSubscriptionRepository(db),
		repositories.NewBillRepository(db),
		nil,
		nil,
		db,
	)

	paymentID := uuid.New()
	processedAt := time.Now()
	rawPayload := `{"order_id":"` + paymentID.String() + `","transaction_status":"settlement"}`
	err = svc.RecordPaymentCallbackLog(PaymentCallbackLogInput{
		Provider:          "midtrans",
		OrderID:           paymentID.String(),
		TransactionStatus: "settlement",
		FraudStatus:       "accept",
		GrossAmount:       "150000.00",
		SignatureValid:    true,
		RawPayload:        rawPayload,
		ReceivedAt:        time.Now(),
		ProcessedAt:       &processedAt,
	})
	if err != nil {
		t.Fatalf("record callback log: %v", err)
	}

	var callbackLog models.PaymentCallbackLog
	if err := db.First(&callbackLog, "order_id = ?", paymentID.String()).Error; err != nil {
		t.Fatalf("find callback log: %v", err)
	}
	if callbackLog.PaymentID == nil || *callbackLog.PaymentID != paymentID {
		t.Fatalf("payment_id = %v, want %s", callbackLog.PaymentID, paymentID)
	}
	if !callbackLog.SignatureValid {
		t.Fatal("signature_valid = false, want true")
	}
	if callbackLog.RawPayload != rawPayload {
		t.Fatalf("raw_payload = %q, want %q", callbackLog.RawPayload, rawPayload)
	}
}
