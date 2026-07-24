package repositories

import (
	"context"
	"errors"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PermissionOverrideRecord struct {
	PermissionKey string
	Effect        string
}

type AuthorizationRepository interface {
	GetUserWithRole(ctx context.Context, userID uuid.UUID) (*models.User, error)
	GetRoleByKey(ctx context.Context, key string) (*models.Role, error)
	GetRolePermissionKeys(ctx context.Context, roleID uuid.UUID) ([]string, error)
	GetUserPermissionOverrides(ctx context.Context, userID uuid.UUID) ([]PermissionOverrideRecord, error)
	GetAllPermissionKeys(ctx context.Context) ([]string, error)
}

type authorizationRepository struct {
	db *gorm.DB
}

func NewAuthorizationRepository(db *gorm.DB) AuthorizationRepository {
	return &authorizationRepository{db: db}
}

func (r *authorizationRepository) GetUserWithRole(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Unscoped().Preload("RoleDefinition").First(&user, "id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authorizationRepository) GetRoleByKey(ctx context.Context, key string) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).First(&role, "key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *authorizationRepository) GetRolePermissionKeys(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	keys := make([]string, 0)
	err := r.db.WithContext(ctx).
		Table("role_permissions").
		Select("permissions.key").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("role_permissions.role_id = ?", roleID).
		Order("permissions.key ASC").
		Scan(&keys).Error
	return keys, err
}

func (r *authorizationRepository) GetUserPermissionOverrides(ctx context.Context, userID uuid.UUID) ([]PermissionOverrideRecord, error) {
	overrides := make([]PermissionOverrideRecord, 0)
	err := r.db.WithContext(ctx).
		Table("user_permission_overrides").
		Select("permissions.key AS permission_key, user_permission_overrides.effect").
		Joins("JOIN permissions ON permissions.id = user_permission_overrides.permission_id").
		Where("user_permission_overrides.user_id = ?", userID).
		Order("permissions.key ASC").
		Scan(&overrides).Error
	return overrides, err
}

func (r *authorizationRepository) GetAllPermissionKeys(ctx context.Context) ([]string, error) {
	keys := make([]string, 0)
	err := r.db.WithContext(ctx).
		Model(&models.Permission{}).
		Order("key ASC").
		Pluck("key", &keys).Error
	return keys, err
}
