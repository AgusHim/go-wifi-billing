package repositories

import (
	"fmt"
	"testing"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFindAllFiltersCustomersBySubscriptionStatus(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Coverage{},
		&models.Odc{},
		&models.Odp{},
		&models.Customer{},
		&models.Package{},
		&models.Subscription{},
		&models.ServiceAccount{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	coverage := models.Coverage{Name: "Test Coverage", CodeArea: "TST"}
	if err := db.Create(&coverage).Error; err != nil {
		t.Fatalf("create coverage: %v", err)
	}
	packageData := models.Package{Name: "Home 20 Mbps", Price: 200000}
	if err := db.Create(&packageData).Error; err != nil {
		t.Fatalf("create package: %v", err)
	}

	withoutSubscription := createCustomerSubscriptionFilterTestCustomer(t, db, coverage.ID, 1)
	configured := createCustomerSubscriptionFilterTestCustomer(t, db, coverage.ID, 2)
	softDeletedSubscription := createCustomerSubscriptionFilterTestCustomer(t, db, coverage.ID, 3)
	suspendedSubscription := createCustomerSubscriptionFilterTestCustomer(t, db, coverage.ID, 4)
	terminatedSubscription := createCustomerSubscriptionFilterTestCustomer(t, db, coverage.ID, 5)

	createCustomerSubscriptionFilterTestSubscription(t, db, configured.ID, packageData.ID, "active", false)
	createCustomerSubscriptionFilterTestSubscription(t, db, softDeletedSubscription.ID, packageData.ID, "active", true)
	createCustomerSubscriptionFilterTestSubscription(t, db, suspendedSubscription.ID, packageData.ID, "suspended", false)
	createCustomerSubscriptionFilterTestSubscription(t, db, terminatedSubscription.ID, packageData.ID, "terminated", false)

	repository := NewCustomerRepository(db)
	missing, missingTotal, err := repository.FindAll(1, 20, "", nil, nil, "missing")
	if err != nil {
		t.Fatalf("find customers missing subscription: %v", err)
	}
	if missingTotal != 4 {
		t.Fatalf("missing total = %d, want 4", missingTotal)
	}
	assertCustomerFilterResult(t, missing, withoutSubscription.ID, true)
	assertCustomerFilterResult(t, missing, softDeletedSubscription.ID, true)
	assertCustomerFilterResult(t, missing, suspendedSubscription.ID, true)
	assertCustomerFilterResult(t, missing, terminatedSubscription.ID, true)
	assertCustomerFilterResult(t, missing, configured.ID, false)
	assertCustomerGapReason(t, missing, withoutSubscription.ID, "Belum pernah disetting subscription")
	assertCustomerGapReason(t, missing, softDeletedSubscription.ID, "Subscription sebelumnya sudah dihapus")
	assertCustomerGapReason(t, missing, suspendedSubscription.ID, "Subscription terakhir berstatus suspended")
	assertCustomerGapReason(t, missing, terminatedSubscription.ID, "Subscription terakhir berstatus terminated")

	configuredCustomers, configuredTotal, err := repository.FindAll(1, 20, "", nil, nil, "configured")
	if err != nil {
		t.Fatalf("find configured customers: %v", err)
	}
	if configuredTotal != 3 {
		t.Fatalf("configured total = %d, want 3", configuredTotal)
	}
	assertCustomerFilterResult(t, configuredCustomers, configured.ID, true)
}

func createCustomerSubscriptionFilterTestCustomer(t *testing.T, db *gorm.DB, coverageID uuid.UUID, sequence int) models.Customer {
	t.Helper()
	user := models.User{
		Name:  fmt.Sprintf("Customer %d", sequence),
		Email: fmt.Sprintf("customer-%d@example.com", sequence),
		Phone: fmt.Sprintf("08120000000%d", sequence),
		Role:  "customer",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	customer := models.Customer{
		UserID:        user.ID,
		CoverageID:    coverageID,
		ServiceNumber: fmt.Sprintf("TST-%06d", sequence),
		Status:        "active",
	}
	if err := db.Create(&customer).Error; err != nil {
		t.Fatalf("create customer: %v", err)
	}
	return customer
}

func createCustomerSubscriptionFilterTestSubscription(t *testing.T, db *gorm.DB, customerID, packageID uuid.UUID, status string, softDelete bool) {
	t.Helper()
	subscription := models.Subscription{
		CustomerID: customerID,
		PackageID:  packageID,
		Status:     status,
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	if softDelete {
		if err := db.Delete(&subscription).Error; err != nil {
			t.Fatalf("soft delete subscription: %v", err)
		}
	}
}

func assertCustomerGapReason(t *testing.T, customers []models.Customer, id uuid.UUID, want string) {
	t.Helper()
	for _, customer := range customers {
		if customer.ID == id {
			if customer.SubscriptionGapReason != want {
				t.Fatalf("customer %s gap reason = %q, want %q", id, customer.SubscriptionGapReason, want)
			}
			return
		}
	}
	t.Fatalf("customer %s not found", id)
}

func assertCustomerFilterResult(t *testing.T, customers []models.Customer, id uuid.UUID, want bool) {
	t.Helper()
	found := false
	for _, customer := range customers {
		if customer.ID == id {
			found = true
			break
		}
	}
	if found != want {
		t.Fatalf("customer %s found = %v, want %v", id, found, want)
	}
}
