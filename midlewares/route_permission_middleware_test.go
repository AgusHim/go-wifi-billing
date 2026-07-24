package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRoutePermissionRegistryEnforcesMethodSpecificPermission(t *testing.T) {
	t.Setenv("JWT_SECRET", "phase-zero-test-secret")
	service := &authorizationServiceStub{decision: &services.AuthorizationDecision{
		UserID:      uuid.New(),
		RoleKey:     "admin",
		Permissions: map[string]services.PermissionSource{"customers.read": services.PermissionSourceRole},
	}}
	app := fiber.New()
	app.Use(EnforceRoutePermissions(service))
	app.Get("/admin_api/customers", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Post("/admin_api/customers", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusCreated) })

	token := signMiddlewareToken(t, jwt.SigningMethodHS256, validMiddlewareClaims("admin"))
	for _, testCase := range []struct {
		method string
		want   int
	}{
		{method: http.MethodGet, want: fiber.StatusOK},
		{method: http.MethodPost, want: fiber.StatusForbidden},
	} {
		request := httptest.NewRequest(testCase.method, "/admin_api/customers", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("%s request: %v", testCase.method, err)
		}
		if response.StatusCode != testCase.want {
			t.Errorf("%s status = %d, want %d", testCase.method, response.StatusCode, testCase.want)
		}
	}
}

func TestRoutePermissionRegistryFailsClosedAndAllowsExplicitPublicRoute(t *testing.T) {
	app := fiber.New()
	app.Use(EnforceRoutePermissions(nil))
	app.Get("/admin_api/unmapped", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Get("/user_api/bills/public/:public_id", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	for _, testCase := range []struct {
		path string
		want int
	}{
		{path: "/admin_api/unmapped", want: fiber.StatusForbidden},
		{path: "/user_api/bills/public/BILL-001", want: fiber.StatusOK},
	} {
		response, err := app.Test(httptest.NewRequest(http.MethodGet, testCase.path, nil))
		if err != nil {
			t.Fatalf("GET %s: %v", testCase.path, err)
		}
		if response.StatusCode != testCase.want {
			t.Errorf("GET %s status = %d, want %d", testCase.path, response.StatusCode, testCase.want)
		}
	}
}

func TestCrossRoleAuthorizationOnRepresentativeEndpoints(t *testing.T) {
	t.Setenv("JWT_SECRET", "phase-zero-test-secret")
	database, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared&_foreign_keys=1"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := database.AutoMigrate(
		&models.Coverage{}, &models.Role{}, &models.Permission{}, &models.User{},
		&models.RolePermission{}, &models.UserPermissionOverride{},
	); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	rolePermissions := map[string][]string{
		"admin":    {"customers.read", "bills.read", "routers.read"},
		"loket":    {"customers.read", "bills.read"},
		"teknisi":  {"customers.read", "routers.read"},
		"customer": {"self.bills.read"},
	}
	permissions := make(map[string]models.Permission)
	for _, keys := range rolePermissions {
		for _, key := range keys {
			if _, exists := permissions[key]; exists {
				continue
			}
			permission := models.Permission{Key: key, Module: permissionModule(key), Action: "read", Name: key, RiskLevel: "low"}
			if err := database.Create(&permission).Error; err != nil {
				t.Fatalf("create permission %s: %v", key, err)
			}
			permissions[key] = permission
		}
	}
	users := make(map[string]models.User)
	for roleKey, keys := range rolePermissions {
		role := models.Role{Key: roleKey, Name: roleKey, IsSystem: true, IsActive: true}
		if err := database.Create(&role).Error; err != nil {
			t.Fatalf("create role %s: %v", roleKey, err)
		}
		for _, key := range keys {
			if err := database.Create(&models.RolePermission{RoleID: role.ID, PermissionID: permissions[key].ID}).Error; err != nil {
				t.Fatalf("assign %s to %s: %v", key, roleKey, err)
			}
		}
		roleID := role.ID
		user := models.User{
			ID: uuid.New(), Name: roleKey, Email: roleKey + "@phase-six.test",
			Role: roleKey, RoleID: &roleID, IsActive: true, PermissionVersion: 1,
		}
		if err := database.Create(&user).Error; err != nil {
			t.Fatalf("create user %s: %v", roleKey, err)
		}
		users[roleKey] = user
	}

	app := fiber.New()
	app.Use(EnforceRoutePermissions(services.NewAuthorizationService(repositories.NewAuthorizationRepository(database))))
	for _, path := range []string{
		"/admin_api/customers", "/admin_api/bills", "/admin_api/routers", "/user_api/bills",
	} {
		app.Get(path, func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	}
	testCases := []struct {
		role string
		path string
		want int
	}{
		{role: "admin", path: "/admin_api/customers", want: fiber.StatusOK},
		{role: "admin", path: "/admin_api/routers", want: fiber.StatusOK},
		{role: "loket", path: "/admin_api/bills", want: fiber.StatusOK},
		{role: "loket", path: "/admin_api/routers", want: fiber.StatusForbidden},
		{role: "teknisi", path: "/admin_api/routers", want: fiber.StatusOK},
		{role: "teknisi", path: "/admin_api/bills", want: fiber.StatusForbidden},
		{role: "customer", path: "/user_api/bills", want: fiber.StatusOK},
		{role: "customer", path: "/admin_api/customers", want: fiber.StatusForbidden},
	}
	for _, testCase := range testCases {
		t.Run(testCase.role+" "+testCase.path, func(t *testing.T) {
			claims := jwt.MapClaims{
				"user_id": users[testCase.role].ID.String(),
				"role":    testCase.role,
				"exp":     time.Now().Add(time.Hour).Unix(),
			}
			request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
			request.Header.Set("Authorization", "Bearer "+signMiddlewareToken(t, jwt.SigningMethodHS256, claims))
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			if response.StatusCode != testCase.want {
				t.Fatalf("status = %d, want %d", response.StatusCode, testCase.want)
			}
		})
	}
}
