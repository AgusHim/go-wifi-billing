package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type authorizationRepositoryStub struct {
	user            *models.User
	userErr         error
	legacyRole      *models.Role
	legacyRoleErr   error
	rolePermissions []string
	roleErr         error
	overrides       []repositories.PermissionOverrideRecord
	overridesErr    error
	allPermissions  []string
	allErr          error
}

func (s *authorizationRepositoryStub) GetUserWithRole(context.Context, uuid.UUID) (*models.User, error) {
	return s.user, s.userErr
}

func (s *authorizationRepositoryStub) GetRoleByKey(context.Context, string) (*models.Role, error) {
	return s.legacyRole, s.legacyRoleErr
}

func (s *authorizationRepositoryStub) GetRolePermissionKeys(context.Context, uuid.UUID) ([]string, error) {
	return s.rolePermissions, s.roleErr
}

func (s *authorizationRepositoryStub) GetUserPermissionOverrides(context.Context, uuid.UUID) ([]repositories.PermissionOverrideRecord, error) {
	return s.overrides, s.overridesErr
}

func (s *authorizationRepositoryStub) GetAllPermissionKeys(context.Context) ([]string, error) {
	return s.allPermissions, s.allErr
}

func TestAuthorizationResolveRoleOnlyAllow(t *testing.T) {
	repository := &authorizationRepositoryStub{
		user:            authorizationTestUser(authorizationTestRole("loket", false, true)),
		rolePermissions: []string{"bills.read", "payments.create"},
	}

	decision, err := NewAuthorizationService(repository).Resolve(context.Background(), repository.user.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !decision.HasPermission("bills.read") || !decision.HasPermission("payments.create") {
		t.Fatalf("role permissions were not resolved: %#v", decision.Permissions)
	}
	if decision.Permissions["bills.read"] != PermissionSourceRole {
		t.Fatalf("permission source = %q, want %q", decision.Permissions["bills.read"], PermissionSourceRole)
	}
	if decision.HasPermission("payments.delete") {
		t.Fatal("unexpected permission was allowed")
	}
}

func TestAuthorizationResolveUserAllowOverride(t *testing.T) {
	repository := &authorizationRepositoryStub{
		user: authorizationTestUser(authorizationTestRole("loket", false, true)),
		overrides: []repositories.PermissionOverrideRecord{
			{PermissionKey: "payments.export", Effect: models.PermissionEffectAllow},
		},
	}

	decision, err := NewAuthorizationService(repository).Resolve(context.Background(), repository.user.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !decision.HasPermission("payments.export") {
		t.Fatal("user allow override was not applied")
	}
	if decision.Permissions["payments.export"] != PermissionSourceUserAllow {
		t.Fatalf("permission source = %q, want %q", decision.Permissions["payments.export"], PermissionSourceUserAllow)
	}
}

func TestAuthorizationResolveUserDenyWinsOverRoleAllow(t *testing.T) {
	repository := &authorizationRepositoryStub{
		user:            authorizationTestUser(authorizationTestRole("admin", false, true)),
		rolePermissions: []string{"customers.read", "customers.delete"},
		overrides: []repositories.PermissionOverrideRecord{
			{PermissionKey: "customers.delete", Effect: models.PermissionEffectDeny},
		},
	}

	decision, err := NewAuthorizationService(repository).Resolve(context.Background(), repository.user.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if decision.HasPermission("customers.delete") {
		t.Fatal("deny override did not remove the role permission")
	}
	if !decision.DeniedPermissions["customers.delete"] {
		t.Fatal("deny reason was not retained in the decision")
	}
	if !decision.HasPermission("customers.read") {
		t.Fatal("unrelated role permission was removed")
	}
}

func TestAuthorizationResolveOwnerAlwaysReceivesApplicationPermissions(t *testing.T) {
	repository := &authorizationRepositoryStub{
		user:           authorizationTestUser(authorizationTestRole("owner", true, true)),
		allPermissions: []string{"access_control.manage", "customers.delete"},
		overrides: []repositories.PermissionOverrideRecord{
			{PermissionKey: "customers.delete", Effect: models.PermissionEffectDeny},
		},
	}

	decision, err := NewAuthorizationService(repository).Resolve(context.Background(), repository.user.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !decision.IsOwner || !decision.HasPermission("access_control.manage") || !decision.HasPermission("customers.delete") {
		t.Fatalf("owner did not receive the catalog: %#v", decision)
	}
	if decision.Permissions["customers.delete"] != PermissionSourceOwner {
		t.Fatalf("owner source = %q, want %q", decision.Permissions["customers.delete"], PermissionSourceOwner)
	}
}

func TestAuthorizationResolveRejectsInactiveOrDeletedPrincipal(t *testing.T) {
	activeRole := authorizationTestRole("admin", false, true)
	inactiveRole := authorizationTestRole("admin", false, false)
	deletedUser := authorizationTestUser(activeRole)
	deletedUser.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	inactiveUser := authorizationTestUser(activeRole)
	inactiveUser.IsActive = false

	testCases := []struct {
		name    string
		user    *models.User
		wantErr error
	}{
		{name: "missing user", user: nil, wantErr: ErrAuthorizationUserNotFound},
		{name: "deleted user", user: deletedUser, wantErr: ErrAuthorizationUserNotFound},
		{name: "inactive user", user: inactiveUser, wantErr: ErrAuthorizationUserInactive},
		{name: "inactive role", user: authorizationTestUser(inactiveRole), wantErr: ErrAuthorizationRoleInactive},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repository := &authorizationRepositoryStub{user: testCase.user}
			_, err := NewAuthorizationService(repository).Resolve(context.Background(), uuid.New())
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestAuthorizationResolveSupportsLegacyRoleFallback(t *testing.T) {
	role := authorizationTestRole("teknisi", false, true)
	user := &models.User{ID: uuid.New(), Role: "technician", IsActive: true, PermissionVersion: 3}
	repository := &authorizationRepositoryStub{
		user:            user,
		legacyRole:      role,
		rolePermissions: []string{"noc.read"},
	}

	decision, err := NewAuthorizationService(repository).Resolve(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if decision.RoleKey != "teknisi" || !decision.HasPermission("noc.read") {
		t.Fatalf("legacy role fallback failed: %#v", decision)
	}
}

func authorizationTestRole(key string, owner, active bool) *models.Role {
	return &models.Role{ID: uuid.New(), Key: key, IsOwner: owner, IsActive: active}
}

func authorizationTestUser(role *models.Role) *models.User {
	roleID := role.ID
	return &models.User{
		ID:                uuid.New(),
		RoleID:            &roleID,
		Role:              role.Key,
		RoleDefinition:    role,
		IsActive:          true,
		PermissionVersion: 7,
	}
}
