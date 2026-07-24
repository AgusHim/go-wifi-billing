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

func TestPrivateRouteGroupsRejectAnonymousRequests(t *testing.T) {
	t.Setenv("JWT_SECRET", "route-phase-zero-secret")
	app := fiber.New()
	Setup(
		app,
		controllers.NewCoverageController(nil),
		controllers.NewUserController(nil),
		controllers.NewPackageController(nil),
		controllers.NewOdcController(nil),
		controllers.NewOdpController(nil),
		controllers.NewCustomerController(nil),
		controllers.NewSubscriptionController(nil, nil, nil),
		controllers.NewBillController(nil),
		controllers.NewComplainController(nil),
		controllers.NewPaymentController(nil),
		controllers.NewRouterController(nil),
		controllers.NewNOCController(nil, nil, nil),
		controllers.NewProvisioningController(nil),
		controllers.NewNetworkPlanController(nil),
		controllers.NewServiceAccountController(nil),
		controllers.NewRenewalController(nil),
		controllers.NewVoucherController(nil),
		controllers.NewWhatsAppController(nil),
		controllers.NewWhatsAppTemplateController(nil),
		controllers.NewExpenseController(nil),
		controllers.NewFinanceController(nil),
		controllers.NewSettingController(nil),
		controllers.NewInventoryController(nil),
		controllers.NewAccessControlController(nil, nil),
		nil,
	)

	testCases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/users"},
		{http.MethodGet, "/user_api/bills/me"},
		{http.MethodGet, "/user_api/payments/me"},
		{http.MethodGet, "/user_api/subscriptions/me"},
		{http.MethodGet, "/user_api/customers/me"},
		{http.MethodGet, "/admin_api/bills"},
		{http.MethodGet, "/admin_api/payments"},
		{http.MethodGet, "/admin_api/customers"},
		{http.MethodGet, "/admin_api/subscriptions"},
		{http.MethodGet, "/admin_api/complains"},
		{http.MethodGet, "/admin_api/coverages"},
		{http.MethodGet, "/admin_api/packages"},
		{http.MethodGet, "/admin_api/odcs"},
		{http.MethodGet, "/admin_api/odps"},
		{http.MethodGet, "/admin_api/routers"},
		{http.MethodGet, "/admin_api/network-plans"},
		{http.MethodGet, "/admin_api/service-accounts"},
		{http.MethodGet, "/admin_api/provisioning/jobs"},
		{http.MethodGet, "/admin_api/noc/overview"},
		{http.MethodGet, "/admin_api/renewals/subscriptions/" + "00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/admin_api/voucher-batches"},
		{http.MethodPost, "/admin_api/whatsapp/bulk-send"},
		{http.MethodGet, "/admin_api/whatsapp/templates"},
		{http.MethodGet, "/admin_api/expenses"},
		{http.MethodGet, "/admin_api/finance/summary"},
		{http.MethodGet, "/admin_api/settings"},
		{http.MethodGet, "/admin_api/inventory/items"},
		{http.MethodGet, "/admin_api/router-import-batches/00000000-0000-0000-0000-000000000001"},
		{http.MethodGet, "/admin_api/access-control/permissions"},
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

func TestEveryPrivateRouteHasPermissionPolicy(t *testing.T) {
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
		controllers.NewSettingController(nil), controllers.NewInventoryController(nil), controllers.NewAccessControlController(nil, nil), nil,
	)

	for _, route := range app.GetRoutes(true) {
		if route.Method == fiber.MethodHead {
			continue
		}
		private := strings.HasPrefix(route.Path, "/admin_api/") || strings.HasPrefix(route.Path, "/user_api/") ||
			route.Path == "/api/users" || strings.HasPrefix(route.Path, "/api/users/") || route.Path == "/api/auth/me"
		if !private {
			continue
		}
		if _, found := middlewares.MatchRoutePermission(route.Method, route.Path); !found {
			t.Errorf("private route has no permission policy: %s %s", route.Method, route.Path)
		}
	}
}
