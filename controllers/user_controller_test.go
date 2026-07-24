package controllers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPublicRegisterCannotChoosePrivilegedRole(t *testing.T) {
	app, database, _ := newUserControllerTestApp(t)
	body := []byte(`{"name":"Public User","email":"public@example.com","phone":"0812","password":"secret123","role":"owner"}`)
	request := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("register request: %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("register status = %d, want 200", response.StatusCode)
	}

	var user models.User
	if err := database.Preload("RoleDefinition").First(&user, "email = ?", "public@example.com").Error; err != nil {
		t.Fatalf("load registered user: %v", err)
	}
	if user.Role != "user" || user.RoleDefinition == nil || user.RoleDefinition.Key != "customer" {
		t.Fatalf("registered role = %q/%+v, want legacy user and canonical customer", user.Role, user.RoleDefinition)
	}
}

func TestUpdateMeIgnoresRoleAndAccessFields(t *testing.T) {
	app, database, customer := newUserControllerTestApp(t)
	body := []byte(`{"name":"Updated Customer","role":"owner","role_id":"` + uuid.NewString() + `","permission_version":999,"is_active":false}`)
	request := httptest.NewRequest("PUT", "/api/auth/me", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+signControllerToken(t, customer.ID, "user"))
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("update profile request: %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("update profile status = %d, want 200", response.StatusCode)
	}

	var reloaded models.User
	if err := database.First(&reloaded, "id = ?", customer.ID).Error; err != nil {
		t.Fatalf("reload customer: %v", err)
	}
	if reloaded.Name != "Updated Customer" {
		t.Fatalf("name = %q, want updated value", reloaded.Name)
	}
	if reloaded.Role != "user" || reloaded.RoleID == nil || *reloaded.RoleID != *customer.RoleID {
		t.Fatalf("profile update changed role: %+v", reloaded)
	}
	if !reloaded.IsActive || reloaded.PermissionVersion != 1 {
		t.Fatalf("profile update changed access fields: active=%v version=%d", reloaded.IsActive, reloaded.PermissionVersion)
	}
}

func TestGetMeReturnsCanonicalRoleAndEffectivePermissions(t *testing.T) {
	app, _, customer := newUserControllerTestApp(t)
	request := httptest.NewRequest("GET", "/api/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+signControllerToken(t, customer.ID, "user"))
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("get me request: %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	var payload struct {
		Data struct {
			Role struct {
				Key string `json:"key"`
			} `json:"role"`
			Permissions       []string `json:"permissions"`
			PermissionVersion int64    `json:"permission_version"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Role.Key != "customer" || payload.Data.PermissionVersion != 1 || payload.Data.Permissions == nil {
		t.Fatalf("unexpected auth profile: %#v", payload.Data)
	}
}

func TestUserAdministrationRequiresAdminRole(t *testing.T) {
	app, _, customer := newUserControllerTestApp(t)
	testCases := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "anonymous", wantStatus: fiber.StatusUnauthorized},
		{name: "customer", token: signControllerToken(t, customer.ID, "user"), wantStatus: fiber.StatusForbidden},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/api/users", nil)
			if testCase.token != "" {
				request.Header.Set("Authorization", "Bearer "+testCase.token)
			}
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("users request: %v", err)
			}
			if response.StatusCode != testCase.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, testCase.wantStatus)
			}
		})
	}
}

func TestAdminUserUpdateIgnoresAccessFields(t *testing.T) {
	app, database, customer := newUserControllerTestApp(t)
	admin := createControllerTestUser(t, database, "admin@example.com", "admin", "admin")
	body := []byte(`{"role":"owner"}`)
	request := httptest.NewRequest("PUT", "/api/users/"+customer.ID.String(), bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+signControllerToken(t, admin.ID, "admin"))
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("admin update request: %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("admin owner assignment status = %d, want 200 with access fields ignored", response.StatusCode)
	}

	var reloaded models.User
	if err := database.First(&reloaded, "id = ?", customer.ID).Error; err != nil {
		t.Fatalf("reload customer: %v", err)
	}
	if reloaded.Role != "user" {
		t.Fatalf("role = %q, want unchanged user", reloaded.Role)
	}
}

func TestUserAdministrationCannotDeleteOwner(t *testing.T) {
	app, database, _ := newUserControllerTestApp(t)
	admin := createControllerTestUser(t, database, "admin-delete-owner@example.com", "admin", "admin")
	owner := createControllerTestUser(t, database, "protected-owner@example.com", "owner", "owner")
	request := httptest.NewRequest("DELETE", "/api/users/"+owner.ID.String(), nil)
	request.Header.Set("Authorization", "Bearer "+signControllerToken(t, admin.ID, "admin"))
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("delete owner request: %v", err)
	}
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.StatusCode)
	}
	var count int64
	if err := database.Model(&models.User{}).Where("id = ?", owner.ID).Count(&count).Error; err != nil {
		t.Fatalf("count owner: %v", err)
	}
	if count != 1 {
		t.Fatal("owner was deleted through user administration")
	}
}

func newUserControllerTestApp(t *testing.T) (*fiber.App, *gorm.DB, models.User) {
	t.Helper()
	t.Setenv("JWT_SECRET", "controller-phase-zero-secret")
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_foreign_keys=1"
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := database.AutoMigrate(&models.Role{}, &models.Permission{}, &models.User{}, &models.RolePermission{}, &models.UserPermissionOverride{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	roles := []models.Role{
		{Key: "owner", Name: "Owner", IsSystem: true, IsOwner: true, IsActive: true},
		{Key: "admin", Name: "Admin", IsSystem: true, IsActive: true},
		{Key: "customer", Name: "Customer", IsSystem: true, IsActive: true},
	}
	for index := range roles {
		if err := database.Create(&roles[index]).Error; err != nil {
			t.Fatalf("create role %s: %v", roles[index].Key, err)
		}
	}
	for _, key := range []string{"users.read", "users.create", "users.update", "users.delete"} {
		permission := models.Permission{Key: key, Module: "users", Action: strings.TrimPrefix(key, "users."), Name: key, RiskLevel: "low"}
		if err := database.Create(&permission).Error; err != nil {
			t.Fatalf("create permission %s: %v", key, err)
		}
		if err := database.Create(&models.RolePermission{RoleID: roles[1].ID, PermissionID: permission.ID}).Error; err != nil {
			t.Fatalf("create admin role permission: %v", err)
		}
	}
	customer := createControllerTestUser(t, database, "customer@example.com", "user", "customer")

	repository := repositories.NewUserRepository(database)
	service := services.NewUserService(repository)
	authorizer := services.NewAuthorizationService(repositories.NewAuthorizationRepository(database))
	controller := NewUserController(service, authorizer)
	app := fiber.New()
	app.Use(middlewares.EnforceRoutePermissions(authorizer))
	controller.RegisterRoutes(app)
	return app, database, customer
}

func createControllerTestUser(t *testing.T, database *gorm.DB, email, legacyRole, canonicalRole string) models.User {
	t.Helper()
	var role models.Role
	if err := database.First(&role, "key = ?", canonicalRole).Error; err != nil {
		t.Fatalf("load role %s: %v", canonicalRole, err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := models.User{
		Name: email, Email: email, Password: string(hash), Role: legacyRole, RoleID: &role.ID,
		PermissionVersion: 1, IsActive: true,
	}
	if err := database.Create(&user).Error; err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return user
}

func signControllerToken(t *testing.T, userID uuid.UUID, role string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID.String(), "role": role, "exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte("controller-phase-zero-secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func decodeControllerResponse(t *testing.T, responseBody []byte) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return payload
}
