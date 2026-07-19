package services

import (
	"testing"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
)

func TestShouldGenerateBillForMonthSkipsFutureSubscriptionStoredInUTC(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	subscription := models.Subscription{
		StartDate: time.Date(2026, time.August, 1, 0, 0, 0, 0, loc).UTC(),
	}

	julyBillMonth := time.Date(2026, time.July, 1, 0, 0, 0, 0, loc)
	if shouldGenerateBillForMonth(subscription, julyBillMonth, loc) {
		t.Fatal("expected August subscription stored in UTC to be skipped for July bills")
	}

	augustBillMonth := time.Date(2026, time.August, 1, 0, 0, 0, 0, loc)
	if !shouldGenerateBillForMonth(subscription, augustBillMonth, loc) {
		t.Fatal("expected August subscription stored in UTC to generate for August bills")
	}
}

func TestShouldGenerateBillForMonthSkipsSubscriptionEndedBeforeBillMonth(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	subscription := models.Subscription{
		StartDate: time.Date(2026, time.June, 1, 0, 0, 0, 0, loc),
		EndDate:   time.Date(2026, time.July, 31, 23, 59, 59, 0, loc).UTC(),
	}

	julyBillMonth := time.Date(2026, time.July, 1, 0, 0, 0, 0, loc)
	if !shouldGenerateBillForMonth(subscription, julyBillMonth, loc) {
		t.Fatal("expected subscription ending in July to generate for July bills")
	}

	augustBillMonth := time.Date(2026, time.August, 1, 0, 0, 0, 0, loc)
	if shouldGenerateBillForMonth(subscription, augustBillMonth, loc) {
		t.Fatal("expected subscription ending in July to be skipped for August bills")
	}
}

func TestDiagnoseMissingCurrentMonthBill(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	periodStart := time.Date(2026, time.July, 1, 0, 0, 0, 0, loc)
	availablePackage := &models.Package{ID: uuid.New(), Name: "Home 20 Mbps", Price: 200000}

	tests := []struct {
		name         string
		subscription models.Subscription
		wantCode     string
		wantEligible bool
	}{
		{
			name: "suspended",
			subscription: models.Subscription{
				Status:  "suspended",
				Package: availablePackage,
			},
			wantCode: "subscription_suspended",
		},
		{
			name: "not started",
			subscription: models.Subscription{
				Status:    "active",
				StartDate: time.Date(2026, time.August, 1, 0, 0, 0, 0, loc),
				Package:   availablePackage,
			},
			wantCode: "subscription_not_started",
		},
		{
			name: "ended",
			subscription: models.Subscription{
				Status:  "active",
				EndDate: time.Date(2026, time.June, 30, 23, 59, 59, 0, loc),
				Package: availablePackage,
			},
			wantCode: "subscription_ended",
		},
		{
			name: "missing package",
			subscription: models.Subscription{
				Status: "active",
			},
			wantCode: "package_missing",
		},
		{
			name: "eligible",
			subscription: models.Subscription{
				Status:    "active",
				StartDate: time.Date(2026, time.June, 1, 0, 0, 0, 0, loc),
				EndDate:   time.Date(2026, time.July, 31, 23, 59, 59, 0, loc),
				Package:   availablePackage,
			},
			wantCode:     "eligible_not_generated",
			wantEligible: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, reason, eligible := diagnoseMissingCurrentMonthBill(tt.subscription, periodStart, loc)
			if code != tt.wantCode {
				t.Fatalf("code = %q, want %q", code, tt.wantCode)
			}
			if reason == "" {
				t.Fatal("reason must not be empty")
			}
			if eligible != tt.wantEligible {
				t.Fatalf("eligible = %v, want %v", eligible, tt.wantEligible)
			}
		})
	}
}
