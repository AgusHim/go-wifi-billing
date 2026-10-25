package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Agushim/go_wifi_billing/controllers"
	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/gofiber/fiber/v2"
)

// setupAppWithPartnerRoutes registers every controller plus the B2B partner
// controller so partner routes take part in the fail-closed checks.
func setupAppWithPartnerRoutes() *fiber.App {
	app := fiber.New()
	Setup(
		app,
		controllers.NewCoverageController(nil), controllers.NewUserController(nil), controllers.NewPackageController(nil),
		controllers.NewOdcController(nil), controllers.NewOdpController(nil), controllers.NewCustomerController(nil),
		controllers.NewSubscriptionController(nil, nil, nil), controllers.NewBillController(nil), controllers.NewComplainController(nil),
		controllers.NewPaymentController(nil), controllers.NewRouterController(nil), controllers.NewNOCController(nil, nil, nil),
		controllers.NewProvisioningController(nil), controllers.NewNetworkPlanController(nil), controllers.NewServiceAccountController(nil),
		controllers.NewRenewalController(nil), controllers.NewVoucherController(nil), controllers.NewWhatsAppController(nil),
		controllers.NewWhatsAppTemplateController(nil), controllers.NewExpenseController(nil), controllers.NewFinanceController(nil),
		controllers.NewSettingController(nil), controllers.NewInventoryController(nil), controllers.NewAccessControlController(nil, nil),
		nil, controllers.NewPartnerController(nil),
	)
	return app
}

func TestPartnerRoutesRejectAnonymousRequests(t *testing.T) {
	t.Setenv("JWT_SECRET", "partner-route-secret")
	app := setupAppWithPartnerRoutes()

	id := "00000000-0000-0000-0000-000000000001"
	testCases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin_api/partners"},
		{http.MethodPost, "/admin_api/partners"},
		{http.MethodGet, "/admin_api/partners/" + id},
		{http.MethodPut, "/admin_api/partners/" + id},
		{http.MethodGet, "/admin_api/partners/" + id + "/accounts"},
		{http.MethodPost, "/admin_api/partners/" + id + "/accounts"},
		{http.MethodGet, "/admin_api/partners/" + id + "/services"},
		{http.MethodGet, "/admin_api/partners/" + id + "/usages"},
		{http.MethodGet, "/admin_api/partners/" + id + "/allocations"},
		{http.MethodGet, "/admin_api/partners/invoices"},
		{http.MethodPost, "/admin_api/partners/invoices/from-service"},
		{http.MethodGet, "/admin_api/partners/invoices/" + id},
		{http.MethodPost, "/admin_api/partners/invoices/" + id + "/issue"},
		{http.MethodGet, "/admin_api/partners/aging"},
		{http.MethodGet, "/admin_api/partners/usages"},
		{http.MethodGet, "/admin_api/partners/allocations"},
		{http.MethodGet, "/admin_api/partners/accounts"},
		{http.MethodGet, "/admin_api/partners/legal-documents"},
		{http.MethodPost, "/admin_api/partners/reminders/send"},
		{http.MethodPost, "/admin_api/partners/invoices/generate-recurring"},
		{http.MethodGet, "/partner_api/me"},
		{http.MethodGet, "/partner_api/services"},
		{http.MethodGet, "/partner_api/usages"},
		{http.MethodGet, "/partner_api/allocations"},
		{http.MethodGet, "/partner_api/documents"},
		{http.MethodGet, "/partner_api/aging"},
		{http.MethodGet, "/partner_api/invoices"},
		{http.MethodGet, "/partner_api/invoices/" + id},
		{http.MethodPost, "/partner_api/invoices/" + id + "/payments"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.method+" "+testCase.path, func(t *testing.T) {
			response, err := app.Test(httptest.NewRequest(testCase.method, testCase.path, nil))
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			if response.StatusCode != fiber.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", response.StatusCode)
			}
		})
	}
}

func TestPartnerRoutesHavePermissionPolicies(t *testing.T) {
	app := setupAppWithPartnerRoutes()

	partnerRoutes := 0
	for _, route := range app.GetRoutes(true) {
		if route.Method == fiber.MethodHead {
			continue
		}
		isPartnerRoute := strings.HasPrefix(route.Path, "/admin_api/partners") || strings.HasPrefix(route.Path, "/partner_api/")
		if !isPartnerRoute {
			continue
		}
		partnerRoutes++
		policy, found := middlewares.MatchRoutePermission(route.Method, route.Path)
		if !found {
			t.Errorf("partner route has no permission policy: %s %s", route.Method, route.Path)
			continue
		}
		if policy.Permission == "" {
			t.Errorf("partner route has an empty permission: %s %s", route.Method, route.Path)
		}
	}
	if partnerRoutes == 0 {
		t.Fatal("no partner routes were registered")
	}
}

// TestPartnerPortalRoutesUseSelfScopedPermissions keeps portal routes on the
// self.partner.* permissions so a partner account can never reach staff APIs.
func TestPartnerPortalRoutesUseSelfScopedPermissions(t *testing.T) {
	app := setupAppWithPartnerRoutes()

	for _, route := range app.GetRoutes(true) {
		if route.Method == fiber.MethodHead {
			continue
		}
		if !strings.HasPrefix(route.Path, "/partner_api/") {
			continue
		}
		policy, found := middlewares.MatchRoutePermission(route.Method, route.Path)
		if !found {
			t.Errorf("portal route has no policy: %s %s", route.Method, route.Path)
			continue
		}
		if !strings.HasPrefix(policy.Permission, "self.partner.") {
			t.Errorf("%s %s uses %q, want a self.partner.* permission", route.Method, route.Path, policy.Permission)
		}
		if policy.DataScope != "partner" {
			t.Errorf("%s %s data scope = %q, want partner", route.Method, route.Path, policy.DataScope)
		}
	}
}

// TestBrevoWebhookIsPublicButSecretProtected documents that the webhook is
// reachable without a user token but refuses to work without a secret.
func TestBrevoWebhookIsPublicButSecretProtected(t *testing.T) {
	t.Setenv("JWT_SECRET", "partner-route-secret")
	t.Setenv("BREVO_WEBHOOK_SECRET", "")
	app := setupAppWithPartnerRoutes()

	response, err := app.Test(httptest.NewRequest(http.MethodPost, "/api/webhooks/brevo", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if response.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 when the webhook secret is not configured", response.StatusCode)
	}
}
