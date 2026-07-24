package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/observability"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type accessControlTestFixture struct {
	db          *gorm.DB
	service     AccessControlService
	authorizer  AuthorizationService
	owner       models.User
	target      models.User
	ownerRole   models.Role
	adminRole   models.Role
	loketRole   models.Role
	permissions map[string]models.Permission
}

func TestAccessControlUpdateUserAccessChangesNextAuthorizationDecision(t *testing.T) {
	fixture := newAccessControlTestFixture(t)
	result, err := fixture.service.UpdateUserAccess(context.Background(), fixture.owner.ID, fixture.target.ID, dto.UpdateUserAccessDTO{
		RoleID: fixture.adminRole.ID.String(),
		Overrides: []dto.UserPermissionOverrideDTO{
			{PermissionKey: "payments.export", Effect: models.PermissionEffectAllow},
		},
		ExpectedPermissionVersion: 1,
	}, dto.AccessChangeMetadata{IPAddress: "127.0.0.1", UserAgent: "phase-three-test", RequestID: "request-1"})
	if err != nil {
		t.Fatalf("update user access: %v", err)
	}
	if result.User.PermissionVersion != 2 || result.Role.Key != "admin" {
		t.Fatalf("unexpected access result: %#v", result)
	}

	decision, err := fixture.authorizer.Resolve(context.Background(), fixture.target.ID)
	if err != nil {
		t.Fatalf("resolve updated access: %v", err)
	}
	if decision.PermissionVersion != 2 || !decision.HasPermission("customers.read") || !decision.HasPermission("payments.export") {
		t.Fatalf("updated permission was not effective on next resolve: %#v", decision)
	}

	var audit models.AccessAuditLog
	if err := fixture.db.First(&audit, "target_id = ?", fixture.target.ID).Error; err != nil {
		t.Fatalf("load audit: %v", err)
	}
	if audit.Action != "user_role_changed" || audit.RequestID != "request-1" || audit.UserAgent != "phase-three-test" {
		t.Fatalf("unexpected audit: %#v", audit)
	}
}

func TestAccessControlOwnerPromotionEmitsSecurityAlertAfterCommit(t *testing.T) {
	fixture := newAccessControlTestFixture(t)
	observability.DefaultAccessControl.Reset()
	t.Cleanup(observability.DefaultAccessControl.Reset)
	_, err := fixture.service.UpdateUserAccess(context.Background(), fixture.owner.ID, fixture.target.ID, dto.UpdateUserAccessDTO{
		RoleID: fixture.ownerRole.ID.String(), ExpectedPermissionVersion: 1, Reason: "approved second owner",
	}, dto.AccessChangeMetadata{})
	if err != nil {
		t.Fatalf("promote owner: %v", err)
	}
	alerts := observability.DefaultAccessControl.Snapshot().Alerts
	if len(alerts) != 1 || alerts[0].Type != "owner_change" || alerts[0].ActorID != fixture.owner.ID || alerts[0].TargetID != fixture.target.ID {
		t.Fatalf("unexpected owner security alert: %#v", alerts)
	}
}

func TestAccessControlUpdateUserAccessConflictLeavesNoPartialData(t *testing.T) {
	fixture := newAccessControlTestFixture(t)
	_, err := fixture.service.UpdateUserAccess(context.Background(), fixture.owner.ID, fixture.target.ID, dto.UpdateUserAccessDTO{
		RoleID: fixture.adminRole.ID.String(),
		Overrides: []dto.UserPermissionOverrideDTO{
			{PermissionKey: "payments.export", Effect: models.PermissionEffectAllow},
		},
		ExpectedPermissionVersion: 99,
	}, dto.AccessChangeMetadata{})
	if !errors.Is(err, ErrAccessControlConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
	assertAccessControlUserUnchanged(t, fixture, fixture.loketRole.ID)
	assertAccessControlAuditCount(t, fixture.db, 0)
}

func TestAccessControlCriticalOverrideRequiresReasonAndRollsBack(t *testing.T) {
	fixture := newAccessControlTestFixture(t)
	_, err := fixture.service.UpdateUserAccess(context.Background(), fixture.owner.ID, fixture.target.ID, dto.UpdateUserAccessDTO{
		RoleID: fixture.loketRole.ID.String(),
		Overrides: []dto.UserPermissionOverrideDTO{
			{PermissionKey: "customers.delete", Effect: models.PermissionEffectAllow},
		},
		ExpectedPermissionVersion: 1,
	}, dto.AccessChangeMetadata{})
	if !errors.Is(err, ErrAccessControlCriticalReason) {
		t.Fatalf("error = %v, want critical reason", err)
	}
	assertAccessControlUserUnchanged(t, fixture, fixture.loketRole.ID)
	assertAccessControlAuditCount(t, fixture.db, 0)
}

func TestAccessControlProtectsLastActiveOwner(t *testing.T) {
	fixture := newAccessControlTestFixture(t)
	_, err := fixture.service.UpdateUserAccess(context.Background(), fixture.owner.ID, fixture.owner.ID, dto.UpdateUserAccessDTO{
		RoleID: fixture.adminRole.ID.String(), ExpectedPermissionVersion: 1, Reason: "demote owner",
	}, dto.AccessChangeMetadata{})
	if !errors.Is(err, ErrAccessControlLastOwner) {
		t.Fatalf("error = %v, want last owner", err)
	}
	var owner models.User
	if err := fixture.db.Preload("RoleDefinition").First(&owner, "id = ?", fixture.owner.ID).Error; err != nil {
		t.Fatalf("reload owner: %v", err)
	}
	if owner.RoleDefinition == nil || !owner.RoleDefinition.IsOwner || owner.PermissionVersion != 1 {
		t.Fatalf("owner access changed: %#v", owner)
	}
	assertAccessControlAuditCount(t, fixture.db, 0)
}

func TestAccessControlAuditFailureRollsBackAccessMutation(t *testing.T) {
	fixture := newAccessControlTestFixture(t)
	missingActor := uuid.New()
	_, err := fixture.service.UpdateUserAccess(context.Background(), missingActor, fixture.target.ID, dto.UpdateUserAccessDTO{
		RoleID: fixture.adminRole.ID.String(), ExpectedPermissionVersion: 1,
	}, dto.AccessChangeMetadata{})
	if err == nil {
		t.Fatal("expected audit foreign-key failure")
	}
	assertAccessControlUserUnchanged(t, fixture, fixture.loketRole.ID)
	assertAccessControlAuditCount(t, fixture.db, 0)
}

func TestAccessControlRolePermissionChangeInvalidatesAssignedUsers(t *testing.T) {
	fixture := newAccessControlTestFixture(t)
	adminUser := createAccessControlTestUser(t, fixture.db, "admin-assigned@example.com", "admin", fixture.adminRole, 4)
	result, err := fixture.service.UpdateRolePermissions(context.Background(), fixture.owner.ID, fixture.adminRole.ID, dto.UpdateRolePermissionsDTO{
		PermissionKeys: []string{"customers.read", "customers.delete"}, Reason: "approve critical baseline",
	}, dto.AccessChangeMetadata{})
	if err != nil {
		t.Fatalf("update role permissions: %v", err)
	}
	if len(result.Permissions) != 2 {
		t.Fatalf("role permissions = %d, want 2", len(result.Permissions))
	}
	var reloaded models.User
	if err := fixture.db.First(&reloaded, "id = ?", adminUser.ID).Error; err != nil {
		t.Fatalf("reload admin user: %v", err)
	}
	if reloaded.PermissionVersion != 5 {
		t.Fatalf("permission version = %d, want 5", reloaded.PermissionVersion)
	}
	assertAccessControlAuditCount(t, fixture.db, 1)
}

func TestAccessControlInvalidRolePermissionChangeIsAtomic(t *testing.T) {
	fixture := newAccessControlTestFixture(t)
	_, err := fixture.service.UpdateRolePermissions(context.Background(), fixture.owner.ID, fixture.adminRole.ID, dto.UpdateRolePermissionsDTO{
		PermissionKeys: []string{"customers.read", "unknown.permission"},
	}, dto.AccessChangeMetadata{})
	if !errors.Is(err, ErrAccessControlInvalidInput) {
		t.Fatalf("error = %v, want invalid input", err)
	}
	var count int64
	if err := fixture.db.Model(&models.RolePermission{}).Where("role_id = ?", fixture.adminRole.ID).Count(&count).Error; err != nil {
		t.Fatalf("count role permissions: %v", err)
	}
	if count != 1 {
		t.Fatalf("role permission count = %d, want original 1", count)
	}
	assertAccessControlAuditCount(t, fixture.db, 0)
}

func TestAccessControlCustomRoleLifecycle(t *testing.T) {
	fixture := newAccessControlTestFixture(t)
	created, err := fixture.service.CreateRole(context.Background(), fixture.owner.ID, dto.CreateRoleDTO{
		Key: "collector", Name: "Collector", Description: "Initial role",
	}, dto.AccessChangeMetadata{})
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	if created.Role.IsSystem || created.Role.IsOwner || !created.Role.IsActive {
		t.Fatalf("custom role flags are invalid: %#v", created.Role)
	}

	updatedName := "Field Collector"
	updatedDescription := "Collects field data"
	updated, err := fixture.service.UpdateRole(context.Background(), fixture.owner.ID, created.Role.ID, dto.UpdateRoleDTO{
		Name: &updatedName, Description: &updatedDescription,
	}, dto.AccessChangeMetadata{})
	if err != nil {
		t.Fatalf("update role: %v", err)
	}
	if updated.Role.Name != updatedName || updated.Role.Description != updatedDescription {
		t.Fatalf("role metadata was not updated: %#v", updated.Role)
	}

	updated, err = fixture.service.UpdateRolePermissions(context.Background(), fixture.owner.ID, created.Role.ID, dto.UpdateRolePermissionsDTO{
		PermissionKeys: []string{"customers.read"},
	}, dto.AccessChangeMetadata{})
	if err != nil {
		t.Fatalf("update custom role permissions: %v", err)
	}
	if len(updated.Permissions) != 1 || updated.Permissions[0].Key != "customers.read" {
		t.Fatalf("custom role permissions = %#v", updated.Permissions)
	}

	roles, total, err := fixture.service.ListRoles(context.Background(), 1, 20, "collector")
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if total != 1 || len(roles) != 1 || roles[0].PermissionCount != 1 {
		t.Fatalf("role list result = %#v, total=%d", roles, total)
	}

	if err := fixture.service.DeleteRole(context.Background(), fixture.owner.ID, created.Role.ID, "role no longer required", dto.AccessChangeMetadata{}); err != nil {
		t.Fatalf("delete role: %v", err)
	}
	if _, err := fixture.service.GetRole(context.Background(), created.Role.ID); !errors.Is(err, ErrAccessControlNotFound) {
		t.Fatalf("deleted role error = %v, want not found", err)
	}
	assertAccessControlAuditCount(t, fixture.db, 4)
}

func TestAccessControlResetUserOverridesRestoresRoleDefault(t *testing.T) {
	fixture := newAccessControlTestFixture(t)
	_, err := fixture.service.UpdateUserAccess(context.Background(), fixture.owner.ID, fixture.target.ID, dto.UpdateUserAccessDTO{
		RoleID: fixture.loketRole.ID.String(),
		Overrides: []dto.UserPermissionOverrideDTO{
			{PermissionKey: "payments.export", Effect: models.PermissionEffectAllow},
		},
		ExpectedPermissionVersion: 1,
	}, dto.AccessChangeMetadata{})
	if err != nil {
		t.Fatalf("add override: %v", err)
	}
	result, err := fixture.service.ResetUserOverrides(context.Background(), fixture.owner.ID, fixture.target.ID, dto.ResetUserOverridesDTO{
		ExpectedPermissionVersion: 2,
	}, dto.AccessChangeMetadata{})
	if err != nil {
		t.Fatalf("reset overrides: %v", err)
	}
	if result.User.PermissionVersion != 3 || len(result.Overrides) != 0 {
		t.Fatalf("override reset result = %#v", result)
	}
	decision, err := fixture.authorizer.Resolve(context.Background(), fixture.target.ID)
	if err != nil {
		t.Fatalf("resolve reset access: %v", err)
	}
	if decision.HasPermission("payments.export") {
		t.Fatal("reset permission remained effective")
	}
	assertAccessControlAuditCount(t, fixture.db, 2)
}

func newAccessControlTestFixture(t *testing.T) accessControlTestFixture {
	t.Helper()
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
	loketRole := models.Role{Key: "loket", Name: "Loket", IsSystem: true, IsActive: true}
	for _, role := range []*models.Role{&ownerRole, &adminRole, &loketRole} {
		if err := database.Create(role).Error; err != nil {
			t.Fatalf("create role %s: %v", role.Key, err)
		}
	}

	permissions := map[string]models.Permission{
		"customers.read":        {Key: "customers.read", Module: "customers", Action: "read", Name: "Read customers", RiskLevel: "low"},
		"customers.delete":      {Key: "customers.delete", Module: "customers", Action: "delete", Name: "Delete customers", RiskLevel: "critical"},
		"payments.export":       {Key: "payments.export", Module: "payments", Action: "export", Name: "Export payments", RiskLevel: "medium"},
		"access_control.manage": {Key: "access_control.manage", Module: "access_control", Action: "manage", Name: "Manage access", RiskLevel: "critical"},
	}
	for key, permission := range permissions {
		permission := permission
		if err := database.Create(&permission).Error; err != nil {
			t.Fatalf("create permission %s: %v", key, err)
		}
		permissions[key] = permission
	}
	rolePermission := models.RolePermission{RoleID: adminRole.ID, PermissionID: permissions["customers.read"].ID}
	if err := database.Create(&rolePermission).Error; err != nil {
		t.Fatalf("create role permission: %v", err)
	}

	owner := createAccessControlTestUser(t, database, "owner-phase-three@example.com", "owner", ownerRole, 1)
	target := createAccessControlTestUser(t, database, "loket-phase-three@example.com", "loket", loketRole, 1)
	authorizer := NewAuthorizationService(repositories.NewAuthorizationRepository(database))
	service := NewAccessControlService(repositories.NewAccessControlRepository(database), authorizer)
	return accessControlTestFixture{
		db: database, service: service, authorizer: authorizer, owner: owner, target: target,
		ownerRole: ownerRole, adminRole: adminRole, loketRole: loketRole, permissions: permissions,
	}
}

func createAccessControlTestUser(t *testing.T, database *gorm.DB, email, legacyRole string, role models.Role, version int64) models.User {
	t.Helper()
	roleID := role.ID
	user := models.User{ID: uuid.New(), Name: email, Email: email, Role: legacyRole, RoleID: &roleID, IsActive: true, PermissionVersion: version}
	if err := database.Create(&user).Error; err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return user
}

func assertAccessControlUserUnchanged(t *testing.T, fixture accessControlTestFixture, wantRoleID uuid.UUID) {
	t.Helper()
	var user models.User
	if err := fixture.db.First(&user, "id = ?", fixture.target.ID).Error; err != nil {
		t.Fatalf("reload target: %v", err)
	}
	if user.RoleID == nil || *user.RoleID != wantRoleID || user.PermissionVersion != 1 {
		t.Fatalf("target access changed: %#v", user)
	}
	var overrideCount int64
	if err := fixture.db.Model(&models.UserPermissionOverride{}).Where("user_id = ?", fixture.target.ID).Count(&overrideCount).Error; err != nil {
		t.Fatalf("count overrides: %v", err)
	}
	if overrideCount != 0 {
		t.Fatalf("override count = %d, want 0", overrideCount)
	}
}

func assertAccessControlAuditCount(t *testing.T, database *gorm.DB, want int64) {
	t.Helper()
	var count int64
	if err := database.Model(&models.AccessAuditLog{}).Count(&count).Error; err != nil {
		t.Fatalf("count audits: %v", err)
	}
	if count != want {
		t.Fatalf("audit count = %d, want %d", count, want)
	}
}
