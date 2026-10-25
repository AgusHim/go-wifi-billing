package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
	grant(adminRole, "partners.read", "partners.manage")
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
}
