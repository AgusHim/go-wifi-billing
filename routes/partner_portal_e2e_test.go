package routes

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Agushim/go_wifi_billing/controllers"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestPartnerLoginAccountCanReachPortalDashboard is an end-to-end check of the
// B2B login flow: staff creates a partner together with a login account, the
// partner signs in with those credentials, reads only its own dashboard, and is
// refused by staff APIs.
func TestPartnerLoginAccountCanReachPortalDashboard(t *testing.T) {
	// The JWT signer is initialised from the environment at package init, so the
	// middleware must read the same value. Only fall back when nothing is set.
	if os.Getenv("JWT_SECRET") == "" {
		t.Setenv("JWT_SECRET", "test-jwt-secret")
	}
	t.Setenv("AUTHZ_ENFORCEMENT_MODE", "enforce")

	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&models.Role{}, &models.Permission{}, &models.RolePermission{}, &models.UserPermissionOverride{},
		&models.User{}, &models.Partner{}, &models.PartnerMember{}, &models.PartnerBandwidthService{},
		&models.PartnerBandwidthUsage{}, &models.PartnerInventoryAllocation{}, &models.PartnerInvoice{},
		&models.PartnerInvoiceLine{}, &models.PartnerInvoiceStatusHistory{}, &models.PartnerPayment{},
		&models.PartnerInvoiceNotification{}, &models.LegalDocument{},
	); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	adminRole := &models.Role{Key: "admin", Name: "Admin", IsActive: true}
	customerRole := &models.Role{Key: "customer", Name: "Customer", IsActive: true}
	for _, role := range []*models.Role{adminRole, customerRole} {
		if err := db.Create(role).Error; err != nil {
			t.Fatalf("seed role %s: %v", role.Key, err)
		}
	}
	grant := func(role *models.Role, keys ...string) {
		for _, key := range keys {
			permission := &models.Permission{Key: key, Module: "partners", Action: "read", Name: key}
			if err := db.Create(permission).Error; err != nil {
				t.Fatalf("seed permission %s: %v", key, err)
			}
			if err := db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: permission.ID}).Error; err != nil {
				t.Fatalf("grant %s: %v", key, err)
			}
		}
	}
	grant(adminRole, "partners.read", "partners.manage", "partners.invoices.read", "partners.invoices.manage", "partners.documents.read", "partners.documents.manage", "partners.payments.verify")
	grant(customerRole, "self.partner.read", "self.partner.payments.create")

	staffPassword := "staff-password"
	hashed, err := bcrypt.GenerateFromPassword([]byte(staffPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash staff password: %v", err)
	}
	staff := &models.User{Name: "Staf B2B", Email: "staff@cantika.net", Password: string(hashed), Role: "admin", RoleID: &adminRole.ID, IsActive: true}
	if err := db.Create(staff).Error; err != nil {
		t.Fatalf("seed staff user: %v", err)
	}

	userRepo := repositories.NewUserRepository(db)
	authorizationSvc := services.NewAuthorizationService(repositories.NewAuthorizationRepository(db))
	userCtrl := controllers.NewUserController(services.NewUserService(userRepo), authorizationSvc)
	partnerSvc := services.NewPartnerService(
		repositories.NewPartnerRepository(db), repositories.NewInventoryRepository(db), userRepo, db, nil,
	)
	partnerCtrl := controllers.NewPartnerController(partnerSvc)

	app := fiber.New()
	Setup(
		app,
		controllers.NewCoverageController(nil), userCtrl, controllers.NewPackageController(nil),
		controllers.NewOdcController(nil), controllers.NewOdpController(nil), controllers.NewCustomerController(nil),
		controllers.NewSubscriptionController(nil, nil, nil), controllers.NewBillController(nil), controllers.NewComplainController(nil),
		controllers.NewPaymentController(nil), controllers.NewRouterController(nil), controllers.NewNOCController(nil, nil, nil),
		controllers.NewProvisioningController(nil), controllers.NewNetworkPlanController(nil), controllers.NewServiceAccountController(nil),
		controllers.NewRenewalController(nil), controllers.NewVoucherController(nil), controllers.NewWhatsAppController(nil),
		controllers.NewWhatsAppTemplateController(nil), controllers.NewExpenseController(nil), controllers.NewFinanceController(nil),
		controllers.NewSettingController(nil), controllers.NewInventoryController(nil), controllers.NewAccessControlController(nil, nil),
		authorizationSvc, partnerCtrl,
	)

	post := func(path, token string, payload any) (*http.Response, map[string]any) {
		t.Helper()
		body, _ := json.Marshal(payload)
		request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("request %s: %v", path, err)
		}
		parsed := map[string]any{}
		_ = json.NewDecoder(response.Body).Decode(&parsed)
		return response, parsed
	}
	get := func(path, token string) (*http.Response, map[string]any) {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("request %s: %v", path, err)
		}
		parsed := map[string]any{}
		_ = json.NewDecoder(response.Body).Decode(&parsed)
		return response, parsed
	}
	// postMultipart mirrors the portal upload form: text fields plus an optional
	// proof file in the same request.
	postMultipart := func(path, token string, fields map[string]string) (*http.Response, map[string]any) {
		t.Helper()
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		for key, value := range fields {
			if err := writer.WriteField(key, value); err != nil {
				t.Fatalf("write field %s: %v", key, err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close multipart writer: %v", err)
		}
		request := httptest.NewRequest(http.MethodPost, path, body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("request %s: %v", path, err)
		}
		parsed := map[string]any{}
		_ = json.NewDecoder(response.Body).Decode(&parsed)
		return response, parsed
	}

	// 1. Staff logs in.
	response, payload := post("/api/auth/login", "", map[string]string{"email": "staff@cantika.net", "password": staffPassword})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("staff login status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	staffToken := payload["data"].(map[string]any)["token"].(string)

	// 2. Staff creates a partner together with a login account.
	response, payload = post("/admin_api/partners", staffToken, map[string]any{
		"code": "MITRA-E2E", "name": "PT Mitra E2E", "billing_email": "billing@mitra-e2e.id",
		"account": map[string]any{"name": "Budi Mitra", "email": "budi@mitra-e2e.id", "password": "partner-pass-1"},
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("create partner status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	created := payload["data"].(map[string]any)
	partnerID := created["partner"].(map[string]any)["id"].(string)
	account := created["account"].(map[string]any)
	if account["role"] != "owner" {
		t.Fatalf("account role = %v, want owner", account["role"])
	}
	if account["temporary_password"] != nil {
		t.Fatalf("explicit password must not be echoed: %v", account["temporary_password"])
	}

	// 3. The partner logs in with the created credentials.
	response, payload = post("/api/auth/login", "", map[string]string{"email": "budi@mitra-e2e.id", "password": "partner-pass-1"})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("partner login status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	partnerToken := payload["data"].(map[string]any)["token"].(string)

	// 4. The partner reaches its own dashboard.
	response, payload = get("/partner_api/me", partnerToken)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("partner dashboard status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	profile := payload["data"].(map[string]any)
	if profile["id"] != partnerID {
		t.Fatalf("portal profile id = %v, want %s", profile["id"], partnerID)
	}

	// 5. The partner cannot reach staff APIs.
	response, _ = get("/admin_api/partners", partnerToken)
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("partner on staff API status = %d, want 403", response.StatusCode)
	}

	// 6. Staff can add another login account later, and it can sign in too.
	response, payload = post("/admin_api/partners/"+partnerID+"/accounts", staffToken, map[string]any{
		"name": "Sari Mitra", "email": "sari@mitra-e2e.id",
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("add account status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	tempPassword, _ := payload["data"].(map[string]any)["temporary_password"].(string)
	if len(tempPassword) < 6 {
		t.Fatalf("expected a generated temporary password, got %q", tempPassword)
	}
	response, payload = post("/api/auth/login", "", map[string]string{"email": "sari@mitra-e2e.id", "password": tempPassword})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("second account login status = %d, want 200 (%v)", response.StatusCode, payload)
	}

	// 7. Staff can add a bandwidth service and then build a manual invoice for it.
	response, payload = post("/admin_api/partners/"+partnerID+"/services", staffToken, map[string]any{
		"name": "Dedicated 100Mbps", "capacity": "100Mbps", "billing_mode": "fixed", "rate": 3000000, "due_day": 10,
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("create service status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	serviceID := payload["data"].(map[string]any)["id"].(string)

	response, payload = post("/admin_api/partners/invoices", staffToken, map[string]any{
		"partner_id": partnerID,
		"due_date":   time.Now().AddDate(0, 0, 10).Format(time.RFC3339),
		"lines": []map[string]any{
			{
				"description": "Dedicated 100Mbps - November 2026", "line_type": "bandwidth",
				"quantity": 1, "unit": "bulan", "unit_price": 3000000,
				"service_id": serviceID, "period_year": 2026, "period_month": 11,
			},
			{
				"description": "Biaya instalasi", "line_type": "other",
				"quantity": 1, "unit": "kali", "unit_price": 250000, "tax_amount": 27500,
			},
		},
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("create invoice status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	createdInvoice := payload["data"].(map[string]any)
	if createdInvoice["status"] != "draft" {
		t.Fatalf("new invoice status = %v, want draft", createdInvoice["status"])
	}
	if createdInvoice["total"].(float64) != 3277500 {
		t.Fatalf("new invoice total = %v, want 3277500", createdInvoice["total"])
	}
	if !strings.HasPrefix(createdInvoice["number"].(string), "DRAFT-") {
		t.Fatalf("draft number = %v, want a DRAFT- prefix", createdInvoice["number"])
	}

	// 8. The new draft shows up in the partner invoice list for staff.
	response, payload = get("/admin_api/partners/invoices?partner_id="+partnerID, staffToken)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("list invoices status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	if len(payload["data"].([]any)) != 1 {
		t.Fatalf("partner invoice count = %d, want 1", len(payload["data"].([]any)))
	}

	// 9. Legal documents are stored per partner, with global documents shared.
	response, payload = post("/admin_api/partners/legal-documents", staffToken, map[string]any{
		"name": "NIB Cantika", "file_path": "legal/nib.pdf", "status": "published", "is_partner_visible": true,
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("create global document status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	response, payload = post("/admin_api/partners/legal-documents", staffToken, map[string]any{
		"name": "Perjanjian Mitra E2E", "file_path": "legal/mitra-e2e.pdf", "status": "published",
		"is_partner_visible": true, "partner_id": partnerID,
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("create partner document status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	if payload["data"].(map[string]any)["partner_id"] != partnerID {
		t.Fatalf("document partner_id = %v, want %s", payload["data"].(map[string]any)["partner_id"], partnerID)
	}

	// 10. The archive can be filtered per partner and per global scope.
	response, payload = get("/admin_api/partners/legal-documents", staffToken)
	if response.StatusCode != http.StatusOK || len(payload["data"].([]any)) != 2 {
		t.Fatalf("unfiltered archive = %d (%v), want 2", len(payload["data"].([]any)), payload)
	}
	response, payload = get("/admin_api/partners/legal-documents?partner_id="+partnerID, staffToken)
	if response.StatusCode != http.StatusOK || len(payload["data"].([]any)) != 1 {
		t.Fatalf("partner archive = %d (%v), want 1", len(payload["data"].([]any)), payload)
	}
	response, payload = get("/admin_api/partners/legal-documents?partner_id=global", staffToken)
	if response.StatusCode != http.StatusOK || len(payload["data"].([]any)) != 1 {
		t.Fatalf("global archive = %d (%v), want 1", len(payload["data"].([]any)), payload)
	}

	// 11. The portal sees the partner document plus the global one, and nothing
	// belonging to another partner.
	response, payload = get("/partner_api/documents", partnerToken)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("portal documents status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	if len(payload["data"].([]any)) != 2 {
		t.Fatalf("portal document count = %d, want 2", len(payload["data"].([]any)))
	}

	// 12. Issuing the draft turns it into a payable (hutang) for the partner.
	invoiceID := createdInvoice["id"].(string)
	response, payload = post("/admin_api/partners/invoices/"+invoiceID+"/issue", staffToken, nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("issue invoice status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	response, payload = get("/admin_api/partners/invoices?unpaid=true", staffToken)
	if response.StatusCode != http.StatusOK || len(payload["data"].([]any)) != 1 {
		t.Fatalf("payables list = %d (%v), want 1", len(payload["data"].([]any)), payload)
	}

	// 13. The partner submits a payment with a transfer reference using a
	// multipart form (the same request shape used when a proof file is attached).
	response, payload = postMultipart("/partner_api/invoices/"+invoiceID+"/payments", partnerToken, map[string]string{
		"amount": "1000000", "method": "transfer", "reference": "BCA-9001", "idempotency_key": "portal-e2e-1",
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("portal multipart payment status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	submitted := payload["data"].(map[string]any)
	if submitted["status"] != "pending" || submitted["reference"] != "BCA-9001" {
		t.Fatalf("portal payment = %+v, want a pending transfer reference", submitted)
	}
	// A pending submission must not change the balance.
	response, payload = get("/admin_api/partners/invoices/"+invoiceID, staffToken)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("get invoice status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	if payload["data"].(map[string]any)["balance_due"].(float64) != 3277500 {
		t.Fatalf("balance after pending payment = %v, want 3277500", payload["data"].(map[string]any)["balance_due"])
	}

	// 14. Staff verifies the partner submission.
	paymentID := submitted["id"].(string)
	response, payload = post("/admin_api/partners/payments/"+paymentID+"/review", staffToken, map[string]any{"approve": true})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("verify payment status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	if payload["data"].(map[string]any)["status"] != "verified" {
		t.Fatalf("verified payment = %+v", payload["data"])
	}

	// 15. Staff settles the rest directly on behalf of the partner.
	response, payload = post("/admin_api/partners/"+partnerID+"/payments", staffToken, map[string]any{
		"invoice_id": invoiceID, "amount": 2277500, "method": "cash", "reference": "KAS-001",
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("record staff payment status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	recorded := payload["data"].(map[string]any)
	if recorded["status"] != "verified" {
		t.Fatalf("staff payment status = %v, want verified", recorded["status"])
	}
	response, payload = get("/admin_api/partners/invoices/"+invoiceID, staffToken)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("get settled invoice status = %d, want 200 (%v)", response.StatusCode, payload)
	}
	settled := payload["data"].(map[string]any)
	if settled["status"] != "paid" || settled["balance_due"].(float64) != 0 {
		t.Fatalf("settled invoice = %v/%v, want paid/0", settled["status"], settled["balance_due"])
	}

	// 16. The settled invoice leaves the payable list.
	response, payload = get("/admin_api/partners/invoices?unpaid=true", staffToken)
	if response.StatusCode != http.StatusOK || len(payload["data"].([]any)) != 0 {
		t.Fatalf("payables after settlement = %d (%v), want 0", len(payload["data"].([]any)), payload)
	}
}
