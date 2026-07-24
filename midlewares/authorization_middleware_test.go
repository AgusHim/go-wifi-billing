package middlewares

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type authorizationServiceStub struct {
	decision *services.AuthorizationDecision
	err      error
	calls    int
}

func (s *authorizationServiceStub) Resolve(context.Context, uuid.UUID) (*services.AuthorizationDecision, error) {
	s.calls++
	return s.decision, s.err
}

func TestPermissionMiddlewareDecisions(t *testing.T) {
	t.Setenv("JWT_SECRET", "phase-zero-test-secret")
	decision := &services.AuthorizationDecision{
		UserID:      uuid.New(),
		RoleKey:     "admin",
		Permissions: map[string]services.PermissionSource{"customers.read": services.PermissionSourceRole, "customers.update": services.PermissionSourceRole},
	}

	testCases := []struct {
		name       string
		guard      func(services.AuthorizationService) fiber.Handler
		wantStatus int
	}{
		{name: "single allow", guard: func(service services.AuthorizationService) fiber.Handler {
			return RequirePermission(service, "customers.read")
		}, wantStatus: fiber.StatusOK},
		{name: "single deny", guard: func(service services.AuthorizationService) fiber.Handler {
			return RequirePermission(service, "customers.delete")
		}, wantStatus: fiber.StatusForbidden},
		{name: "any allow", guard: func(service services.AuthorizationService) fiber.Handler {
			return RequireAnyPermission(service, "customers.delete", "customers.read")
		}, wantStatus: fiber.StatusOK},
		{name: "any deny", guard: func(service services.AuthorizationService) fiber.Handler {
			return RequireAnyPermission(service, "customers.delete", "bills.delete")
		}, wantStatus: fiber.StatusForbidden},
		{name: "all allow", guard: func(service services.AuthorizationService) fiber.Handler {
			return RequireAllPermissions(service, "customers.read", "customers.update")
		}, wantStatus: fiber.StatusOK},
		{name: "all deny", guard: func(service services.AuthorizationService) fiber.Handler {
			return RequireAllPermissions(service, "customers.read", "customers.delete")
		}, wantStatus: fiber.StatusForbidden},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			service := &authorizationServiceStub{decision: decision}
			status := authorizationMiddlewareStatus(t, testCase.guard(service))
			if status != testCase.wantStatus {
				t.Fatalf("status = %d, want %d", status, testCase.wantStatus)
			}
			if service.calls != 1 {
				t.Fatalf("resolve calls = %d, want 1", service.calls)
			}
		})
	}
}

func TestRequireOwnerRejectsNonOwnerEvenWithManagePermission(t *testing.T) {
	t.Setenv("JWT_SECRET", "phase-zero-test-secret")
	service := &authorizationServiceStub{decision: &services.AuthorizationDecision{
		UserID:      uuid.New(),
		RoleKey:     "admin",
		IsOwner:     false,
		Permissions: map[string]services.PermissionSource{"access_control.manage": services.PermissionSourceUserAllow},
	}}

	status := authorizationMiddlewareStatus(t, RequireOwner(service))
	if status != fiber.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
}

func TestPermissionMiddlewareRejectsInactivePrincipalOnEveryRequest(t *testing.T) {
	t.Setenv("JWT_SECRET", "phase-zero-test-secret")
	service := &authorizationServiceStub{err: services.ErrAuthorizationUserInactive}
	guard := RequirePermission(service, "customers.read")

	for requestNumber := 1; requestNumber <= 2; requestNumber++ {
		status := authorizationMiddlewareStatus(t, guard)
		if status != fiber.StatusForbidden {
			t.Fatalf("request %d status = %d, want 403", requestNumber, status)
		}
	}
	if service.calls != 2 {
		t.Fatalf("resolve calls = %d, want 2", service.calls)
	}
}

func TestPermissionMiddlewareReturns500ForResolverFailure(t *testing.T) {
	t.Setenv("JWT_SECRET", "phase-zero-test-secret")
	service := &authorizationServiceStub{err: errors.New("database unavailable")}
	status := authorizationMiddlewareStatus(t, RequirePermission(service, "customers.read"))
	if status != fiber.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", status)
	}
}

func TestAuthorizationRolloutModesAndBaselineRollback(t *testing.T) {
	t.Setenv("JWT_SECRET", "phase-zero-test-secret")
	decision := &services.AuthorizationDecision{
		UserID: uuid.New(), RoleKey: "loket",
		Permissions:         map[string]services.PermissionSource{},
		BaselinePermissions: map[string]bool{"customers.read": true},
	}
	testCases := []struct {
		name             string
		mode             string
		modules          string
		baselineRollback string
		permission       string
		wantStatus       int
	}{
		{name: "shadow observes but allows", mode: "shadow", permission: "customers.delete", wantStatus: fiber.StatusOK},
		{name: "warning allows module outside allowlist", mode: "warning", modules: "bills", permission: "customers.delete", wantStatus: fiber.StatusOK},
		{name: "warning enforces selected module", mode: "warning", modules: "customers,bills", permission: "customers.delete", wantStatus: fiber.StatusForbidden},
		{name: "enforce denies", mode: "enforce", permission: "customers.delete", wantStatus: fiber.StatusForbidden},
		{name: "baseline rollback allows role default", mode: "enforce", baselineRollback: "true", permission: "customers.read", wantStatus: fiber.StatusOK},
		{name: "baseline rollback does not grant new permission", mode: "enforce", baselineRollback: "true", permission: "customers.delete", wantStatus: fiber.StatusForbidden},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("AUTHZ_ENFORCEMENT_MODE", testCase.mode)
			t.Setenv("AUTHZ_ENFORCED_MODULES", testCase.modules)
			t.Setenv("AUTHZ_ROLLBACK_BASELINE", testCase.baselineRollback)
			status := authorizationMiddlewareStatus(t, RequirePermission(&authorizationServiceStub{decision: decision}, testCase.permission))
			if status != testCase.wantStatus {
				t.Fatalf("status = %d, want %d", status, testCase.wantStatus)
			}
		})
	}
}

func TestShadowModeNeverRelaxesOwnerGuard(t *testing.T) {
	t.Setenv("JWT_SECRET", "phase-zero-test-secret")
	t.Setenv("AUTHZ_ENFORCEMENT_MODE", "shadow")
	service := &authorizationServiceStub{decision: &services.AuthorizationDecision{
		UserID: uuid.New(), RoleKey: "admin", Permissions: map[string]services.PermissionSource{},
	}}
	if status := authorizationMiddlewareStatus(t, RequireOwner(service)); status != fiber.StatusForbidden {
		t.Fatalf("owner status = %d, want 403", status)
	}
}

func authorizationMiddlewareStatus(t *testing.T, guard fiber.Handler) int {
	t.Helper()
	app := fiber.New()
	app.Get("/private", UserProtected(), guard, func(c *fiber.Ctx) error {
		if c.Locals(AuthorizationDecisionLocal) == nil {
			t.Fatal("authorization decision was not stored")
		}
		return c.SendStatus(fiber.StatusOK)
	})

	request := httptest.NewRequest("GET", "/private", nil)
	request.Header.Set("Authorization", "Bearer "+signMiddlewareToken(t, jwt.SigningMethodHS256, validMiddlewareClaims("admin")))
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return response.StatusCode
}
