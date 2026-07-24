package controllers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAccessControlAPIIsOwnerOnly(t *testing.T) {
	app, _, owner, admin, _ := newAccessControlControllerTestApp(t)
	testCases := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "anonymous", wantStatus: fiber.StatusUnauthorized},
		{name: "admin with delegated manage key", token: signControllerToken(t, admin.ID, "admin"), wantStatus: fiber.StatusForbidden},
		{name: "owner", token: signControllerToken(t, owner.ID, "owner"), wantStatus: fiber.StatusOK},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/admin_api/access-control/permissions", nil)
			if testCase.token != "" {
				request.Header.Set("Authorization", "Bearer "+testCase.token)
			}
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			if response.StatusCode != testCase.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, testCase.wantStatus)
			}
		})
	}
}

func TestAllAccessControlRoutesRejectNonOwner(t *testing.T) {
	app, _, _, admin, target := newAccessControlControllerTestApp(t)
	roleID := target.RoleID.String()
	userID := target.ID.String()
	testCases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin_api/access-control/permissions"},
		{http.MethodGet, "/admin_api/access-control/roles"},
		{http.MethodGet, "/admin_api/access-control/roles/" + roleID},
		{http.MethodPost, "/admin_api/access-control/roles"},
		{http.MethodPut, "/admin_api/access-control/roles/" + roleID},
		{http.MethodPut, "/admin_api/access-control/roles/" + roleID + "/permissions"},
		{http.MethodDelete, "/admin_api/access-control/roles/" + roleID},
		{http.MethodGet, "/admin_api/access-control/users"},
		{http.MethodGet, "/admin_api/access-control/users/" + userID},
		{http.MethodPut, "/admin_api/access-control/users/" + userID},
		{http.MethodDelete, "/admin_api/access-control/users/" + userID + "/overrides"},
		{http.MethodGet, "/admin_api/access-control/audit-logs"},
		{http.MethodGet, "/admin_api/access-control/audit-logs/export"},
		{http.MethodGet, "/admin_api/access-control/metrics"},
	}
	token := signControllerToken(t, admin.ID, "admin")
	for _, testCase := range testCases {
		t.Run(testCase.method+" "+testCase.path, func(t *testing.T) {
			request := httptest.NewRequest(testCase.method, testCase.path, nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			if response.StatusCode != fiber.StatusForbidden {
				t.Fatalf("status = %d, want 403", response.StatusCode)
			}
		})
	}
}

func TestAccessControlOwnerCanExportAuditCSVAndReadMetrics(t *testing.T) {
	app, database, owner, _, target := newAccessControlControllerTestApp(t)
	if err := database.Create(&models.AccessAuditLog{
		ActorUserID: owner.ID, TargetType: "user", TargetID: target.ID,
		Action: "user_role_changed", Reason: "phase-six-export",
	}).Error; err != nil {
		t.Fatalf("create audit: %v", err)
	}
	token := signControllerToken(t, owner.ID, "owner")
	for _, testCase := range []struct {
		path        string
		contentType string
	}{
		{path: "/admin_api/access-control/audit-logs/export", contentType: "text/csv"},
		{path: "/admin_api/access-control/metrics", contentType: "application/json"},
	} {
		request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("GET %s: %v", testCase.path, err)
		}
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", testCase.path, response.StatusCode)
		}
		if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, testCase.contentType) {
			t.Fatalf("GET %s content type = %q", testCase.path, contentType)
		}
	}
}

func TestAccessControlAPIMapsOptimisticConflictTo409(t *testing.T) {
	app, _, owner, _, target := newAccessControlControllerTestApp(t)
	body := []byte(`{"role_id":"` + target.RoleID.String() + `","overrides":[],"expected_permission_version":99}`)
	request := httptest.NewRequest(http.MethodPut, "/admin_api/access-control/users/"+target.ID.String(), bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+signControllerToken(t, owner.ID, "owner"))
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if response.StatusCode != fiber.StatusConflict {
		t.Fatalf("status = %d, want 409", response.StatusCode)
	}
}

func newAccessControlControllerTestApp(t *testing.T) (*fiber.App, *gorm.DB, models.User, models.User, models.User) {
	t.Helper()
	t.Setenv("JWT_SECRET", "controller-phase-zero-secret")
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_foreign_keys=1"
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := database.AutoMigrate(
		&models.Coverage{}, &models.Role{}, &models.Permission{}, &models.User{},
		&models.RolePermission{}, &models.UserPermissionOverride{}, &models.AccessAuditLog{},
	); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	ownerRole := models.Role{Key: "owner", Name: "Owner", IsSystem: true, IsOwner: true, IsActive: true}
	adminRole := models.Role{Key: "admin", Name: "Admin", IsSystem: true, IsActive: true}
	customerRole := models.Role{Key: "customer", Name: "Customer", IsSystem: true, IsActive: true}
	for _, role := range []*models.Role{&ownerRole, &adminRole, &customerRole} {
		if err := database.Create(role).Error; err != nil {
			t.Fatalf("create role: %v", err)
		}
	}
	permission := models.Permission{Key: "access_control.manage", Module: "access_control", Action: "manage", Name: "Manage access", RiskLevel: "critical"}
	if err := database.Create(&permission).Error; err != nil {
		t.Fatalf("create permission: %v", err)
	}
	if err := database.Create(&models.RolePermission{RoleID: adminRole.ID, PermissionID: permission.ID}).Error; err != nil {
		t.Fatalf("create admin permission: %v", err)
	}
	owner := createAccessControlControllerUser(t, database, "owner-api@example.com", ownerRole)
	admin := createAccessControlControllerUser(t, database, "admin-api@example.com", adminRole)
	target := createAccessControlControllerUser(t, database, "target-api@example.com", customerRole)

	authorizer := services.NewAuthorizationService(repositories.NewAuthorizationRepository(database))
	service := services.NewAccessControlService(repositories.NewAccessControlRepository(database), authorizer)
	controller := NewAccessControlController(service, authorizer)
	app := fiber.New()
	controller.RegisterRoutes(app)
	return app, database, owner, admin, target
}

func createAccessControlControllerUser(t *testing.T, database *gorm.DB, email string, role models.Role) models.User {
	t.Helper()
	roleID := role.ID
	user := models.User{ID: uuid.New(), Name: email, Email: email, Role: role.Key, RoleID: &roleID, IsActive: true, PermissionVersion: 1}
	if err := database.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}
