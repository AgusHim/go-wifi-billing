package controllers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/Agushim/go_wifi_billing/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCustomerComplainRoutesAreScopedToAuthenticatedUser(t *testing.T) {
	app, database, firstUser, firstCustomer, secondCustomer := newComplainControllerTestApp(t)
	firstComplain := createComplainControllerTestRecord(t, database, firstCustomer.ID, "First customer issue")
	secondComplain := createComplainControllerTestRecord(t, database, secondCustomer.ID, "Second customer issue")
	token := signControllerToken(t, firstUser.ID, "user")

	request := httptest.NewRequest("GET", "/admin_api/complains", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("list complains: %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("list status = %d, want 200", response.StatusCode)
	}
	var listPayload struct {
		Data []models.Complain `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&listPayload); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listPayload.Data) != 1 || listPayload.Data[0].ID != firstComplain.ID {
		t.Fatalf("customer list leaked another user's complains: %+v", listPayload.Data)
	}

	request = httptest.NewRequest("GET", "/admin_api/complains/"+secondComplain.ID.String(), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err = app.Test(request)
	if err != nil {
		t.Fatalf("get foreign complain: %v", err)
	}
	if response.StatusCode != fiber.StatusNotFound {
		t.Fatalf("foreign complain status = %d, want 404", response.StatusCode)
	}

	foreignCreate := []byte(`{"customer_id":"` + secondCustomer.ID.String() + `","subscription_id":"` + secondComplain.SubscriptionID.String() + `","complaint_type":"internet","description":"forged"}`)
	request = httptest.NewRequest("POST", "/admin_api/complains", bytes.NewReader(foreignCreate))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err = app.Test(request)
	if err != nil {
		t.Fatalf("create foreign complain: %v", err)
	}
	if response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("foreign create status = %d, want 403", response.StatusCode)
	}

	foreignSubscription := []byte(`{"customer_id":"` + firstCustomer.ID.String() + `","subscription_id":"` + secondComplain.SubscriptionID.String() + `","complaint_type":"internet","description":"forged subscription"}`)
	request = httptest.NewRequest("POST", "/admin_api/complains", bytes.NewReader(foreignSubscription))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err = app.Test(request)
	if err != nil {
		t.Fatalf("create complain with foreign subscription: %v", err)
	}
	if response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("foreign subscription status = %d, want 403", response.StatusCode)
	}
}

func TestCustomerCannotChangeOperationalComplainFields(t *testing.T) {
	app, database, user, customer, _ := newComplainControllerTestApp(t)
	complain := createComplainControllerTestRecord(t, database, customer.ID, "Original")
	foreignTechnician := uuid.New()
	body := []byte(`{"complaint_type":"slow","description":"Customer update","status":"closed","priority":"critical","technician_id":"` + foreignTechnician.String() + `","resolution_note":"forged"}`)
	request := httptest.NewRequest("PUT", "/admin_api/complains/"+complain.ID.String(), bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+signControllerToken(t, user.ID, "customer"))
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("update complain: %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("update status = %d, want 200", response.StatusCode)
	}

	var reloaded models.Complain
	if err := database.First(&reloaded, "id = ?", complain.ID).Error; err != nil {
		t.Fatalf("reload complain: %v", err)
	}
	if reloaded.Description != "Customer update" || reloaded.ComplaintType != "slow" {
		t.Fatalf("editable fields not updated: %+v", reloaded)
	}
	if reloaded.Status != "open" || reloaded.Priority != "normal" || reloaded.TechnicianID != nil || reloaded.ResolutionNote != "" {
		t.Fatalf("customer changed operational fields: %+v", reloaded)
	}
}

func newComplainControllerTestApp(t *testing.T) (*fiber.App, *gorm.DB, models.User, models.Customer, models.Customer) {
	t.Helper()
	t.Setenv("JWT_SECRET", "controller-phase-zero-secret")
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_foreign_keys=1"
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := database.AutoMigrate(
		&models.Role{}, &models.Permission{}, &models.User{}, &models.RolePermission{}, &models.UserPermissionOverride{},
		&models.Coverage{}, &models.Customer{}, &models.Package{}, &models.Subscription{}, &models.Complain{},
	); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	coverage := models.Coverage{Name: "Test", CodeArea: "TST"}
	if err := database.Create(&coverage).Error; err != nil {
		t.Fatalf("create coverage: %v", err)
	}
	customerRole := models.Role{Key: "customer", Name: "Customer", IsSystem: true, IsActive: true}
	if err := database.Create(&customerRole).Error; err != nil {
		t.Fatalf("create customer role: %v", err)
	}
	permission := models.Permission{Key: "self.complaints.manage", Module: "self", Action: "complaints.manage", Name: "Manage own complaints", RiskLevel: "low"}
	if err := database.Create(&permission).Error; err != nil {
		t.Fatalf("create permission: %v", err)
	}
	if err := database.Create(&models.RolePermission{RoleID: customerRole.ID, PermissionID: permission.ID}).Error; err != nil {
		t.Fatalf("create role permission: %v", err)
	}
	roleID := customerRole.ID
	firstUser := models.User{Name: "First", Email: "first@example.com", Role: "user", RoleID: &roleID, IsActive: true, PermissionVersion: 1}
	secondUser := models.User{Name: "Second", Email: "second@example.com", Role: "user", RoleID: &roleID, IsActive: true, PermissionVersion: 1}
	if err := database.Create(&firstUser).Error; err != nil {
		t.Fatalf("create first user: %v", err)
	}
	if err := database.Create(&secondUser).Error; err != nil {
		t.Fatalf("create second user: %v", err)
	}
	firstCustomer := models.Customer{UserID: firstUser.ID, CoverageID: coverage.ID, ServiceNumber: "TST-1", Status: "active"}
	secondCustomer := models.Customer{UserID: secondUser.ID, CoverageID: coverage.ID, ServiceNumber: "TST-2", Status: "active"}
	if err := database.Create(&firstCustomer).Error; err != nil {
		t.Fatalf("create first customer: %v", err)
	}
	if err := database.Create(&secondCustomer).Error; err != nil {
		t.Fatalf("create second customer: %v", err)
	}

	repository := repositories.NewComplainRepository(database)
	service := services.NewComplainService(repository)
	controller := NewComplainController(service)
	authorizer := services.NewAuthorizationService(repositories.NewAuthorizationRepository(database))
	app := fiber.New()
	app.Use(middlewares.EnforceRoutePermissions(authorizer))
	controller.RegisterRoutes(app)
	return app, database, firstUser, firstCustomer, secondCustomer
}

func createComplainControllerTestRecord(t *testing.T, database *gorm.DB, customerID uuid.UUID, description string) models.Complain {
	t.Helper()
	packageData := models.Package{Name: "Test Package", Price: 100000}
	if err := database.Create(&packageData).Error; err != nil {
		t.Fatalf("create package: %v", err)
	}
	subscription := models.Subscription{CustomerID: customerID, PackageID: packageData.ID, Status: "active"}
	if err := database.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	complain := models.Complain{
		CustomerID: customerID, SubscriptionID: subscription.ID, ComplaintType: "internet",
		Description: description, Status: "open", Priority: "normal",
	}
	if err := database.Create(&complain).Error; err != nil {
		t.Fatalf("create complain: %v", err)
	}
	return complain
}
