package middlewares

import (
	"net/http"
	"strings"

	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
)

type RoutePermissionPolicy struct {
	Method         string
	Path           string
	Permission     string
	AnyPermissions []string
	OwnerOnly      bool
	AuthOnly       bool
	DataScope      string
	Risk           string
}

const RoutePermissionPolicyLocal = "route_permission_policy"

var RoutePermissionPolicies = []RoutePermissionPolicy{
	{Method: http.MethodGet, Path: "/user_api/bills/public/:public_id", Permission: "__public__", DataScope: "public", Risk: "low"},
	// Authenticated self-service routes.
	{http.MethodGet, "/api/auth/me", "", nil, false, true, "self", "low"},
	{http.MethodPut, "/api/auth/me", "", nil, false, true, "self", "low"},
	{http.MethodPost, "/api/users", "users.create", nil, false, false, "global", "high"},
	{http.MethodGet, "/api/users", "users.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/api/users/:id", "users.read", nil, false, false, "global", "medium"},
	{http.MethodPut, "/api/users/:id", "users.update", nil, false, false, "global", "high"},
	{http.MethodDelete, "/api/users/:id", "users.delete", nil, false, false, "global", "critical"},
	{http.MethodGet, "/user_api/customers/me", "", nil, false, true, "self", "low"},
	{http.MethodGet, "/user_api/subscriptions/me", "", nil, false, true, "self", "low"},
	{http.MethodGet, "/user_api/bills", "self.bills.read", nil, false, false, "self", "low"},
	{http.MethodGet, "/user_api/bills/me", "self.bills.read", nil, false, false, "self", "low"},
	{http.MethodGet, "/user_api/payments", "self.payments.read", nil, false, false, "self", "low"},
	{http.MethodGet, "/user_api/payments/me", "self.payments.read", nil, false, false, "self", "low"},
	{http.MethodGet, "/user_api/payments/user/:user_id", "self.payments.read", nil, false, false, "self", "low"},
	{http.MethodPost, "/user_api/payments/midtrans", "self.payments.create", nil, false, false, "self", "medium"},

	// Billing and payments.
	{http.MethodGet, "/admin_api/bills/dashboard/stats", "dashboard.read", nil, false, false, "operational", "low"},
	{http.MethodGet, "/admin_api/bills/dashboard/charts", "dashboard.read", nil, false, false, "operational", "low"},
	{http.MethodGet, "/admin_api/bills/recent/paid", "bills.read", nil, false, false, "operational", "low"},
	{http.MethodGet, "/admin_api/bills/missing/current-month", "bills.read", nil, false, false, "operational", "low"},
	{http.MethodGet, "/admin_api/bills/generate", "bills.generate", nil, false, false, "global", "high"},
	{http.MethodPost, "/admin_api/bills/generate", "bills.generate", nil, false, false, "global", "high"},
	{http.MethodPost, "/admin_api/bills/generate/dry-run", "bills.generate", nil, false, false, "global", "high"},
	{http.MethodPost, "/admin_api/bills/mark-overdue", "bills.mark_overdue", nil, false, false, "global", "high"},
	{http.MethodGet, "/admin_api/bills/send-reminders", "bills.send_reminder", nil, false, false, "global", "medium"},
	{http.MethodPost, "/admin_api/bills/create", "bills.create", nil, false, false, "operational", "medium"},
	{http.MethodGet, "/admin_api/bills", "bills.read", nil, false, false, "operational", "low"},
	{http.MethodGet, "/admin_api/bills/:id", "bills.read", nil, false, false, "operational", "low"},
	{http.MethodPut, "/admin_api/bills/:id", "bills.update", nil, false, false, "operational", "high"},
	{http.MethodDelete, "/admin_api/bills/generated/current-month/unpaid", "bills.delete", nil, false, false, "global", "critical"},
	{http.MethodDelete, "/admin_api/bills/:id", "bills.delete", nil, false, false, "global", "critical"},
	{http.MethodPost, "/admin_api/payments", "payments.create", nil, false, false, "operational", "high"},
	{http.MethodPost, "/admin_api/payments/batch", "payments.create", nil, false, false, "operational", "high"},
	{http.MethodGet, "/admin_api/payments", "payments.read", nil, false, false, "operational", "low"},
	{http.MethodGet, "/admin_api/payments/export/csv", "payments.export", nil, false, false, "operational", "medium"},
	{http.MethodGet, "/admin_api/payments/:id", "payments.read", nil, false, false, "operational", "low"},
	{http.MethodPut, "/admin_api/payments/:id", "payments.update", nil, false, false, "operational", "high"},
	{http.MethodDelete, "/admin_api/payments/:id", "payments.delete", nil, false, false, "global", "critical"},

	// Customers, subscriptions, and complaints.
	{http.MethodGet, "/admin_api/customers/export", "customers.export", nil, false, false, "operational", "medium"},
	{http.MethodPost, "/admin_api/customers/import", "customers.import", nil, false, false, "operational", "high"},
	{http.MethodGet, "/admin_api/customers", "customers.read", nil, false, false, "operational", "low"},
	{http.MethodGet, "/admin_api/customers/by_user/:user_id", "customers.read", nil, false, false, "operational", "low"},
	{http.MethodGet, "/admin_api/customers/:id", "customers.read", nil, false, false, "operational", "low"},
	{http.MethodPost, "/admin_api/customers", "customers.create", nil, false, false, "operational", "medium"},
	{http.MethodPut, "/admin_api/customers/:id", "customers.update", nil, false, false, "operational", "medium"},
	{http.MethodDelete, "/admin_api/customers/:id", "customers.delete", nil, false, false, "global", "critical"},
	{http.MethodGet, "/admin_api/subscriptions", "subscriptions.read", nil, false, false, "operational", "low"},
	{http.MethodGet, "/admin_api/subscriptions/:id", "subscriptions.read", nil, false, false, "operational", "low"},
	{http.MethodGet, "/admin_api/subscriptions/customer/:customer_id", "subscriptions.read", nil, false, false, "operational", "low"},
	{http.MethodPost, "/admin_api/subscriptions", "subscriptions.create", nil, false, false, "operational", "medium"},
	{http.MethodPut, "/admin_api/subscriptions/:id", "subscriptions.update", nil, false, false, "operational", "high"},
	{http.MethodDelete, "/admin_api/subscriptions/:id", "subscriptions.delete", nil, false, false, "global", "critical"},
	{http.MethodPost, "/admin_api/complains", "", []string{"complaints.create", "self.complaints.manage"}, false, false, "ownership_or_operational", "low"},
	{http.MethodGet, "/admin_api/complains", "", []string{"complaints.read", "self.complaints.manage"}, false, false, "ownership_or_operational", "low"},
	{http.MethodGet, "/admin_api/complains/:id", "", []string{"complaints.read", "self.complaints.manage"}, false, false, "ownership_or_operational", "low"},
	{http.MethodPut, "/admin_api/complains/:id", "", []string{"complaints.update", "self.complaints.manage"}, false, false, "ownership_or_operational", "medium"},
	{http.MethodDelete, "/admin_api/complains/:id", "", []string{"complaints.delete", "self.complaints.manage"}, false, false, "ownership_or_operational", "high"},

	// Master network data.
	{http.MethodGet, "/admin_api/coverages", "coverages.read", nil, false, false, "global", "low"},
	{http.MethodGet, "/admin_api/coverages/:id", "coverages.read", nil, false, false, "global", "low"},
	{http.MethodPost, "/admin_api/coverages", "coverages.create", nil, false, false, "global", "high"},
	{http.MethodPut, "/admin_api/coverages/:id", "coverages.update", nil, false, false, "global", "high"},
	{http.MethodDelete, "/admin_api/coverages/:id", "coverages.delete", nil, false, false, "global", "critical"},
	{http.MethodGet, "/admin_api/packages", "packages.read", nil, false, false, "global", "low"},
	{http.MethodGet, "/admin_api/packages/:id", "packages.read", nil, false, false, "global", "low"},
	{http.MethodPost, "/admin_api/packages", "packages.create", nil, false, false, "global", "high"},
	{http.MethodPut, "/admin_api/packages/:id", "packages.update", nil, false, false, "global", "high"},
	{http.MethodDelete, "/admin_api/packages/:id", "packages.delete", nil, false, false, "global", "critical"},
	{http.MethodGet, "/admin_api/odcs", "odcs.read", nil, false, false, "global", "low"},
	{http.MethodGet, "/admin_api/odcs/:id", "odcs.read", nil, false, false, "global", "low"},
	{http.MethodPost, "/admin_api/odcs", "odcs.create", nil, false, false, "global", "high"},
	{http.MethodPut, "/admin_api/odcs/:id", "odcs.update", nil, false, false, "global", "high"},
	{http.MethodDelete, "/admin_api/odcs/:id", "odcs.delete", nil, false, false, "global", "critical"},
	{http.MethodGet, "/admin_api/odps", "odps.read", nil, false, false, "global", "low"},
	{http.MethodGet, "/admin_api/odps/:id", "odps.read", nil, false, false, "global", "low"},
	{http.MethodPost, "/admin_api/odps", "odps.create", nil, false, false, "global", "high"},
	{http.MethodPut, "/admin_api/odps/:id", "odps.update", nil, false, false, "global", "high"},
	{http.MethodDelete, "/admin_api/odps/:id", "odps.delete", nil, false, false, "global", "critical"},

	// Routers, provisioning, and NOC.
	{http.MethodGet, "/admin_api/routers", "routers.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/routers/:id", "routers.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/routers/:id/resources", "routers.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/routers/:id/import-preview", "routers.import", nil, false, false, "global", "critical"},
	{http.MethodGet, "/admin_api/routers/:id/import-batches", "routers.import", nil, false, false, "global", "critical"},
	{http.MethodPost, "/admin_api/routers", "routers.create", nil, false, false, "global", "critical"},
	{http.MethodPost, "/admin_api/routers/health-check", "routers.health_check", nil, false, false, "global", "high"},
	{http.MethodPost, "/admin_api/routers/:id/import-staging", "routers.import", nil, false, false, "global", "critical"},
	{http.MethodPost, "/admin_api/routers/:id/test-connection", "routers.test_connection", nil, false, false, "global", "high"},
	{http.MethodPut, "/admin_api/routers/:id", "routers.update", nil, false, false, "global", "critical"},
	{http.MethodDelete, "/admin_api/routers/:id", "routers.delete", nil, false, false, "global", "critical"},
	{http.MethodGet, "/admin_api/router-import-batches/:id", "routers.import", nil, false, false, "global", "critical"},
	{http.MethodGet, "/admin_api/network-plans", "network_plans.read", nil, false, false, "global", "low"},
	{http.MethodGet, "/admin_api/network-plans/:id", "network_plans.read", nil, false, false, "global", "low"},
	{http.MethodPost, "/admin_api/network-plans", "network_plans.create", nil, false, false, "global", "high"},
	{http.MethodPut, "/admin_api/network-plans/:id", "network_plans.update", nil, false, false, "global", "high"},
	{http.MethodDelete, "/admin_api/network-plans/:id", "network_plans.delete", nil, false, false, "global", "critical"},
	{http.MethodPost, "/admin_api/network-plans/sync-from-router", "network_plans.sync", nil, false, false, "global", "critical"},
	{http.MethodGet, "/admin_api/provisioning/jobs", "provisioning_logs.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/provisioning/logs", "provisioning_logs.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/noc/overview", "noc.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/noc/routers", "noc.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/noc/routers/:id/snapshots", "noc.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/noc/routers/:id/interfaces", "noc.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/noc/customers", "noc.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/noc/reconciliation", "noc.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/noc/alerts", "noc.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/noc/metrics", "noc.read", nil, false, false, "global", "medium"},
	{http.MethodPost, "/admin_api/noc/collect", "noc.collect", nil, false, false, "global", "high"},
	{http.MethodPost, "/admin_api/noc/alerts/evaluate", "noc.evaluate_alerts", nil, false, false, "global", "high"},
	{http.MethodPost, "/admin_api/noc/alerts/:id/ack", "noc.manage_alerts", nil, false, false, "global", "high"},
	{http.MethodPost, "/admin_api/noc/alerts/:id/resolve", "noc.manage_alerts", nil, false, false, "global", "high"},
	{http.MethodPost, "/admin_api/noc/service-accounts/:id/actions", "noc.run_action", nil, false, false, "global", "critical"},
	{http.MethodPost, "/admin_api/noc/reconciliation/:id/resolve", "noc.reconcile", nil, false, false, "global", "critical"},

	// Service accounts and renewals.
	{http.MethodGet, "/admin_api/service-accounts", "service_accounts.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/service-accounts/:id", "service_accounts.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/service-accounts/:id/status-history", "service_accounts.read", nil, false, false, "global", "medium"},
	{http.MethodPost, "/admin_api/service-accounts", "service_accounts.create", nil, false, false, "global", "high"},
	{http.MethodPut, "/admin_api/service-accounts/:id", "service_accounts.update", nil, false, false, "global", "high"},
	{http.MethodDelete, "/admin_api/service-accounts/:id", "service_accounts.delete", nil, false, false, "global", "critical"},
	{http.MethodPost, "/admin_api/service-accounts/:id/provision", "service_accounts.provision", nil, false, false, "global", "critical"},
	{http.MethodPost, "/admin_api/service-accounts/:id/suspend", "service_accounts.suspend", nil, false, false, "global", "critical"},
	{http.MethodPost, "/admin_api/service-accounts/:id/unsuspend", "service_accounts.unsuspend", nil, false, false, "global", "critical"},
	{http.MethodPost, "/admin_api/service-accounts/:id/terminate", "service_accounts.terminate", nil, false, false, "global", "critical"},
	{http.MethodPost, "/admin_api/service-accounts/:id/change-plan", "service_accounts.change_plan", nil, false, false, "global", "critical"},
	{http.MethodPost, "/admin_api/renewals/auto-generate", "subscriptions.renew", nil, false, false, "global", "high"},
	{http.MethodGet, "/admin_api/renewals/subscriptions/:id", "subscriptions.read", nil, false, false, "global", "low"},
	{http.MethodPost, "/admin_api/renewals/subscriptions/:id/sync-recurring", "subscriptions.renew", nil, false, false, "global", "high"},

	// Administration.
	{http.MethodGet, "/admin_api/expenses", "expenses.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/expenses/:id", "expenses.read", nil, false, false, "global", "medium"},
	{http.MethodPost, "/admin_api/expenses", "expenses.create", nil, false, false, "global", "high"},
	{http.MethodPut, "/admin_api/expenses/:id", "expenses.update", nil, false, false, "global", "high"},
	{http.MethodDelete, "/admin_api/expenses/:id", "expenses.delete", nil, false, false, "global", "critical"},
	{http.MethodGet, "/admin_api/finance/summary", "finance.read", nil, false, false, "global", "high"},
	{http.MethodGet, "/admin_api/finance/monthly", "finance.read", nil, false, false, "global", "high"},
	{http.MethodGet, "/admin_api/finance/by-subscription", "finance.read", nil, false, false, "global", "high"},
	{http.MethodGet, "/admin_api/settings", "settings.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/settings/:key", "settings.read", nil, false, false, "global", "medium"},
	{http.MethodPut, "/admin_api/settings/:key", "settings.update", nil, false, false, "global", "critical"},
	{http.MethodPost, "/admin_api/whatsapp/bulk-send", "whatsapp.send", nil, false, false, "global", "high"},
	{http.MethodGet, "/admin_api/whatsapp/templates", "whatsapp.read", nil, false, false, "global", "medium"},
	{http.MethodGet, "/admin_api/whatsapp/templates/:id", "whatsapp.read", nil, false, false, "global", "medium"},
	{http.MethodPost, "/admin_api/whatsapp/templates", "whatsapp_templates.manage", nil, false, false, "global", "high"},
	{http.MethodPut, "/admin_api/whatsapp/templates/:id", "whatsapp_templates.manage", nil, false, false, "global", "high"},
	{http.MethodDelete, "/admin_api/whatsapp/templates/:id", "whatsapp_templates.manage", nil, false, false, "global", "high"},
	{http.MethodGet, "/admin_api/voucher-batches", "vouchers.read", nil, false, false, "global", "low"},
	{http.MethodGet, "/admin_api/voucher-batches/:id", "vouchers.read", nil, false, false, "global", "low"},
	{http.MethodPost, "/admin_api/voucher-batches", "vouchers.create", nil, false, false, "global", "high"},
	{http.MethodGet, "/admin_api/vouchers", "vouchers.read", nil, false, false, "global", "low"},
	{http.MethodGet, "/admin_api/vouchers/:id", "vouchers.read", nil, false, false, "global", "low"},

	// Inventory.
	{http.MethodGet, "/admin_api/inventory/items", "inventory.read", nil, false, false, "global", "low"},
	{http.MethodPost, "/admin_api/inventory/items", "inventory.manage_master", nil, false, false, "global", "high"},
	{http.MethodGet, "/admin_api/inventory/items/:id", "inventory.read", nil, false, false, "global", "low"},
	{http.MethodPut, "/admin_api/inventory/items/:id", "inventory.manage_master", nil, false, false, "global", "high"},
	{http.MethodDelete, "/admin_api/inventory/items/:id", "inventory.manage_master", nil, false, false, "global", "high"},
}

// Repetitive inventory policies are appended programmatically to keep the
// registry readable while still producing explicit method/path entries.
func init() {
	addInventoryPolicies()
	addAccessControlPolicies()
}

func EnforceRoutePermissions(authorizationService services.AuthorizationService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		policy, found := MatchRoutePermission(c.Method(), c.Path())
		if !found {
			if isPrivateNamespace(c.Path()) {
				recordAuthorizationStatus(c, "__missing_policy__", fiber.StatusForbidden)
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "route permission policy missing"})
			}
			return c.Next()
		}
		c.Locals(RoutePermissionPolicyLocal, policy)
		if policy.Permission == "__public__" {
			return c.Next()
		}
		if err := AuthenticateRequest(c); err != nil {
			recordAuthorizationStatus(c, routePolicyPermissionLabel(policy), fiber.StatusUnauthorized)
			return unauthorizedResponse(c)
		}
		if policy.AuthOnly {
			return c.Next()
		}
		if policy.OwnerOnly {
			return RequireOwner(authorizationService)(c)
		}
		if len(policy.AnyPermissions) > 0 {
			return RequireAnyPermission(authorizationService, policy.AnyPermissions...)(c)
		}
		return RequirePermission(authorizationService, policy.Permission)(c)
	}
}

func routePolicyPermissionLabel(policy RoutePermissionPolicy) string {
	if policy.OwnerOnly {
		return "__owner__"
	}
	if len(policy.AnyPermissions) > 0 {
		return strings.Join(policy.AnyPermissions, ",")
	}
	if policy.AuthOnly {
		return "__authentication__"
	}
	return policy.Permission
}

func MatchRoutePermission(method, path string) (RoutePermissionPolicy, bool) {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = normalizeRoutePath(path)
	for _, policy := range RoutePermissionPolicies {
		if policy.Method == method && matchRoutePath(policy.Path, path) {
			return policy, true
		}
	}
	return RoutePermissionPolicy{}, false
}

func isPrivateNamespace(path string) bool {
	return strings.HasPrefix(path, "/admin_api/") || strings.HasPrefix(path, "/user_api/") ||
		path == "/api/users" || strings.HasPrefix(path, "/api/users/") || path == "/api/auth/me"
}

func matchRoutePath(pattern, actual string) bool {
	patternParts := strings.Split(strings.Trim(normalizeRoutePath(pattern), "/"), "/")
	actualParts := strings.Split(strings.Trim(normalizeRoutePath(actual), "/"), "/")
	if len(patternParts) != len(actualParts) {
		return false
	}
	for index := range patternParts {
		if strings.HasPrefix(patternParts[index], ":") {
			if actualParts[index] == "" {
				return false
			}
			continue
		}
		if patternParts[index] != actualParts[index] {
			return false
		}
	}
	return true
}

func normalizeRoutePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if len(path) > 1 {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}

func addAccessControlPolicies() {
	paths := []struct{ method, path string }{
		{http.MethodGet, "/admin_api/access-control/permissions"}, {http.MethodGet, "/admin_api/access-control/roles"},
		{http.MethodGet, "/admin_api/access-control/roles/:id"}, {http.MethodPost, "/admin_api/access-control/roles"},
		{http.MethodPut, "/admin_api/access-control/roles/:id"}, {http.MethodPut, "/admin_api/access-control/roles/:id/permissions"},
		{http.MethodDelete, "/admin_api/access-control/roles/:id"}, {http.MethodGet, "/admin_api/access-control/users"},
		{http.MethodGet, "/admin_api/access-control/users/:id"}, {http.MethodPut, "/admin_api/access-control/users/:id"},
		{http.MethodDelete, "/admin_api/access-control/users/:id/overrides"}, {http.MethodGet, "/admin_api/access-control/audit-logs"},
		{http.MethodGet, "/admin_api/access-control/audit-logs/export"}, {http.MethodGet, "/admin_api/access-control/metrics"},
	}
	for _, item := range paths {
		RoutePermissionPolicies = append(RoutePermissionPolicies, RoutePermissionPolicy{Method: item.method, Path: item.path, OwnerOnly: true, DataScope: "owner", Risk: "critical"})
	}
}

func addInventoryPolicies() {
	type inventoryPolicy struct{ method, path, permission, risk string }
	policies := []inventoryPolicy{
		{http.MethodGet, "/admin_api/inventory/locations", "inventory.read", "low"}, {http.MethodPost, "/admin_api/inventory/locations", "inventory.manage_master", "high"},
		{http.MethodGet, "/admin_api/inventory/locations/:id", "inventory.read", "low"}, {http.MethodPut, "/admin_api/inventory/locations/:id", "inventory.manage_master", "high"}, {http.MethodDelete, "/admin_api/inventory/locations/:id", "inventory.manage_master", "high"},
		{http.MethodGet, "/admin_api/inventory/stocks", "inventory.read", "low"}, {http.MethodGet, "/admin_api/inventory/movements", "inventory.read", "low"}, {http.MethodGet, "/admin_api/inventory/serial-items", "inventory.read", "low"},
		{http.MethodGet, "/admin_api/inventory/suppliers", "inventory.read", "low"}, {http.MethodPost, "/admin_api/inventory/suppliers", "inventory.manage_master", "high"}, {http.MethodGet, "/admin_api/inventory/suppliers/:id", "inventory.read", "low"}, {http.MethodPut, "/admin_api/inventory/suppliers/:id", "inventory.manage_master", "high"}, {http.MethodDelete, "/admin_api/inventory/suppliers/:id", "inventory.manage_master", "high"},
		{http.MethodGet, "/admin_api/inventory/purchase-orders", "inventory.read", "low"}, {http.MethodPost, "/admin_api/inventory/purchase-orders", "inventory.purchase", "high"}, {http.MethodGet, "/admin_api/inventory/purchase-orders/:id", "inventory.read", "low"}, {http.MethodPut, "/admin_api/inventory/purchase-orders/:id", "inventory.purchase", "high"},
		{http.MethodGet, "/admin_api/inventory/goods-receipts", "inventory.read", "low"}, {http.MethodPost, "/admin_api/inventory/goods-receipts", "inventory.receive", "high"}, {http.MethodGet, "/admin_api/inventory/goods-receipts/:id", "inventory.read", "low"},
		{http.MethodGet, "/admin_api/inventory/stock-transfers", "inventory.read", "low"}, {http.MethodPost, "/admin_api/inventory/stock-transfers", "inventory.transfer", "high"}, {http.MethodGet, "/admin_api/inventory/stock-transfers/:id", "inventory.read", "low"}, {http.MethodPost, "/admin_api/inventory/stock-transfers/:id/receive", "inventory.transfer", "high"},
		{http.MethodGet, "/admin_api/inventory/material-usages", "inventory.read", "low"}, {http.MethodPost, "/admin_api/inventory/material-usages", "inventory.use_material", "high"}, {http.MethodGet, "/admin_api/inventory/material-usages/:id", "inventory.read", "low"},
		{http.MethodGet, "/admin_api/inventory/customer-assets", "inventory.read", "low"}, {http.MethodPost, "/admin_api/inventory/customer-assets/:id/return", "inventory.use_material", "high"}, {http.MethodPost, "/admin_api/inventory/customer-assets/:id/replace", "inventory.use_material", "high"}, {http.MethodPost, "/admin_api/inventory/customer-assets/:id/status", "inventory.use_material", "high"},
		{http.MethodGet, "/admin_api/inventory/network-assets", "inventory.read", "low"}, {http.MethodPost, "/admin_api/inventory/network-assets/:id/status", "inventory.use_material", "high"},
		{http.MethodGet, "/admin_api/inventory/stock-opnames", "inventory.read", "low"}, {http.MethodPost, "/admin_api/inventory/stock-opnames", "inventory.stock_opname", "high"}, {http.MethodGet, "/admin_api/inventory/stock-opnames/:id", "inventory.read", "low"}, {http.MethodPost, "/admin_api/inventory/stock-opnames/:id/submit", "inventory.stock_opname", "high"}, {http.MethodPost, "/admin_api/inventory/stock-opnames/:id/approve", "inventory.approve", "critical"},
		{http.MethodGet, "/admin_api/inventory/accounting/accounts", "inventory.accounting.read", "high"}, {http.MethodGet, "/admin_api/inventory/accounting/journals", "inventory.accounting.read", "high"}, {http.MethodGet, "/admin_api/inventory/accounting/valuation", "inventory.accounting.read", "high"}, {http.MethodGet, "/admin_api/inventory/accounting/period-locks", "inventory.accounting.read", "high"}, {http.MethodPost, "/admin_api/inventory/accounting/period-locks", "inventory.accounting.manage", "critical"},
		{http.MethodGet, "/admin_api/inventory/supplier-invoices", "inventory.accounting.read", "high"}, {http.MethodPost, "/admin_api/inventory/supplier-invoices", "inventory.accounting.manage", "critical"}, {http.MethodGet, "/admin_api/inventory/supplier-invoices/:id", "inventory.accounting.read", "high"}, {http.MethodGet, "/admin_api/inventory/supplier-payments", "inventory.accounting.read", "high"}, {http.MethodPost, "/admin_api/inventory/supplier-payments", "inventory.accounting.manage", "critical"},
	}
	for _, item := range policies {
		RoutePermissionPolicies = append(RoutePermissionPolicies, RoutePermissionPolicy{Method: item.method, Path: item.path, Permission: item.permission, DataScope: "global", Risk: item.risk})
	}
}
