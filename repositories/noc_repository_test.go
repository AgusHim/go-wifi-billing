package repositories

import (
	"testing"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCountOnlineSessionsUsesLatestSnapshotPerRouter(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.ServiceSessionSnapshot{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	repo := NewNOCRepository(db)
	routerID := uuid.New()
	firstCollect := time.Now().Add(-2 * time.Minute)
	secondCollect := firstCollect.Add(time.Minute)

	insertSessionSnapshot(t, db, routerID, "pppoe", "customer-1", firstCollect)
	insertSessionSnapshot(t, db, routerID, "pppoe", "customer-2", firstCollect)
	insertSessionSnapshot(t, db, routerID, "pppoe", "customer-1", secondCollect)
	insertSessionSnapshot(t, db, routerID, "pppoe", "customer-2", secondCollect)

	total, err := repo.CountOnlineSessionsSince(time.Now().Add(-15 * time.Minute))
	if err != nil {
		t.Fatalf("count online sessions: %v", err)
	}
	if total != 2 {
		t.Fatalf("total online sessions = %d, want 2", total)
	}

	pppoeTotal, err := repo.CountOnlineSessionsByTypeSince("pppoe", time.Now().Add(-15*time.Minute))
	if err != nil {
		t.Fatalf("count pppoe sessions: %v", err)
	}
	if pppoeTotal != 2 {
		t.Fatalf("pppoe online sessions = %d, want 2", pppoeTotal)
	}
}

func insertSessionSnapshot(t *testing.T, db *gorm.DB, routerID uuid.UUID, serviceType string, username string, collectedAt time.Time) {
	t.Helper()
	snapshot := models.ServiceSessionSnapshot{
		RouterID:    routerID,
		ServiceType: serviceType,
		Username:    username,
		Online:      true,
		CollectedAt: collectedAt,
	}
	if err := db.Create(&snapshot).Error; err != nil {
		t.Fatalf("create session snapshot: %v", err)
	}
}
