package repositories

import (
	"testing"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFindWithoutBillForPeriodHandlesPeriodFieldsLegacyBillsAndSoftDeletes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Coverage{},
		&models.Customer{},
		&models.Package{},
		&models.NetworkPlan{},
		&models.Subscription{},
		&models.Bill{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	periodStart := time.Date(2026, time.July, 1, 0, 0, 0, 0, loc)
	periodEnd := periodStart.AddDate(0, 1, 0)

	withoutBill := createMissingBillTestSubscription(t, db, periodStart)
	withPeriodBill := createMissingBillTestSubscription(t, db, periodStart.Add(time.Minute))
	withLegacyBill := createMissingBillTestSubscription(t, db, periodStart.Add(2*time.Minute))
	withPreviousBill := createMissingBillTestSubscription(t, db, periodStart.Add(3*time.Minute))
	withDeletedBill := createMissingBillTestSubscription(t, db, periodStart.Add(4*time.Minute))

	createMissingBillTestBill(t, db, withPeriodBill, intPtr(2026), intPtr(7), periodStart.AddDate(0, -1, 0), false)
	createMissingBillTestBill(t, db, withLegacyBill, nil, nil, periodStart.AddDate(0, 0, 5), false)
	createMissingBillTestBill(t, db, withPreviousBill, intPtr(2026), intPtr(6), periodStart.AddDate(0, -1, 0), false)
	createMissingBillTestBill(t, db, withDeletedBill, intPtr(2026), intPtr(7), periodStart, true)

	items, total, err := NewSubscriptionRepository(db).FindWithoutBillForPeriod(
		1,
		20,
		"",
		nil,
		nil,
		2026,
		7,
		periodStart,
		periodEnd,
	)
	if err != nil {
		t.Fatalf("find subscriptions without bill: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}

	got := make(map[uuid.UUID]bool, len(items))
	for _, item := range items {
		got[item.ID] = true
	}
	for _, expectedID := range []uuid.UUID{withoutBill, withPreviousBill, withDeletedBill} {
		if !got[expectedID] {
			t.Errorf("expected subscription %s in result", expectedID)
		}
	}
	for _, excludedID := range []uuid.UUID{withPeriodBill, withLegacyBill} {
		if got[excludedID] {
			t.Errorf("subscription %s with current period bill must be excluded", excludedID)
		}
	}
}

func createMissingBillTestSubscription(t *testing.T, db *gorm.DB, createdAt time.Time) uuid.UUID {
	t.Helper()
	subscription := models.Subscription{
		CustomerID: uuid.New(),
		PackageID:  uuid.New(),
		Status:     "active",
		CreatedAt:  createdAt,
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	return subscription.ID
}

func createMissingBillTestBill(t *testing.T, db *gorm.DB, subscriptionID uuid.UUID, year, month *int, billDate time.Time, deleted bool) {
	t.Helper()
	bill := models.Bill{
		ID:             uuid.New(),
		PublicID:       uuid.NewString(),
		SubscriptionID: subscriptionID,
		CustomerID:     uuid.New(),
		BillDate:       billDate,
		DueDate:        billDate.AddDate(0, 0, 10),
		PeriodYear:     year,
		PeriodMonth:    month,
		Status:         "unpaid",
	}
	if err := db.Create(&bill).Error; err != nil {
		t.Fatalf("create bill: %v", err)
	}
	if deleted {
		if err := db.Delete(&bill).Error; err != nil {
			t.Fatalf("soft delete bill: %v", err)
		}
	}
}
