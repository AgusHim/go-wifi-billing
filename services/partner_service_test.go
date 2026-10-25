package services

import (
	"testing"
	"time"

	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newPartnerTestService(t *testing.T) (PartnerService, *gorm.DB) {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&models.Partner{}, &models.PartnerMember{}, &models.PartnerBandwidthService{},
		&models.PartnerBandwidthUsage{}, &models.PartnerInventoryAllocation{},
		&models.PartnerInvoice{}, &models.PartnerInvoiceLine{}, &models.PartnerInvoiceStatusHistory{},
		&models.PartnerPayment{}, &models.PartnerInvoiceNotification{}, &models.LegalDocument{},
		&models.InventoryItem{}, &models.InventoryLocation{}, &models.InventoryStock{},
		&models.InventoryStockMovement{}, &models.InventorySerialItem{}, &models.User{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	svc := NewPartnerService(repositories.NewPartnerRepository(db), repositories.NewInventoryRepository(db), db, nil)
	return svc, db
}

func createTestPartner(t *testing.T, svc PartnerService, code string) *models.Partner {
	t.Helper()
	partner, err := svc.CreatePartner(dto.PartnerRequest{Code: code, Name: "Mitra " + code, BillingEmail: code + "@example.com"})
	if err != nil {
		t.Fatalf("create partner %s: %v", code, err)
	}
	return partner
}

func createTestService(t *testing.T, svc PartnerService, partnerID uuid.UUID, mode string, rate int) *models.PartnerBandwidthService {
	t.Helper()
	row, err := svc.CreateBandwidthService(partnerID, dto.PartnerBandwidthServiceRequest{Name: "Dedicated " + mode, Capacity: "100Mbps", BillingMode: mode, Rate: rate})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	return row
}

// TestPartnerBillingModesAndPeriodDeduplication covers the MVP rule that all
// three billing modes produce invoice lines and the same period cannot be
// billed twice.
func TestPartnerBillingModesAndPeriodDeduplication(t *testing.T) {
	svc, _ := newPartnerTestService(t)
	partner := createTestPartner(t, svc, "MITRA1")
	actor := uuid.New()

	fixed := createTestService(t, svc, partner.ID, "fixed", 5000000)
	invoice, err := svc.GenerateInvoiceForService(partner.ID, dto.PartnerInvoiceFromServiceRequest{ServiceID: fixed.ID.String(), PeriodYear: 2026, PeriodMonth: 7}, actor)
	if err != nil {
		t.Fatalf("generate fixed invoice: %v", err)
	}
	if invoice.Total != 5000000 {
		t.Fatalf("fixed invoice total = %d, want 5000000", invoice.Total)
	}
	if _, err := svc.GenerateInvoiceForService(partner.ID, dto.PartnerInvoiceFromServiceRequest{ServiceID: fixed.ID.String(), PeriodYear: 2026, PeriodMonth: 7}, actor); err == nil {
		t.Fatal("expected duplicate period billing to be rejected")
	}

	usage := createTestService(t, svc, partner.ID, "usage", 10000)
	if _, err := svc.GenerateInvoiceForService(partner.ID, dto.PartnerInvoiceFromServiceRequest{ServiceID: usage.ID.String(), PeriodYear: 2026, PeriodMonth: 7}, actor); err == nil {
		t.Fatal("expected usage billing without an approved reading to fail")
	}
	reading, err := svc.RecordUsage(partner.ID, dto.PartnerBandwidthUsageRequest{ServiceID: usage.ID.String(), PeriodYear: 2026, PeriodMonth: 7, Quantity: 250, Unit: "Mbps"}, actor)
	if err != nil {
		t.Fatalf("record usage: %v", err)
	}
	if _, err := svc.GenerateInvoiceForService(partner.ID, dto.PartnerInvoiceFromServiceRequest{ServiceID: usage.ID.String(), PeriodYear: 2026, PeriodMonth: 7}, actor); err == nil {
		t.Fatal("expected pending usage to be unbillable")
	}
	if _, err := svc.ReviewUsage(reading.ID, dto.PartnerBandwidthUsageApprovalRequest{Approve: true}, actor); err != nil {
		t.Fatalf("approve usage: %v", err)
	}
	usageInvoice, err := svc.GenerateInvoiceForService(partner.ID, dto.PartnerInvoiceFromServiceRequest{ServiceID: usage.ID.String(), PeriodYear: 2026, PeriodMonth: 7}, actor)
	if err != nil {
		t.Fatalf("generate usage invoice: %v", err)
	}
	if usageInvoice.Total != 2500000 {
		t.Fatalf("usage invoice total = %d, want 2500000", usageInvoice.Total)
	}

	manual := createTestService(t, svc, partner.ID, "manual", 0)
	manualInvoice, err := svc.GenerateInvoiceForService(partner.ID, dto.PartnerInvoiceFromServiceRequest{ServiceID: manual.ID.String(), PeriodYear: 2026, PeriodMonth: 7, Amount: 1750000, TaxAmount: 250000}, actor)
	if err != nil {
		t.Fatalf("generate manual invoice: %v", err)
	}
	if manualInvoice.Total != 2000000 || manualInvoice.TaxAmount != 250000 {
		t.Fatalf("manual invoice total = %d tax = %d, want 2000000/250000", manualInvoice.Total, manualInvoice.TaxAmount)
	}
}

// TestPartnerInventoryAllocationLoanAndSale covers stock movement, invoice
// linkage, and single-billing of a sold item.
func TestPartnerInventoryAllocationLoanAndSale(t *testing.T) {
	svc, db := newPartnerTestService(t)
	partner := createTestPartner(t, svc, "MITRA2")
	actor := uuid.New()

	item := models.InventoryItem{ID: uuid.New(), SKU: "ONT-X", Name: "ONT X", Unit: "pcs", IsActive: true}
	location := models.InventoryLocation{ID: uuid.New(), Name: "Gudang", Type: "warehouse", IsActive: true}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("create location: %v", err)
	}
	if err := db.Create(&models.InventoryStock{ID: uuid.New(), ItemID: item.ID, LocationID: location.ID, QuantityOnHand: 10, AverageCost: 400000}).Error; err != nil {
		t.Fatalf("create stock: %v", err)
	}

	// A loan leaves Cantika.net ownership and can be returned.
	loan, err := svc.CreateAllocation(partner.ID, dto.PartnerAllocationRequest{ItemID: item.ID.String(), Quantity: 2, FromLocationID: location.ID.String(), TransactionType: "loan"}, actor)
	if err != nil {
		t.Fatalf("create loan allocation: %v", err)
	}
	assertStock(t, db, item.ID, location.ID, 8)

	// A loan must never be billable.
	if _, err := svc.CreateInvoice(dto.PartnerInvoiceRequest{PartnerID: partner.ID.String(), DueDate: time.Now(), Lines: []dto.PartnerInvoiceLineRequest{{Description: "Loan", LineType: "inventory", Quantity: 2, UnitPrice: 100, AllocationID: loan.ID.String()}}}, actor); err == nil {
		t.Fatal("expected loan allocation to be rejected for billing")
	}

	if _, err := svc.ReturnAllocation(loan.ID, dto.PartnerAllocationReturnRequest{ToLocationID: location.ID.String(), Notes: "dikembalikan"}, actor); err != nil {
		t.Fatalf("return loan: %v", err)
	}
	assertStock(t, db, item.ID, location.ID, 10)

	// A sale reduces stock once and must be linked to an invoice line.
	sale, err := svc.CreateAllocation(partner.ID, dto.PartnerAllocationRequest{ItemID: item.ID.String(), Quantity: 3, UnitPrice: 550000, FromLocationID: location.ID.String(), TransactionType: "sale"}, actor)
	if err != nil {
		t.Fatalf("create sale allocation: %v", err)
	}
	assertStock(t, db, item.ID, location.ID, 7)

	invoice, err := svc.CreateInvoice(dto.PartnerInvoiceRequest{PartnerID: partner.ID.String(), DueDate: time.Now(), Lines: []dto.PartnerInvoiceLineRequest{{Description: "Penjualan ONT", LineType: "inventory", Quantity: 3, UnitPrice: 550000, AllocationID: sale.ID.String()}}}, actor)
	if err != nil {
		t.Fatalf("create inventory invoice: %v", err)
	}
	if invoice.Total != 1650000 {
		t.Fatalf("inventory invoice total = %d, want 1650000", invoice.Total)
	}
	stored, err := svc.ListAllocations(&partner.ID, "", "sale")
	if err != nil || len(stored) != 1 {
		t.Fatalf("list allocations = %v (%d), want 1", err, len(stored))
	}
	if stored[0].InvoiceLineID == nil {
		t.Fatal("sale allocation was not linked to an invoice line")
	}
	if _, err := svc.CreateInvoice(dto.PartnerInvoiceRequest{PartnerID: partner.ID.String(), DueDate: time.Now(), Lines: []dto.PartnerInvoiceLineRequest{{Description: "Penjualan ONT lagi", LineType: "inventory", Quantity: 3, UnitPrice: 550000, AllocationID: sale.ID.String()}}}, actor); err == nil {
		t.Fatal("expected an already billed allocation to be rejected")
	}
	// Stock must not have moved again on the rejected attempt.
	assertStock(t, db, item.ID, location.ID, 7)
}

func assertStock(t *testing.T, db *gorm.DB, itemID, locationID uuid.UUID, want float64) {
	t.Helper()
	var stock models.InventoryStock
	if err := db.Where("item_id = ? AND location_id = ?", itemID, locationID).First(&stock).Error; err != nil {
		t.Fatalf("find stock: %v", err)
	}
	if stock.QuantityOnHand != want {
		t.Fatalf("stock on hand = %v, want %v", stock.QuantityOnHand, want)
	}
}

// TestPartnerPaymentVerificationAndIdempotency covers the MVP rule that only a
// verified payment reduces the balance, partial payments update status, and a
// repeated request does not double-apply.
func TestPartnerPaymentVerificationAndIdempotency(t *testing.T) {
	svc, _ := newPartnerTestService(t)
	partner := createTestPartner(t, svc, "MITRA3")
	actor := uuid.New()

	service := createTestService(t, svc, partner.ID, "fixed", 1000000)
	invoice, err := svc.GenerateInvoiceForService(partner.ID, dto.PartnerInvoiceFromServiceRequest{ServiceID: service.ID.String(), PeriodYear: 2026, PeriodMonth: 8}, actor)
	if err != nil {
		t.Fatalf("generate invoice: %v", err)
	}
	if _, err := svc.IssueInvoice(invoice.ID, actor); err != nil {
		t.Fatalf("issue invoice: %v", err)
	}

	// A pending submission must not change the balance.
	pending, err := svc.SubmitPayment(invoice.ID, dto.PartnerPaymentRequest{Amount: 400000, Method: "transfer", IdempotencyKey: "key-1"}, actor)
	if err != nil {
		t.Fatalf("submit payment: %v", err)
	}
	// Retrying with the same key must return the same record, not a new one.
	again, err := svc.SubmitPayment(invoice.ID, dto.PartnerPaymentRequest{Amount: 400000, Method: "transfer", IdempotencyKey: "key-1"}, actor)
	if err != nil {
		t.Fatalf("resubmit payment: %v", err)
	}
	if again.ID != pending.ID {
		t.Fatalf("idempotent resubmit created a new payment %s != %s", again.ID, pending.ID)
	}
	assertInvoiceState(t, svc, invoice.ID, "issued", 1000000, 0)

	// A rejected payment must not change the balance either.
	rejected, err := svc.SubmitPayment(invoice.ID, dto.PartnerPaymentRequest{Amount: 100000, Method: "transfer", IdempotencyKey: "key-2"}, actor)
	if err != nil {
		t.Fatalf("submit rejected candidate: %v", err)
	}
	if _, err := svc.VerifyPayment(rejected.ID, dto.PartnerPaymentVerificationRequest{Approve: false, RejectionReason: "bukti tidak jelas"}, actor); err != nil {
		t.Fatalf("reject payment: %v", err)
	}
	assertInvoiceState(t, svc, invoice.ID, "issued", 1000000, 0)

	// A partial verification moves the invoice to partially_paid.
	if _, err := svc.VerifyPayment(pending.ID, dto.PartnerPaymentVerificationRequest{Approve: true}, actor); err != nil {
		t.Fatalf("verify partial payment: %v", err)
	}
	assertInvoiceState(t, svc, invoice.ID, "partially_paid", 600000, 400000)
	// Verifying the same payment twice must be rejected.
	if _, err := svc.VerifyPayment(pending.ID, dto.PartnerPaymentVerificationRequest{Approve: true}, actor); err == nil {
		t.Fatal("expected double verification to be rejected")
	}

	// Paying the remainder settles the invoice exactly once.
	settle, err := svc.SubmitPayment(invoice.ID, dto.PartnerPaymentRequest{Amount: 600000, Method: "transfer", IdempotencyKey: "key-3"}, actor)
	if err != nil {
		t.Fatalf("submit settlement: %v", err)
	}
	if _, err := svc.VerifyPayment(settle.ID, dto.PartnerPaymentVerificationRequest{Approve: true}, actor); err != nil {
		t.Fatalf("verify settlement: %v", err)
	}
	assertInvoiceState(t, svc, invoice.ID, "paid", 0, 1000000)

	history, err := svc.ListInvoiceHistory(invoice.ID)
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(history) < 4 {
		t.Fatalf("status history length = %d, want at least 4 (created, issued, partial, paid)", len(history))
	}
}

func assertInvoiceState(t *testing.T, svc PartnerService, id uuid.UUID, status string, balance, paid int) {
	t.Helper()
	invoice, err := svc.GetInvoice(id, nil)
	if err != nil {
		t.Fatalf("get invoice: %v", err)
	}
	if invoice.Status != status || invoice.BalanceDue != balance || invoice.VerifiedPaidAmount != paid {
		t.Fatalf("invoice = (%s, %d, %d), want (%s, %d, %d)", invoice.Status, invoice.BalanceDue, invoice.VerifiedPaidAmount, status, balance, paid)
	}
}

// TestPartnerCrossAccessIsBlocked verifies one partner cannot read another
// partner's invoice, and that legal documents are scoped for portal users.
func TestPartnerCrossAccessIsBlocked(t *testing.T) {
	svc, _ := newPartnerTestService(t)
	first := createTestPartner(t, svc, "MITRA4")
	second := createTestPartner(t, svc, "MITRA5")
	actor := uuid.New()

	service := createTestService(t, svc, first.ID, "fixed", 750000)
	invoice, err := svc.GenerateInvoiceForService(first.ID, dto.PartnerInvoiceFromServiceRequest{ServiceID: service.ID.String(), PeriodYear: 2026, PeriodMonth: 9}, actor)
	if err != nil {
		t.Fatalf("generate invoice: %v", err)
	}
	if _, err := svc.GetInvoice(invoice.ID, &second.ID); err == nil {
		t.Fatal("expected cross-partner invoice access to be blocked")
	}
	if _, err := svc.GetInvoice(invoice.ID, &first.ID); err != nil {
		t.Fatalf("owner should read own invoice: %v", err)
	}

	private, err := svc.CreateLegalDocument(dto.LegalDocumentRequest{Name: "NIB", FilePath: "private/nib.pdf", Status: "published", IsPartnerVisible: false}, actor)
	if err != nil {
		t.Fatalf("create private document: %v", err)
	}
	if _, err := svc.GetLegalDocument(private.ID, &first.ID); err == nil {
		t.Fatal("expected a non-shared document to be hidden from partners")
	}
	shared, err := svc.CreateLegalDocument(dto.LegalDocumentRequest{Name: "SLA", FilePath: "shared/sla.pdf", Status: "published", IsPartnerVisible: true}, actor)
	if err != nil {
		t.Fatalf("create shared document: %v", err)
	}
	if _, err := svc.GetLegalDocument(shared.ID, &first.ID); err != nil {
		t.Fatalf("partner should read a shared document: %v", err)
	}
	if _, err := svc.GetLegalDocument(private.ID, nil); err != nil {
		t.Fatalf("staff should read any document: %v", err)
	}
}

// TestPartnerAgingReportBuckets verifies the receivables aging report totals.
func TestPartnerAgingReportBuckets(t *testing.T) {
	svc, db := newPartnerTestService(t)
	partner := createTestPartner(t, svc, "MITRA6")
	now := time.Now()

	create := func(dueDate time.Time, total int) {
		invoice := models.PartnerInvoice{ID: uuid.New(), Number: "AG-" + uuid.NewString()[:8], PartnerID: partner.ID, DueDate: dueDate, Status: "issued", Total: total, BalanceDue: total, PartnerNameSnapshot: partner.Name}
		if err := db.Create(&invoice).Error; err != nil {
			t.Fatalf("create aging invoice: %v", err)
		}
	}
	create(now.AddDate(0, 0, 10), 100000)  // current
	create(now.AddDate(0, 0, -10), 200000) // 1-30 days
	create(now.AddDate(0, 0, -45), 300000) // 31-60 days
	create(now.AddDate(0, 0, -100), 400000)

	rows, err := svc.AgingReport(nil)
	if err != nil {
		t.Fatalf("aging report: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("aging rows = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.Current != 100000 || row.Days1To30 != 200000 || row.Days31To60 != 300000 || row.Over90 != 400000 {
		t.Fatalf("aging buckets = %+v", row)
	}
	if row.TotalDue != 1000000 || row.InvoiceCount != 4 {
		t.Fatalf("aging totals = %d/%d, want 1000000/4", row.TotalDue, row.InvoiceCount)
	}
}
