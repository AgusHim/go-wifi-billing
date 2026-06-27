package repositories

import (
	"testing"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetDashboardStatsCountsOnlyCurrentActiveSubscriptionBillsWithExclusiveStatuses(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.Subscription{}, &models.Bill{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 12, 0, 0, 0, time.Local)
	futureDueDate := now.AddDate(0, 0, 1)
	pastDueDate := now.AddDate(0, 0, -1)

	activePaid := createDashboardStatsSubscription(t, db, "active", false)
	activeUnpaid := createDashboardStatsSubscription(t, db, "active", false)
	activeUnpaidPastDue := createDashboardStatsSubscription(t, db, "active", false)
	activeOverdue := createDashboardStatsSubscription(t, db, "active", false)
	suspendedWithBill := createDashboardStatsSubscription(t, db, "suspended", false)
	deletedWithBill := createDashboardStatsSubscription(t, db, "active", true)

	createDashboardStatsBill(t, db, activePaid.ID, "paid", startOfMonth, pastDueDate)
	createDashboardStatsBill(t, db, activeUnpaid.ID, "unpaid", startOfMonth, futureDueDate)
	createDashboardStatsBill(t, db, activeUnpaidPastDue.ID, "unpaid", startOfMonth, pastDueDate)
	createDashboardStatsBill(t, db, activeOverdue.ID, "overdue", startOfMonth, pastDueDate)
	createDashboardStatsBill(t, db, suspendedWithBill.ID, "paid", startOfMonth, pastDueDate)
	createDashboardStatsBill(t, db, deletedWithBill.ID, "paid", startOfMonth, pastDueDate)

	stats, err := NewBillRepository(db).GetDashboardStats()
	if err != nil {
		t.Fatalf("get dashboard stats: %v", err)
	}

	if stats["total_subscriptions"] != 4 {
		t.Fatalf("total_subscriptions = %d, want 4", stats["total_subscriptions"])
	}
	if stats["paid_bills"] != 1 {
		t.Fatalf("paid_bills = %d, want 1", stats["paid_bills"])
	}
	if stats["unpaid_bills"] != 1 {
		t.Fatalf("unpaid_bills = %d, want 1", stats["unpaid_bills"])
	}
	if stats["overdue_bills"] != 2 {
		t.Fatalf("overdue_bills = %d, want 2", stats["overdue_bills"])
	}
	if totalBills := stats["paid_bills"] + stats["unpaid_bills"] + stats["overdue_bills"]; totalBills != stats["total_subscriptions"] {
		t.Fatalf("total bills = %d, want total_subscriptions %d", totalBills, stats["total_subscriptions"])
	}
}

func TestGetDashboardChartRowsCountsOnlyActiveSubscriptions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.Subscription{}, &models.Bill{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	now := time.Now()
	activeSubscription := createDashboardStatsSubscription(t, db, "active", false)
	terminatedSubscription := createDashboardStatsSubscription(t, db, "terminated", false)
	deletedSubscription := createDashboardStatsSubscription(t, db, "active", true)

	createDashboardStatsBill(t, db, activeSubscription.ID, "paid", now, now)
	createDashboardStatsBill(t, db, terminatedSubscription.ID, "paid", now, now)
	createDashboardStatsBill(t, db, deletedSubscription.ID, "paid", now, now)

	rows, err := NewBillRepository(db).GetDashboardChartRows(now.AddDate(0, 0, -1), nil)
	if err != nil {
		t.Fatalf("get dashboard chart rows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("chart rows = %d, want 1", len(rows))
	}
	if rows[0].SubscriptionID != activeSubscription.ID {
		t.Fatalf("chart row subscription_id = %s, want %s", rows[0].SubscriptionID, activeSubscription.ID)
	}
}

func createDashboardStatsSubscription(t *testing.T, db *gorm.DB, status string, deleted bool) models.Subscription {
	t.Helper()

	subscription := models.Subscription{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
		PackageID:  uuid.New(),
		Status:     status,
	}
	if deleted {
		subscription.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	return subscription
}

func createDashboardStatsBill(t *testing.T, db *gorm.DB, subscriptionID uuid.UUID, status string, billDate time.Time, dueDate time.Time) {
	t.Helper()

	bill := models.Bill{
		ID:             uuid.New(),
		PublicID:       uuid.NewString(),
		SubscriptionID: subscriptionID,
		CustomerID:     uuid.New(),
		BillDate:       billDate,
		DueDate:        dueDate,
		Status:         status,
	}
	if err := db.Create(&bill).Error; err != nil {
		t.Fatalf("create bill: %v", err)
	}
}
