package services

import (
	"testing"
	"time"

	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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
		&models.InventoryStockMovement{}, &models.InventorySerialItem{},
		&models.User{}, &models.Role{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	// The portal login account uses the canonical `customer` role.
	if err := db.Create(&models.Role{ID: uuid.New(), Key: "customer", Name: "Customer", IsActive: true}).Error; err != nil {
		t.Fatalf("seed customer role: %v", err)
	}
	svc := NewPartnerService(repositories.NewPartnerRepository(db), repositories.NewInventoryRepository(db), repositories.NewUserRepository(db), db, nil)
	return svc, db
}

func createTestPartner(t *testing.T, svc PartnerService, code string) *models.Partner {
	t.Helper()
	result, err := svc.CreatePartner(dto.PartnerRequest{Code: code, Name: "Mitra " + code, BillingEmail: code + "@example.com"})
	if err != nil {
		t.Fatalf("create partner %s: %v", code, err)
	}
	return result.Partner
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

// TestPartnerLoginAccountCreation verifies that creating a partner can also
// create a working portal login account, and that the account is the one the
// portal resolves for that user.
func TestPartnerLoginAccountCreation(t *testing.T) {
	svc, db := newPartnerTestService(t)

	result, err := svc.CreatePartner(dto.PartnerRequest{
		Code: "MITRA7", Name: "PT Mitra Tujuh", BillingEmail: "billing@mitra7.id",
		Account: &dto.PartnerAccountRequest{Name: "Budi", Email: "budi@mitra7.id", Phone: "08120000000", Password: "rahasia123"},
	})
	if err != nil {
		t.Fatalf("create partner with account: %v", err)
	}
	if result.Account == nil {
		t.Fatal("expected a login account to be created")
	}
	if result.Account.Role != "owner" {
		t.Fatalf("first account role = %q, want owner", result.Account.Role)
	}
	if result.Account.TemporaryPassword != "" {
		t.Fatal("an explicit password must not be echoed back")
	}

	// The user must exist with the customer role and a verifiable password hash.
	var user models.User
	if err := db.Where("email = ?", "budi@mitra7.id").First(&user).Error; err != nil {
		t.Fatalf("find portal user: %v", err)
	}
	if user.Role != "user" {
		t.Fatalf("legacy role = %q, want user", user.Role)
	}
	if user.RoleID == nil {
		t.Fatal("portal user has no canonical role assigned")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("rahasia123")) != nil {
		t.Fatal("stored password hash does not match the requested password")
	}

	// The portal resolves the partner from the member link.
	partner, err := svc.PartnerForUser(user.ID)
	if err != nil {
		t.Fatalf("resolve partner for portal user: %v", err)
	}
	if partner.ID != result.Partner.ID {
		t.Fatalf("portal partner = %s, want %s", partner.ID, result.Partner.ID)
	}

	accounts, err := svc.ListPartnerAccounts(result.Partner.ID)
	if err != nil || len(accounts) != 1 {
		t.Fatalf("list accounts = %v (%d), want 1", err, len(accounts))
	}
	if accounts[0].Email != "budi@mitra7.id" || accounts[0].TemporaryPassword != "" {
		t.Fatalf("listed account = %+v", accounts[0])
	}

	// A second account is added as a regular member and gets a temporary password.
	second, err := svc.CreatePartnerAccount(result.Partner.ID, dto.PartnerAccountRequest{Name: "Sari", Email: "sari@mitra7.id"}, uuid.New())
	if err != nil {
		t.Fatalf("create second account: %v", err)
	}
	if second.Role != "member" {
		t.Fatalf("second account role = %q, want member", second.Role)
	}
	if len(second.TemporaryPassword) < 6 {
		t.Fatalf("expected a generated temporary password, got %q", second.TemporaryPassword)
	}
	var secondUser models.User
	if err := db.Where("email = ?", "sari@mitra7.id").First(&secondUser).Error; err != nil {
		t.Fatalf("find second portal user: %v", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(secondUser.Password), []byte(second.TemporaryPassword)) != nil {
		t.Fatal("generated temporary password does not unlock the account")
	}

	// Duplicate access for the same email must be rejected.
	if _, err := svc.CreatePartnerAccount(result.Partner.ID, dto.PartnerAccountRequest{Name: "Budi", Email: "budi@mitra7.id"}, uuid.New()); err == nil {
		t.Fatal("expected duplicate partner access to be rejected")
	}

	// The cross-partner listing must carry the owning partner for each account.
	all, err := svc.ListAllPartnerAccounts()
	if err != nil {
		t.Fatalf("list all partner accounts: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("cross-partner account count = %d, want 2", len(all))
	}
	for _, account := range all {
		if account.PartnerID != result.Partner.ID || account.PartnerName == "" || account.PartnerCode != "MITRA7" {
			t.Fatalf("cross-partner account missing partner info: %+v", account)
		}
		if account.TemporaryPassword != "" {
			t.Fatal("listing must never expose a password")
		}
	}
}

// TestPartnerAccountValidationRejectsBadInput verifies account validation and
// that a rejected account does not leave a partner behind.
func TestPartnerAccountValidationRejectsBadInput(t *testing.T) {
	svc, db := newPartnerTestService(t)

	if _, err := svc.CreatePartner(dto.PartnerRequest{
		Code: "MITRA8", Name: "PT Mitra Delapan", BillingEmail: "billing@mitra8.id",
		Account: &dto.PartnerAccountRequest{Name: "Tanpa Email", Email: "bukan-email"},
	}); err == nil {
		t.Fatal("expected an invalid account email to be rejected")
	}
	var count int64
	if err := db.Model(&models.Partner{}).Where("code = ?", "MITRA8").Count(&count).Error; err != nil {
		t.Fatalf("count partners: %v", err)
	}
	if count != 0 {
		t.Fatalf("rejected account left %d partner rows behind", count)
	}

	if _, err := svc.CreatePartner(dto.PartnerRequest{
		Code: "MITRA9", Name: "PT Mitra Sembilan", BillingEmail: "billing@mitra9.id",
		Account: &dto.PartnerAccountRequest{Name: "Budi", Email: "budi@mitra9.id", Password: "123"},
	}); err == nil {
		t.Fatal("expected a short password to be rejected")
	}

	partner := createTestPartner(t, svc, "MITRA10")
	if _, err := svc.CreatePartnerAccount(partner.ID, dto.PartnerAccountRequest{Email: "x@y.id"}, uuid.New()); err == nil {
		t.Fatal("expected a missing account name to be rejected")
	}

	// A duplicate partner code must be reported as an input problem.
	if _, err := svc.CreatePartner(dto.PartnerRequest{Code: "mitra10", Name: "Duplikat", BillingEmail: "dup@mitra10.id"}); err == nil {
		t.Fatal("expected a duplicate partner code to be rejected")
	}
}

// TestPartnerManualMixedInvoice covers the multi-line invoice created from the
// staff dashboard: a bandwidth line with an explicit period, a sold inventory
// line, and a free-form line in one invoice.
func TestPartnerManualMixedInvoice(t *testing.T) {
	svc, db := newPartnerTestService(t)
	partner := createTestPartner(t, svc, "MITRA11")
	actor := uuid.New()

	service := createTestService(t, svc, partner.ID, "fixed", 3000000)

	item := models.InventoryItem{ID: uuid.New(), SKU: "ROUTER-Y", Name: "Router Y", Unit: "unit", IsActive: true}
	location := models.InventoryLocation{ID: uuid.New(), Name: "Gudang", Type: "warehouse", IsActive: true}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create item: %v", err)
	}
	if err := db.Create(&location).Error; err != nil {
		t.Fatalf("create location: %v", err)
	}
	if err := db.Create(&models.InventoryStock{ID: uuid.New(), ItemID: item.ID, LocationID: location.ID, QuantityOnHand: 5, AverageCost: 500000}).Error; err != nil {
		t.Fatalf("create stock: %v", err)
	}
	sale, err := svc.CreateAllocation(partner.ID, dto.PartnerAllocationRequest{
		ItemID: item.ID.String(), Quantity: 2, UnitPrice: 750000,
		FromLocationID: location.ID.String(), TransactionType: "sale",
	}, actor)
	if err != nil {
		t.Fatalf("create sale allocation: %v", err)
	}

	year, month := 2026, 11
	invoice, err := svc.CreateInvoice(dto.PartnerInvoiceRequest{
		PartnerID: partner.ID.String(),
		DueDate:   time.Now().AddDate(0, 0, 10),
		Lines: []dto.PartnerInvoiceLineRequest{
			{Description: "Dedicated 100Mbps - November 2026", LineType: "bandwidth", Quantity: 1, Unit: "bulan", UnitPrice: 3000000, ServiceID: service.ID.String(), PeriodYear: &year, PeriodMonth: &month},
			{Description: "Penjualan Router Y", LineType: "inventory", Quantity: 2, Unit: "unit", UnitPrice: 750000, AllocationID: sale.ID.String()},
			{Description: "Biaya instalasi", LineType: "other", Quantity: 1, Unit: "kali", UnitPrice: 250000, TaxAmount: 27500},
		},
	}, actor)
	if err != nil {
		t.Fatalf("create mixed invoice: %v", err)
	}
	if invoice.Status != "draft" {
		t.Fatalf("status = %q, want draft", invoice.Status)
	}
	if len(invoice.Lines) != 3 {
		t.Fatalf("line count = %d, want 3", len(invoice.Lines))
	}
	if invoice.Subtotal != 4750000 {
		t.Fatalf("subtotal = %d, want 4750000", invoice.Subtotal)
	}
	if invoice.TaxAmount != 27500 || invoice.Total != 4777500 || invoice.BalanceDue != invoice.Total {
		t.Fatalf("tax/total/balance = %d/%d/%d", invoice.TaxAmount, invoice.Total, invoice.BalanceDue)
	}

	// The sold item must be linked to the invoice line so it cannot be billed twice.
	billed, err := svc.ListAllocations(&partner.ID, "", "sale")
	if err != nil || len(billed) != 1 {
		t.Fatalf("list sale allocations = %v (%d), want 1", err, len(billed))
	}
	if billed[0].InvoiceLineID == nil {
		t.Fatal("sold allocation was not linked to the invoice line")
	}

	// The same service period must be rejected on a second invoice.
	if _, err := svc.CreateInvoice(dto.PartnerInvoiceRequest{
		PartnerID: partner.ID.String(), DueDate: time.Now(),
		Lines: []dto.PartnerInvoiceLineRequest{{
			Description: "Dedicated 100Mbps - November 2026", LineType: "bandwidth", Quantity: 1, Unit: "bulan",
			UnitPrice: 3000000, ServiceID: service.ID.String(), PeriodYear: &year, PeriodMonth: &month,
		}},
	}, actor); err == nil {
		t.Fatal("expected the billed service period to be rejected")
	}

	// Issuing the draft records the audit trail.
	issued, err := svc.IssueInvoice(invoice.ID, actor)
	if err != nil {
		t.Fatalf("issue invoice: %v", err)
	}
	if issued.Status != "issued" {
		t.Fatalf("issued status = %q, want issued", issued.Status)
	}
	history, err := svc.ListInvoiceHistory(invoice.ID)
	if err != nil || len(history) < 2 {
		t.Fatalf("status history = %v (%d), want at least 2", err, len(history))
	}
}

// TestPartnerDocumentScoping covers the legal document archive: each partner
// owns its own documents, documents without a partner apply to everyone, and a
// partner can only read documents that are shared and belong to it.
func TestPartnerDocumentScoping(t *testing.T) {
	svc, _ := newPartnerTestService(t)
	first := createTestPartner(t, svc, "MITRA12")
	second := createTestPartner(t, svc, "MITRA13")
	actor := uuid.New()

	firstID := first.ID.String()
	secondID := second.ID.String()
	create := func(name string, partnerID *string, visible bool) *models.LegalDocument {
		t.Helper()
		doc, err := svc.CreateLegalDocument(dto.LegalDocumentRequest{
			Name: name, FilePath: "legal/" + name + ".pdf", Status: "published",
			IsPartnerVisible: visible, PartnerID: partnerID,
		}, actor)
		if err != nil {
			t.Fatalf("create document %s: %v", name, err)
		}
		return doc
	}

	global := create("NIB Cantika", nil, true)
	firstDoc := create("Perjanjian Mitra12", &firstID, true)
	secondDoc := create("Perjanjian Mitra13", &secondID, true)
	firstPrivate := create("Internal Mitra12", &firstID, false)

	if global.PartnerID != nil {
		t.Fatal("a document created without partner_id must be global")
	}
	if firstDoc.PartnerID == nil || *firstDoc.PartnerID != first.ID {
		t.Fatal("document was not assigned to the requested partner")
	}
	if firstDoc.PartnerName != first.Name || firstDoc.PartnerCode != "MITRA12" {
		t.Fatalf("document partner label = %q/%q", firstDoc.PartnerName, firstDoc.PartnerCode)
	}

	// Staff archive without a filter returns everything.
	all, err := svc.ListLegalDocuments(repositories.LegalDocumentFilter{})
	if err != nil {
		t.Fatalf("list all documents: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("unfiltered document count = %d, want 4", len(all))
	}

	// Filter by partner id returns only that partner's documents.
	mine, err := svc.ListLegalDocuments(repositories.LegalDocumentFilter{PartnerID: &first.ID})
	if err != nil {
		t.Fatalf("list partner documents: %v", err)
	}
	if len(mine) != 2 {
		t.Fatalf("partner document count = %d, want 2", len(mine))
	}
	for _, doc := range mine {
		if doc.PartnerID == nil || *doc.PartnerID != first.ID {
			t.Fatalf("partner filter returned a foreign document: %+v", doc)
		}
	}

	// Filter for the global scope returns only documents without a partner.
	globals, err := svc.ListLegalDocuments(repositories.LegalDocumentFilter{UnassignedOnly: true})
	if err != nil {
		t.Fatalf("list global documents: %v", err)
	}
	if len(globals) != 1 || globals[0].ID != global.ID {
		t.Fatalf("global document count = %d, want only the NIB", len(globals))
	}

	// Portal listing: own documents plus the ones shared with everyone.
	visible := true
	portal, err := svc.ListLegalDocuments(repositories.LegalDocumentFilter{
		PartnerVisible: &visible, PartnerID: &first.ID, IncludeUnassigned: true,
	})
	if err != nil {
		t.Fatalf("list portal documents: %v", err)
	}
	if len(portal) != 2 {
		t.Fatalf("portal document count = %d, want 2 (own + global)", len(portal))
	}
	for _, doc := range portal {
		if doc.ID == secondDoc.ID || doc.ID == firstPrivate.ID {
			t.Fatalf("portal leaked a foreign or private document: %s", doc.Name)
		}
	}

	// Cross-partner reads are blocked, shared documents are readable.
	if _, err := svc.GetLegalDocument(firstDoc.ID, &second.ID); err == nil {
		t.Fatal("expected a foreign partner document to be hidden")
	}
	if _, err := svc.GetLegalDocument(firstDoc.ID, &first.ID); err != nil {
		t.Fatalf("owner partner should read its own document: %v", err)
	}
	if _, err := svc.GetLegalDocument(global.ID, &second.ID); err != nil {
		t.Fatalf("a global document should be readable by any partner: %v", err)
	}
	if _, err := svc.GetLegalDocument(firstPrivate.ID, &first.ID); err == nil {
		t.Fatal("expected a document not shared with partners to be hidden")
	}
	if _, err := svc.GetLegalDocument(secondDoc.ID, nil); err != nil {
		t.Fatalf("staff should read any document: %v", err)
	}

	// An unknown partner id is rejected instead of silently stored.
	bogus := uuid.NewString()
	if _, err := svc.CreateLegalDocument(dto.LegalDocumentRequest{
		Name: "Salah mitra", FilePath: "legal/x.pdf", Status: "published", PartnerID: &bogus,
	}, actor); err == nil {
		t.Fatal("expected an unknown partner_id to be rejected")
	}

	// Updating with an empty partner_id turns a partner document back to global.
	cleared := ""
	updated, err := svc.UpdateLegalDocument(firstDoc.ID, dto.LegalDocumentRequest{PartnerID: &cleared})
	if err != nil {
		t.Fatalf("clear document partner: %v", err)
	}
	if updated.PartnerID != nil {
		t.Fatal("expected the document to become global")
	}
}
