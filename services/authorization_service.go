package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

var (
	ErrAuthorizationUserNotFound = errors.New("authorization user not found")
	ErrAuthorizationUserInactive = errors.New("authorization user inactive")
	ErrAuthorizationRoleMissing  = errors.New("authorization role missing")
	ErrAuthorizationRoleInactive = errors.New("authorization role inactive")
)

type PermissionSource string

const (
	PermissionSourceOwner     PermissionSource = "owner"
	PermissionSourceRole      PermissionSource = "role"
	PermissionSourceUserAllow PermissionSource = "user_allow"
)

type AuthorizationDecision struct {
	UserID              uuid.UUID
	RoleID              uuid.UUID
	RoleKey             string
	IsOwner             bool
	PermissionVersion   int64
	Permissions         map[string]PermissionSource
	BaselinePermissions map[string]bool
	DeniedPermissions   map[string]bool
}

func (d *AuthorizationDecision) HasPermission(key string) bool {
	if d == nil {
		return false
	}
	_, allowed := d.Permissions[key]
	return allowed
}

type AuthorizationService interface {
	Resolve(ctx context.Context, userID uuid.UUID) (*AuthorizationDecision, error)
}

type authorizationService struct {
	repository repositories.AuthorizationRepository
}

func NewAuthorizationService(repository repositories.AuthorizationRepository) AuthorizationService {
	return &authorizationService{repository: repository}
}

// Resolve deliberately reads from the database on every authorization check.
// This makes permission_version observable and ensures an owner's changes take
// effect on the next request without relying on a cache invalidation window.
func (s *authorizationService) Resolve(ctx context.Context, userID uuid.UUID) (*AuthorizationDecision, error) {
	user, err := s.repository.GetUserWithRole(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load authorization user: %w", err)
	}
	if user == nil || user.DeletedAt.Valid {
		return nil, ErrAuthorizationUserNotFound
	}
	if !user.IsActive {
		return nil, ErrAuthorizationUserInactive
	}

	role, err := s.resolveRole(ctx, user)
	if err != nil {
		return nil, err
	}
	if !role.IsActive {
		return nil, ErrAuthorizationRoleInactive
	}

	decision := &AuthorizationDecision{
		UserID:              user.ID,
		RoleID:              role.ID,
		RoleKey:             role.Key,
		IsOwner:             role.IsOwner,
		PermissionVersion:   user.PermissionVersion,
		Permissions:         make(map[string]PermissionSource),
		BaselinePermissions: make(map[string]bool),
		DeniedPermissions:   make(map[string]bool),
	}

	if role.IsOwner {
		keys, err := s.repository.GetAllPermissionKeys(ctx)
		if err != nil {
			return nil, fmt.Errorf("load owner permissions: %w", err)
		}
		for _, key := range keys {
			decision.Permissions[key] = PermissionSourceOwner
			decision.BaselinePermissions[key] = true
		}
		return decision, nil
	}

	rolePermissions, err := s.repository.GetRolePermissionKeys(ctx, role.ID)
	if err != nil {
		return nil, fmt.Errorf("load role permissions: %w", err)
	}
	for _, key := range rolePermissions {
		decision.Permissions[key] = PermissionSourceRole
		decision.BaselinePermissions[key] = true
	}

	overrides, err := s.repository.GetUserPermissionOverrides(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("load user permission overrides: %w", err)
	}
	for _, override := range overrides {
		if override.Effect == models.PermissionEffectDeny {
			decision.DeniedPermissions[override.PermissionKey] = true
			continue
		}
		if override.Effect == models.PermissionEffectAllow {
			decision.Permissions[override.PermissionKey] = PermissionSourceUserAllow
		}
	}
	for key := range decision.DeniedPermissions {
		delete(decision.Permissions, key)
	}

	return decision, nil
}

func (s *authorizationService) resolveRole(ctx context.Context, user *models.User) (*models.Role, error) {
	if user.RoleID != nil {
		if user.RoleDefinition == nil || user.RoleDefinition.ID != *user.RoleID {
			return nil, ErrAuthorizationRoleMissing
		}
		return user.RoleDefinition, nil
	}

	roleKey, known := models.CanonicalRoleKey(user.Role)
	if !known {
		return nil, ErrAuthorizationRoleMissing
	}
	role, err := s.repository.GetRoleByKey(ctx, roleKey)
	if err != nil {
		return nil, fmt.Errorf("load authorization role: %w", err)
	}
	if role == nil {
		return nil, ErrAuthorizationRoleMissing
	}
	return role, nil
}
