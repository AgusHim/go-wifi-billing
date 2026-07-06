package repositories

import (
	"testing"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFindEligibleForRetry(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&models.ProvisioningJob{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	repo := NewProvisioningJobRepository(db)
	now := time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)
	past := now.Add(-time.Minute)
	future := now.Add(time.Minute)

	eligiblePending := models.ProvisioningJob{ID: uuid.New(), EntityType: "service_account", Status: "pending", AttemptCount: 0}
	eligibleFailed := models.ProvisioningJob{ID: uuid.New(), EntityType: "service_account", Status: "failed", AttemptCount: 1, ScheduledAt: &past}
	futureFailed := models.ProvisioningJob{ID: uuid.New(), EntityType: "service_account", Status: "failed", AttemptCount: 1, ScheduledAt: &future}
	maxedFailed := models.ProvisioningJob{ID: uuid.New(), EntityType: "service_account", Status: "failed", AttemptCount: 3, ScheduledAt: &past}
	successJob := models.ProvisioningJob{ID: uuid.New(), EntityType: "service_account", Status: "success", AttemptCount: 1, ScheduledAt: &past}

	for _, job := range []models.ProvisioningJob{eligiblePending, eligibleFailed, futureFailed, maxedFailed, successJob} {
		if err := db.Create(&job).Error; err != nil {
			t.Fatalf("create job %s: %v", job.ID, err)
		}
	}

	jobs, err := repo.FindEligibleForRetry(now, 3, 10)
	if err != nil {
		t.Fatalf("find eligible jobs: %v", err)
	}

	got := map[uuid.UUID]bool{}
	for _, job := range jobs {
		got[job.ID] = true
	}

	if !got[eligiblePending.ID] {
		t.Fatal("pending job without scheduled_at should be eligible")
	}
	if !got[eligibleFailed.ID] {
		t.Fatal("failed job with due scheduled_at should be eligible")
	}
	if got[futureFailed.ID] {
		t.Fatal("failed job with future scheduled_at should not be eligible")
	}
	if got[maxedFailed.ID] {
		t.Fatal("failed job at max attempts should not be eligible")
	}
	if got[successJob.ID] {
		t.Fatal("success job should not be eligible")
	}
}
