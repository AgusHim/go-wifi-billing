package services

import (
	"testing"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
)

func TestBuildRenewalBillUsesSubscriptionEndPeriod(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	sub := models.Subscription{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
		EndDate:    time.Date(2026, time.August, 31, 23, 59, 59, 0, loc),
		DueDay:     31,
		Package: &models.Package{
			Name:  "20 Mbps",
			Price: 150000,
		},
	}
	referenceTime := time.Date(2026, time.July, 25, 10, 0, 0, 0, loc)

	bill, err := buildRenewalBill(sub, referenceTime)
	if err != nil {
		t.Fatalf("build renewal bill: %v", err)
	}

	if bill.PeriodYear == nil || *bill.PeriodYear != 2026 {
		t.Fatalf("period_year = %v, want 2026", bill.PeriodYear)
	}
	if bill.PeriodMonth == nil || *bill.PeriodMonth != 8 {
		t.Fatalf("period_month = %v, want 8", bill.PeriodMonth)
	}
	if bill.PeriodStart == nil || !bill.PeriodStart.In(loc).Equal(time.Date(2026, time.August, 1, 0, 0, 0, 0, loc)) {
		t.Fatalf("period_start = %v, want August 1 2026 WIB", bill.PeriodStart)
	}
	if bill.PeriodEnd == nil || !bill.PeriodEnd.In(loc).Equal(time.Date(2026, time.August, 31, 23, 59, 59, 0, loc)) {
		t.Fatalf("period_end = %v, want August 31 2026 WIB", bill.PeriodEnd)
	}
	if !bill.DueDate.In(loc).Equal(time.Date(2026, time.August, 31, 23, 59, 59, 0, loc)) {
		t.Fatalf("due_date = %v, want August 31 2026 WIB", bill.DueDate)
	}
	if bill.Source != "auto_renewal" {
		t.Fatalf("source = %q, want auto_renewal", bill.Source)
	}
}
