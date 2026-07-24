package seed

import (
	"strings"
	"testing"

	middlewares "github.com/Agushim/go_wifi_billing/midlewares"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRoutePermissionCatalogCoverage(t *testing.T) {
	catalog := make(map[string]bool, len(permissionCatalog))
	for _, permission := range permissionCatalog {
		catalog[permission.Key] = true
	}
	for _, policy := range middlewares.RoutePermissionPolicies {
		keys := append([]string{}, policy.AnyPermissions...)
		if policy.Permission != "" && policy.Permission != "__public__" {
			keys = append(keys, policy.Permission)
		}
		for _, key := range keys {
			if !catalog[key] {
				t.Errorf("route %s %s references permission missing from catalog: %s", policy.Method, policy.Path, key)
			}
			if strings.TrimSpace(key) != key {
				t.Errorf("route permission contains whitespace: %q", key)
			}
		}
	}
}

func TestSeedAccessControlIsIdempotentAndBootstrapsExplicitOwner(t *testing.T) {
	database := newAccessControlTestDB(t)
	owner := createAccessControlTestUser(t, database, "owner@example.com", "admin")
	createAccessControlTestUser(t, database, "customer@example.com", "user")

	if err := SeedAccessControl(database, owner.Email); err != nil {
		t.Fatalf("first seed: %v", err)
	}

	var firstRolePermissionCount int64
	database.Model(&models.RolePermission{}).Count(&firstRolePermissionCount)
	if err := SeedAccessControl(database, owner.Email); err != nil {
		t.Fatalf("second seed: %v", err)
	}

	assertAccessControlFoundationCounts(t, database, firstRolePermissionCount)

	var reloadedOwner models.User
	if err := database.First(&reloadedOwner, "id = ?", owner.ID).Error; err != nil {
		t.Fatalf("reload owner: %v", err)
	}
	if reloadedOwner.Role != "admin" {
		t.Fatalf("legacy role changed = %q, want admin compatibility value", reloadedOwner.Role)
	}
	if reloadedOwner.RoleID == nil {
		t.Fatal("owner role_id was not backfilled")
	}
	var ownerRole models.Role
	if err := database.First(&ownerRole, "id = ?", *reloadedOwner.RoleID).Error; err != nil {
		t.Fatalf("load owner role: %v", err)
	}
	if !ownerRole.IsOwner || ownerRole.Key != "owner" {
		t.Fatalf("bootstrap role = %+v, want owner", ownerRole)
	}
	if reloadedOwner.PermissionVersion != 1 {
		t.Fatalf("permission_version = %d, want 1", reloadedOwner.PermissionVersion)
	}
}

func TestSeedAccessControlBackfillsLegacyAliases(t *testing.T) {
	database := newAccessControlTestDB(t)
	testCases := []struct {
		email string
		role  string
		want  string
	}{
		{email: "root@example.com", role: "root", want: "owner"},
		{email: "tech@example.com", role: "technician", want: "teknisi"},
		{email: "noc@example.com", role: "noc", want: "teknisi"},
		{email: "user@example.com", role: "user", want: "customer"},
	}
	for _, testCase := range testCases {
		createAccessControlTestUser(t, database, testCase.email, testCase.role)
	}

	if err := SeedAccessControl(database, ""); err != nil {
		t.Fatalf("seed aliases: %v", err)
	}
	for _, testCase := range testCases {
		var user models.User
		if err := database.Preload("RoleDefinition").First(&user, "email = ?", testCase.email).Error; err != nil {
			t.Fatalf("load %s: %v", testCase.email, err)
		}
		if user.RoleDefinition == nil || user.RoleDefinition.Key != testCase.want {
			t.Fatalf("%s canonical role = %+v, want %s", testCase.email, user.RoleDefinition, testCase.want)
		}
		if user.Role != testCase.role {
			t.Fatalf("%s legacy role changed = %s, want %s", testCase.email, user.Role, testCase.role)
		}
	}
}

func TestSeedAccessControlFailsClosedWithoutOwner(t *testing.T) {
	database := newAccessControlTestDB(t)
	createAccessControlTestUser(t, database, "admin@example.com", "admin")

	err := SeedAccessControl(database, "")
	if err == nil || !strings.Contains(err.Error(), "INITIAL_OWNER_EMAIL is required") {
		t.Fatalf("error = %v, want missing INITIAL_OWNER_EMAIL error", err)
	}

	var roleCount int64
	if err := database.Model(&models.Role{}).Count(&roleCount).Error; err != nil {
		t.Fatalf("count roles after rollback: %v", err)
	}
	if roleCount != 0 {
		t.Fatalf("transaction left %d roles after failed bootstrap, want 0", roleCount)
	}
}

func TestSeedAccessControlRejectsUnknownLegacyRole(t *testing.T) {
	database := newAccessControlTestDB(t)
	createAccessControlTestUser(t, database, "owner@example.com", "root")
	createAccessControlTestUser(t, database, "unknown@example.com", "supervisor")

	err := SeedAccessControl(database, "")
	if err == nil || !strings.Contains(err.Error(), "unknown role") {
		t.Fatalf("error = %v, want unknown role error", err)
	}
}

func TestAccessControlConstraintsAndIndexes(t *testing.T) {
	database := newAccessControlTestDB(t)
	owner := createAccessControlTestUser(t, database, "owner@example.com", "root")
	if err := SeedAccessControl(database, ""); err != nil {
		t.Fatalf("seed: %v", err)
	}

	secondOwnerRole := models.Role{Key: "other-owner", Name: "Other owner", IsSystem: true, IsOwner: true, IsActive: true}
	if err := database.Create(&secondOwnerRole).Error; err == nil {
		t.Fatal("second owner role was accepted; want unique owner constraint error")
	}

	var permission models.Permission
	if err := database.First(&permission).Error; err != nil {
		t.Fatalf("load permission: %v", err)
	}
	invalidOverride := models.UserPermissionOverride{
		UserID: owner.ID, PermissionID: permission.ID, Effect: "invalid", CreatedBy: owner.ID, UpdatedBy: owner.ID,
	}
	if err := database.Create(&invalidOverride).Error; err == nil {
		t.Fatal("invalid permission effect was accepted; want check constraint error")
	}

	indexes := []struct {
		model any
		name  string
	}{
		{model: &models.Role{}, name: "idx_roles_single_owner"},
		{model: &models.Permission{}, name: "idx_permissions_module_sort"},
		{model: &models.RolePermission{}, name: "idx_role_permissions_permission_role"},
		{model: &models.UserPermissionOverride{}, name: "idx_user_permission_overrides_permission_user"},
	}
	for _, index := range indexes {
		if !database.Migrator().HasIndex(index.model, index.name) {
			t.Errorf("missing index %s", index.name)
		}
	}
}

func TestAuditLegacyUserRolesIsReadOnlyAndReportsBlockers(t *testing.T) {
	database := newAccessControlTestDB(t)
	createAccessControlTestUser(t, database, "admin@example.com", "admin")
	createAccessControlTestUser(t, database, "unknown@example.com", "supervisor")

	report, err := AuditLegacyUserRoles(database, "admin@example.com")
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if len(report.UnknownRoleUsers) != 1 || report.UnknownRoleUsers[0].Email != "unknown@example.com" {
		t.Fatalf("unknown users = %+v", report.UnknownRoleUsers)
	}
	if len(report.OwnerCandidates) != 1 || report.OwnerCandidates[0].Email != "admin@example.com" {
		t.Fatalf("owner candidates = %+v", report.OwnerCandidates)
	}

	var mapped int64
	if err := database.Model(&models.User{}).Where("role_id IS NOT NULL").Count(&mapped).Error; err != nil {
		t.Fatalf("count mapped users: %v", err)
	}
	if mapped != 0 {
		t.Fatalf("audit mutated %d users", mapped)
	}
}

func newAccessControlTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_foreign_keys=1"
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := database.AutoMigrate(
		&models.Coverage{},
		&models.Role{},
		&models.Permission{},
		&models.User{},
		&models.RolePermission{},
		&models.UserPermissionOverride{},
		&models.AccessAuditLog{},
	); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	return database
}

func createAccessControlTestUser(t *testing.T, database *gorm.DB, email, role string) models.User {
	t.Helper()
	user := models.User{Name: email, Email: email, Role: role, IsActive: true}
	if err := database.Create(&user).Error; err != nil {
		t.Fatalf("create user %s: %v", email, err)
	}
	return user
}

func assertAccessControlFoundationCounts(t *testing.T, database *gorm.DB, wantRolePermissions int64) {
	t.Helper()
	var roleCount, permissionCount, rolePermissionCount, ownerRoleCount, activeOwnerCount, unmappedCount int64
	database.Model(&models.Role{}).Count(&roleCount)
	database.Model(&models.Permission{}).Count(&permissionCount)
	database.Model(&models.RolePermission{}).Count(&rolePermissionCount)
	database.Model(&models.Role{}).Where("is_owner = ?", true).Count(&ownerRoleCount)
	database.Table("users").Joins("JOIN roles ON roles.id = users.role_id").
		Where("roles.is_owner = ? AND users.is_active = ? AND users.deleted_at IS NULL", true, true).Count(&activeOwnerCount)
	database.Unscoped().Model(&models.User{}).Where("role_id IS NULL").Count(&unmappedCount)

	if roleCount != int64(len(canonicalRoles)) {
		t.Errorf("roles = %d, want %d", roleCount, len(canonicalRoles))
	}
	if permissionCount != int64(len(permissionCatalog)) {
		t.Errorf("permissions = %d, want %d", permissionCount, len(permissionCatalog))
	}
	if rolePermissionCount != wantRolePermissions {
		t.Errorf("role permissions = %d, want idempotent count %d", rolePermissionCount, wantRolePermissions)
	}
	if ownerRoleCount != 1 {
		t.Errorf("owner roles = %d, want 1", ownerRoleCount)
	}
	if activeOwnerCount < 1 {
		t.Errorf("active owners = %d, want at least 1", activeOwnerCount)
	}
	if unmappedCount != 0 {
		t.Errorf("unmapped users = %d, want 0", unmappedCount)
	}
}
