package services

import (
	"testing"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
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
