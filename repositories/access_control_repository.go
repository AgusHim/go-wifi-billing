package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AccessControlRoleRecord struct {
	models.Role
	UserCount       int64 `json:"user_count"`
	PermissionCount int64 `json:"permission_count"`
}

type AccessControlUserRecord struct {
	ID                uuid.UUID  `json:"id"`
	Name              string     `json:"name"`
	Email             string     `json:"email"`
	IsActive          bool       `json:"is_active"`
	PermissionVersion int64      `json:"permission_version"`
	RoleID            *uuid.UUID `json:"role_id,omitempty"`
	RoleKey           string     `json:"role_key"`
	RoleName          string     `json:"role_name"`
	RoleIsOwner       bool       `json:"role_is_owner"`
}

type AccessAuditFilter struct {
	ActorUserID *uuid.UUID
	TargetType  string
	TargetID    *uuid.UUID
	Action      string
	DateFrom    *time.Time
	DateTo      *time.Time
}

type AccessControlRepository interface {
	WithinTransaction(ctx context.Context, fn func(AccessControlRepository) error) error
	ListPermissions(ctx context.Context) ([]models.Permission, error)
	FindPermissionsByKeys(ctx context.Context, keys []string) ([]models.Permission, error)
	ListRoles(ctx context.Context, page, limit int, search string) ([]AccessControlRoleRecord, int64, error)
	GetRoleByID(ctx context.Context, roleID uuid.UUID, forUpdate bool) (*models.Role, error)
	GetRoleByKey(ctx context.Context, key string) (*models.Role, error)
	GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]models.Permission, error)
	CreateRole(ctx context.Context, role *models.Role) error
	UpdateRole(ctx context.Context, role *models.Role) error
	DeleteRole(ctx context.Context, roleID uuid.UUID) error
	CountUsersByRole(ctx context.Context, roleID uuid.UUID) (int64, error)
	ReplaceRolePermissions(ctx context.Context, roleID uuid.UUID, permissions []models.Permission, actorID uuid.UUID) error
	IncrementRoleUsersPermissionVersion(ctx context.Context, roleID uuid.UUID) error
	ListUsers(ctx context.Context, page, limit int, search string, roleID *uuid.UUID) ([]AccessControlUserRecord, int64, error)
	GetUserByID(ctx context.Context, userID uuid.UUID, forUpdate bool) (*models.User, error)
	GetUserOverrides(ctx context.Context, userID uuid.UUID) ([]models.UserPermissionOverride, error)
	ReplaceUserAccess(ctx context.Context, user *models.User, role *models.Role, overrides []models.UserPermissionOverride, expectedVersion int64) (bool, error)
	ResetUserOverrides(ctx context.Context, userID uuid.UUID, expectedVersion int64) (bool, error)
	CountActiveOwners(ctx context.Context) (int64, error)
	CreateAuditLog(ctx context.Context, audit *models.AccessAuditLog) error
	ListAuditLogs(ctx context.Context, page, limit int, filter AccessAuditFilter) ([]models.AccessAuditLog, int64, error)
}

type accessControlRepository struct {
	db *gorm.DB
}

func NewAccessControlRepository(db *gorm.DB) AccessControlRepository {
	return &accessControlRepository{db: db}
}

func (r *accessControlRepository) WithinTransaction(ctx context.Context, fn func(AccessControlRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&accessControlRepository{db: tx})
	})
}

func (r *accessControlRepository) ListPermissions(ctx context.Context) ([]models.Permission, error) {
	var permissions []models.Permission
	err := r.db.WithContext(ctx).Order("module ASC, sort_order ASC, key ASC").Find(&permissions).Error
	return permissions, err
}

func (r *accessControlRepository) FindPermissionsByKeys(ctx context.Context, keys []string) ([]models.Permission, error) {
	permissions := make([]models.Permission, 0)
	if len(keys) == 0 {
		return permissions, nil
	}
	err := r.db.WithContext(ctx).Where("key IN ?", keys).Order("key ASC").Find(&permissions).Error
	return permissions, err
}

func (r *accessControlRepository) ListRoles(ctx context.Context, page, limit int, search string) ([]AccessControlRoleRecord, int64, error) {
	var total int64
	base := r.db.WithContext(ctx).Model(&models.Role{})
	if search != "" {
		pattern := "%" + search + "%"
		base = base.Where("LOWER(roles.key) LIKE LOWER(?) OR LOWER(roles.name) LIKE LOWER(?)", pattern, pattern)
	}
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var records []AccessControlRoleRecord
	err := base.
		Select("roles.*, COUNT(DISTINCT users.id) AS user_count, COUNT(DISTINCT role_permissions.permission_id) AS permission_count").
		Joins("LEFT JOIN users ON users.role_id = roles.id AND users.deleted_at IS NULL").
		Joins("LEFT JOIN role_permissions ON role_permissions.role_id = roles.id").
		Group("roles.id").
		Order("roles.is_owner DESC, roles.is_system DESC, roles.name ASC").
		Offset((page - 1) * limit).
		Limit(limit).
		Scan(&records).Error
	return records, total, err
}

func (r *accessControlRepository) GetRoleByID(ctx context.Context, roleID uuid.UUID, forUpdate bool) (*models.Role, error) {
	query := r.db.WithContext(ctx)
	if forUpdate {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var role models.Role
	err := query.First(&role, "id = ?", roleID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &role, err
}

func (r *accessControlRepository) GetRoleByKey(ctx context.Context, key string) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).First(&role, "key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &role, err
}

func (r *accessControlRepository) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]models.Permission, error) {
	var permissions []models.Permission
	err := r.db.WithContext(ctx).
		Table("permissions").
		Select("permissions.*").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Order("permissions.module ASC, permissions.sort_order ASC, permissions.key ASC").
		Scan(&permissions).Error
	return permissions, err
}

func (r *accessControlRepository) CreateRole(ctx context.Context, role *models.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *accessControlRepository) UpdateRole(ctx context.Context, role *models.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

func (r *accessControlRepository) DeleteRole(ctx context.Context, roleID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.Role{}, "id = ?", roleID).Error
}

func (r *accessControlRepository) CountUsersByRole(ctx context.Context, roleID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Where("role_id = ?", roleID).Count(&count).Error
	return count, err
}

func (r *accessControlRepository) ReplaceRolePermissions(ctx context.Context, roleID uuid.UUID, permissions []models.Permission, actorID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
		return err
	}
	if len(permissions) == 0 {
		return nil
	}
	rows := make([]models.RolePermission, 0, len(permissions))
	for _, permission := range permissions {
		createdBy := actorID
		rows = append(rows, models.RolePermission{RoleID: roleID, PermissionID: permission.ID, CreatedBy: &createdBy})
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}

func (r *accessControlRepository) IncrementRoleUsersPermissionVersion(ctx context.Context, roleID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.User{}).
		Where("role_id = ?", roleID).
		UpdateColumn("permission_version", gorm.Expr("permission_version + 1")).Error
}

func (r *accessControlRepository) ListUsers(ctx context.Context, page, limit int, search string, roleID *uuid.UUID) ([]AccessControlUserRecord, int64, error) {
	base := r.db.WithContext(ctx).Model(&models.User{}).
		Joins("LEFT JOIN roles ON roles.id = users.role_id")
	if search != "" {
		pattern := "%" + search + "%"
		base = base.Where("LOWER(users.name) LIKE LOWER(?) OR LOWER(users.email) LIKE LOWER(?)", pattern, pattern)
	}
	if roleID != nil {
		base = base.Where("users.role_id = ?", *roleID)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []AccessControlUserRecord
	err := base.Select(
		"users.id, users.name, users.email, users.is_active, users.permission_version, users.role_id, " +
			"roles.key AS role_key, roles.name AS role_name, COALESCE(roles.is_owner, false) AS role_is_owner",
	).
		Order("users.name ASC").
		Offset((page - 1) * limit).
		Limit(limit).
		Scan(&records).Error
	return records, total, err
}

func (r *accessControlRepository) GetUserByID(ctx context.Context, userID uuid.UUID, forUpdate bool) (*models.User, error) {
	query := r.db.WithContext(ctx).Unscoped().Preload("RoleDefinition")
	if forUpdate {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var user models.User
	err := query.First(&user, "id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *accessControlRepository) GetUserOverrides(ctx context.Context, userID uuid.UUID) ([]models.UserPermissionOverride, error) {
	var overrides []models.UserPermissionOverride
	err := r.db.WithContext(ctx).Preload("Permission").Where("user_id = ?", userID).Find(&overrides).Error
	return overrides, err
}

func (r *accessControlRepository) ReplaceUserAccess(ctx context.Context, user *models.User, role *models.Role, overrides []models.UserPermissionOverride, expectedVersion int64) (bool, error) {
	result := r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ? AND permission_version = ?", user.ID, expectedVersion).
		Updates(map[string]interface{}{
			"role_id":            role.ID,
			"role":               role.Key,
			"permission_version": gorm.Expr("permission_version + 1"),
		})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected != 1 {
		return false, nil
	}
	if err := r.db.WithContext(ctx).Where("user_id = ?", user.ID).Delete(&models.UserPermissionOverride{}).Error; err != nil {
		return false, err
	}
	if len(overrides) > 0 {
		if err := r.db.WithContext(ctx).Create(&overrides).Error; err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *accessControlRepository) ResetUserOverrides(ctx context.Context, userID uuid.UUID, expectedVersion int64) (bool, error) {
	result := r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ? AND permission_version = ?", userID, expectedVersion).
		UpdateColumn("permission_version", gorm.Expr("permission_version + 1"))
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected != 1 {
		return false, nil
	}
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.UserPermissionOverride{}).Error; err != nil {
		return false, err
	}
	return true, nil
}

func (r *accessControlRepository) CountActiveOwners(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).
		Joins("JOIN roles ON roles.id = users.role_id").
		Where("users.is_active = ? AND roles.is_owner = ? AND roles.is_active = ?", true, true, true).
		Count(&count).Error
	return count, err
}

func (r *accessControlRepository) CreateAuditLog(ctx context.Context, audit *models.AccessAuditLog) error {
	return r.db.WithContext(ctx).Create(audit).Error
}

func (r *accessControlRepository) ListAuditLogs(ctx context.Context, page, limit int, filter AccessAuditFilter) ([]models.AccessAuditLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.AccessAuditLog{})
	if filter.ActorUserID != nil {
		query = query.Where("actor_user_id = ?", *filter.ActorUserID)
	}
	if filter.TargetType != "" {
		query = query.Where("target_type = ?", filter.TargetType)
	}
	if filter.TargetID != nil {
		query = query.Where("target_id = ?", *filter.TargetID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.DateFrom != nil {
		query = query.Where("created_at >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("created_at <= ?", *filter.DateTo)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var logs []models.AccessAuditLog
	err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&logs).Error
	return logs, total, err
}
