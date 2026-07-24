package repositories

import (
	"context"
	"testing"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuthorizationRepositoryLoadsPrincipalAndPermissionInputs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:authorization_repository?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Coverage{},
		&models.Role{},
		&models.Permission{},
		&models.User{},
		&models.RolePermission{},
		&models.UserPermissionOverride{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	role := models.Role{Key: "admin", Name: "Admin", IsSystem: true, IsActive: true}
	permissionFromRole := models.Permission{Key: "customers.read", Module: "customers", Action: "read", Name: "Read customers", RiskLevel: "low"}
	permissionFromOverride := models.Permission{Key: "payments.export", Module: "payments", Action: "export", Name: "Export payments", RiskLevel: "medium"}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := db.Create([]*models.Permission{&permissionFromRole, &permissionFromOverride}).Error; err != nil {
		t.Fatalf("create permissions: %v", err)
	}
	roleID := role.ID
	user := models.User{
		ID:                uuid.New(),
		RoleID:            &roleID,
		Name:              "Authorization User",
		Email:             "authorization-repository@example.com",
		Role:              "admin",
		IsActive:          true,
		PermissionVersion: 9,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: permissionFromRole.ID}).Error; err != nil {
		t.Fatalf("create role permission: %v", err)
	}
	if err := db.Create(&models.UserPermissionOverride{
		UserID:       user.ID,
		PermissionID: permissionFromOverride.ID,
		Effect:       models.PermissionEffectAllow,
		CreatedBy:    user.ID,
		UpdatedBy:    user.ID,
	}).Error; err != nil {
		t.Fatalf("create user override: %v", err)
	}

	repository := NewAuthorizationRepository(db)
	loadedUser, err := repository.GetUserWithRole(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("load user: %v", err)
	}
	if loadedUser == nil || loadedUser.RoleDefinition == nil || loadedUser.RoleDefinition.Key != "admin" {
		t.Fatalf("role was not preloaded: %#v", loadedUser)
	}

	roleKeys, err := repository.GetRolePermissionKeys(context.Background(), role.ID)
	if err != nil {
		t.Fatalf("load role permissions: %v", err)
	}
	if len(roleKeys) != 1 || roleKeys[0] != "customers.read" {
		t.Fatalf("role keys = %#v", roleKeys)
	}

	overrides, err := repository.GetUserPermissionOverrides(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("load overrides: %v", err)
	}
	if len(overrides) != 1 || overrides[0].PermissionKey != "payments.export" || overrides[0].Effect != models.PermissionEffectAllow {
		t.Fatalf("overrides = %#v", overrides)
	}

	allKeys, err := repository.GetAllPermissionKeys(context.Background())
	if err != nil {
		t.Fatalf("load permission catalog: %v", err)
	}
	if len(allKeys) != 2 || allKeys[0] != "customers.read" || allKeys[1] != "payments.export" {
		t.Fatalf("catalog keys = %#v", allKeys)
	}
}

func TestAuthorizationRepositoryIncludesSoftDeletedUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:authorization_repository_deleted?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&models.Coverage{}, &models.Role{}, &models.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	role := models.Role{Key: "customer", Name: "Customer", IsSystem: true, IsActive: true}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	roleID := role.ID
	user := models.User{ID: uuid.New(), RoleID: &roleID, Name: "Deleted User", Email: "deleted-authorization@example.com", Role: "customer", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Delete(&user).Error; err != nil {
		t.Fatalf("delete user: %v", err)
	}

	loadedUser, err := NewAuthorizationRepository(db).GetUserWithRole(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("load deleted user: %v", err)
	}
	if loadedUser == nil || !loadedUser.DeletedAt.Valid {
		t.Fatalf("soft-deleted principal must remain visible to resolver: %#v", loadedUser)
	}
}
