package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/dto"
	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/observability"
	"github.com/Agushim/go_wifi_billing/repositories"
	"github.com/google/uuid"
)

var (
	ErrAccessControlNotFound       = errors.New("access control resource not found")
	ErrAccessControlInvalidInput   = errors.New("invalid access control input")
	ErrAccessControlConflict       = errors.New("permission version conflict")
	ErrAccessControlLastOwner      = errors.New("last active owner cannot be changed")
	ErrAccessControlSystemRole     = errors.New("system role operation is not allowed")
	ErrAccessControlCriticalReason = errors.New("reason is required for critical access changes")
)

var roleKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,49}$`)

type PermissionGroup struct {
	Module      string              `json:"module"`
	Permissions []models.Permission `json:"permissions"`
}

type RoleDetail struct {
	Role        *models.Role        `json:"role"`
	Permissions []models.Permission `json:"permissions"`
	UserCount   int64               `json:"user_count"`
}

type UserOverrideDetail struct {
	Permission models.Permission `json:"permission"`
	Effect     string            `json:"effect"`
	Reason     string            `json:"reason"`
}

type UserAccessDetail struct {
	User                 repositories.AccessControlUserRecord `json:"user"`
	Role                 *models.Role                         `json:"role"`
	RolePermissions      []models.Permission                  `json:"role_permissions"`
	Overrides            []UserOverrideDetail                 `json:"overrides"`
	EffectivePermissions map[string]PermissionSource          `json:"effective_permissions"`
	DeniedPermissions    []string                             `json:"denied_permissions"`
}

type AccessControlService interface {
	ListPermissions(ctx context.Context) ([]PermissionGroup, error)
	ListRoles(ctx context.Context, page, limit int, search string) ([]repositories.AccessControlRoleRecord, int64, error)
	GetRole(ctx context.Context, roleID uuid.UUID) (*RoleDetail, error)
	CreateRole(ctx context.Context, actorID uuid.UUID, input dto.CreateRoleDTO, metadata dto.AccessChangeMetadata) (*RoleDetail, error)
	UpdateRole(ctx context.Context, actorID, roleID uuid.UUID, input dto.UpdateRoleDTO, metadata dto.AccessChangeMetadata) (*RoleDetail, error)
	UpdateRolePermissions(ctx context.Context, actorID, roleID uuid.UUID, input dto.UpdateRolePermissionsDTO, metadata dto.AccessChangeMetadata) (*RoleDetail, error)
	DeleteRole(ctx context.Context, actorID, roleID uuid.UUID, reason string, metadata dto.AccessChangeMetadata) error
	ListUsers(ctx context.Context, page, limit int, search string, roleID *uuid.UUID) ([]repositories.AccessControlUserRecord, int64, error)
	GetUserAccess(ctx context.Context, userID uuid.UUID) (*UserAccessDetail, error)
	UpdateUserAccess(ctx context.Context, actorID, userID uuid.UUID, input dto.UpdateUserAccessDTO, metadata dto.AccessChangeMetadata) (*UserAccessDetail, error)
	ResetUserOverrides(ctx context.Context, actorID, userID uuid.UUID, input dto.ResetUserOverridesDTO, metadata dto.AccessChangeMetadata) (*UserAccessDetail, error)
	ListAuditLogs(ctx context.Context, page, limit int, filter repositories.AccessAuditFilter) ([]models.AccessAuditLog, int64, error)
	ExportAuditLogs(ctx context.Context, filter repositories.AccessAuditFilter) ([]byte, error)
}

type accessControlService struct {
	repository           repositories.AccessControlRepository
	authorizationService AuthorizationService
}

func NewAccessControlService(repository repositories.AccessControlRepository, authorizationService AuthorizationService) AccessControlService {
	return &accessControlService{repository: repository, authorizationService: authorizationService}
}

func (s *accessControlService) ListPermissions(ctx context.Context) ([]PermissionGroup, error) {
	permissions, err := s.repository.ListPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permission catalog: %w", err)
	}
	groups := make([]PermissionGroup, 0)
	for _, permission := range permissions {
		if len(groups) == 0 || groups[len(groups)-1].Module != permission.Module {
			groups = append(groups, PermissionGroup{Module: permission.Module, Permissions: make([]models.Permission, 0)})
		}
		groups[len(groups)-1].Permissions = append(groups[len(groups)-1].Permissions, permission)
	}
	return groups, nil
}

func (s *accessControlService) ListRoles(ctx context.Context, page, limit int, search string) ([]repositories.AccessControlRoleRecord, int64, error) {
	return s.repository.ListRoles(ctx, page, limit, strings.TrimSpace(search))
}

func (s *accessControlService) GetRole(ctx context.Context, roleID uuid.UUID) (*RoleDetail, error) {
	role, err := s.repository.GetRoleByID(ctx, roleID, false)
	if err != nil {
		return nil, fmt.Errorf("get role: %w", err)
	}
	if role == nil {
		return nil, ErrAccessControlNotFound
	}
	permissions, err := s.repository.GetRolePermissions(ctx, role.ID)
	if err != nil {
		return nil, fmt.Errorf("get role permissions: %w", err)
	}
	userCount, err := s.repository.CountUsersByRole(ctx, role.ID)
	if err != nil {
		return nil, fmt.Errorf("count role users: %w", err)
	}
	return &RoleDetail{Role: role, Permissions: permissions, UserCount: userCount}, nil
}

func (s *accessControlService) CreateRole(ctx context.Context, actorID uuid.UUID, input dto.CreateRoleDTO, metadata dto.AccessChangeMetadata) (*RoleDetail, error) {
	key := strings.ToLower(strings.TrimSpace(input.Key))
	name := strings.TrimSpace(input.Name)
	if !roleKeyPattern.MatchString(key) || name == "" {
		return nil, fmt.Errorf("%w: role key or name is invalid", ErrAccessControlInvalidInput)
	}
	var createdID uuid.UUID
	err := s.repository.WithinTransaction(ctx, func(repository repositories.AccessControlRepository) error {
		existing, err := repository.GetRoleByKey(ctx, key)
		if err != nil {
			return err
		}
		if existing != nil {
			return fmt.Errorf("%w: role key already exists", ErrAccessControlConflict)
		}
		role := &models.Role{Key: key, Name: name, Description: strings.TrimSpace(input.Description), IsActive: true, CreatedBy: &actorID, UpdatedBy: &actorID}
		if err := repository.CreateRole(ctx, role); err != nil {
			return err
		}
		createdID = role.ID
		return createAccessAudit(repository, ctx, actorID, "role", role.ID, "role_created", nil, role, "", metadata)
	})
	if err != nil {
		return nil, err
	}
	return s.GetRole(ctx, createdID)
}

func (s *accessControlService) UpdateRole(ctx context.Context, actorID, roleID uuid.UUID, input dto.UpdateRoleDTO, metadata dto.AccessChangeMetadata) (*RoleDetail, error) {
	err := s.repository.WithinTransaction(ctx, func(repository repositories.AccessControlRepository) error {
		role, err := repository.GetRoleByID(ctx, roleID, true)
		if err != nil {
			return err
		}
		if role == nil {
			return ErrAccessControlNotFound
		}
		before := *role
		if input.Name != nil {
			name := strings.TrimSpace(*input.Name)
			if name == "" {
				return fmt.Errorf("%w: role name is required", ErrAccessControlInvalidInput)
			}
			role.Name = name
		}
		if input.Description != nil {
			role.Description = strings.TrimSpace(*input.Description)
		}
		if input.IsActive != nil {
			if role.IsOwner && !*input.IsActive {
				return ErrAccessControlLastOwner
			}
			if !*input.IsActive && strings.TrimSpace(input.Reason) == "" {
				return ErrAccessControlCriticalReason
			}
			role.IsActive = *input.IsActive
		}
		role.UpdatedBy = &actorID
		if err := repository.UpdateRole(ctx, role); err != nil {
			return err
		}
		if before.IsActive != role.IsActive {
			if err := repository.IncrementRoleUsersPermissionVersion(ctx, role.ID); err != nil {
				return err
			}
		}
		return createAccessAudit(repository, ctx, actorID, "role", role.ID, "role_updated", before, role, input.Reason, metadata)
	})
	if err != nil {
		return nil, err
	}
	return s.GetRole(ctx, roleID)
}

func (s *accessControlService) UpdateRolePermissions(ctx context.Context, actorID, roleID uuid.UUID, input dto.UpdateRolePermissionsDTO, metadata dto.AccessChangeMetadata) (*RoleDetail, error) {
	keys, err := normalizePermissionKeys(input.PermissionKeys)
	if err != nil {
		return nil, err
	}
	err = s.repository.WithinTransaction(ctx, func(repository repositories.AccessControlRepository) error {
		role, err := repository.GetRoleByID(ctx, roleID, true)
		if err != nil {
			return err
		}
		if role == nil {
			return ErrAccessControlNotFound
		}
		if role.IsOwner {
			return ErrAccessControlSystemRole
		}
		before, err := repository.GetRolePermissions(ctx, role.ID)
		if err != nil {
			return err
		}
		after, err := repository.FindPermissionsByKeys(ctx, keys)
		if err != nil {
			return err
		}
		if len(after) != len(keys) {
			return fmt.Errorf("%w: one or more permission keys are unknown", ErrAccessControlInvalidInput)
		}
		if criticalPermissionsChanged(before, after) && strings.TrimSpace(input.Reason) == "" {
			return ErrAccessControlCriticalReason
		}
		if err := repository.ReplaceRolePermissions(ctx, role.ID, after, actorID); err != nil {
			return err
		}
		if err := repository.IncrementRoleUsersPermissionVersion(ctx, role.ID); err != nil {
			return err
		}
		return createAccessAudit(repository, ctx, actorID, "role", role.ID, "role_permissions_changed", permissionKeys(before), permissionKeys(after), input.Reason, metadata)
	})
	if err != nil {
		return nil, err
	}
	return s.GetRole(ctx, roleID)
}

func (s *accessControlService) DeleteRole(ctx context.Context, actorID, roleID uuid.UUID, reason string, metadata dto.AccessChangeMetadata) error {
	return s.repository.WithinTransaction(ctx, func(repository repositories.AccessControlRepository) error {
		role, err := repository.GetRoleByID(ctx, roleID, true)
		if err != nil {
			return err
		}
		if role == nil {
			return ErrAccessControlNotFound
		}
		if role.IsSystem || role.IsOwner {
			return ErrAccessControlSystemRole
		}
		count, err := repository.CountUsersByRole(ctx, role.ID)
		if err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("%w: role is still assigned to users", ErrAccessControlConflict)
		}
		if err := repository.DeleteRole(ctx, role.ID); err != nil {
			return err
		}
		return createAccessAudit(repository, ctx, actorID, "role", role.ID, "role_deleted", role, nil, reason, metadata)
	})
}

func (s *accessControlService) ListUsers(ctx context.Context, page, limit int, search string, roleID *uuid.UUID) ([]repositories.AccessControlUserRecord, int64, error) {
	return s.repository.ListUsers(ctx, page, limit, strings.TrimSpace(search), roleID)
}

func (s *accessControlService) GetUserAccess(ctx context.Context, userID uuid.UUID) (*UserAccessDetail, error) {
	user, err := s.repository.GetUserByID(ctx, userID, false)
	if err != nil {
		return nil, fmt.Errorf("get access-control user: %w", err)
	}
	if user == nil || user.DeletedAt.Valid || user.RoleDefinition == nil {
		return nil, ErrAccessControlNotFound
	}
	rolePermissions, err := s.repository.GetRolePermissions(ctx, user.RoleDefinition.ID)
	if err != nil {
		return nil, err
	}
	overrides, err := s.repository.GetUserOverrides(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	decision, err := s.authorizationService.Resolve(ctx, user.ID)
	if err != nil {
		if !errors.Is(err, ErrAuthorizationUserInactive) && !errors.Is(err, ErrAuthorizationRoleInactive) {
			return nil, err
		}
		decision, err = s.previewEffectivePermissions(ctx, user, rolePermissions, overrides)
		if err != nil {
			return nil, err
		}
	}
	overrideDetails := make([]UserOverrideDetail, 0, len(overrides))
	for _, override := range overrides {
		if override.Permission != nil {
			overrideDetails = append(overrideDetails, UserOverrideDetail{Permission: *override.Permission, Effect: override.Effect, Reason: override.Reason})
		}
	}
	denied := make([]string, 0, len(decision.DeniedPermissions))
	for key := range decision.DeniedPermissions {
		denied = append(denied, key)
	}
	sort.Strings(denied)
	roleID := user.RoleDefinition.ID
	return &UserAccessDetail{
		User: repositories.AccessControlUserRecord{
			ID: user.ID, Name: user.Name, Email: user.Email, IsActive: user.IsActive,
			PermissionVersion: user.PermissionVersion, RoleID: &roleID, RoleKey: user.RoleDefinition.Key,
			RoleName: user.RoleDefinition.Name, RoleIsOwner: user.RoleDefinition.IsOwner,
		},
		Role: user.RoleDefinition, RolePermissions: rolePermissions, Overrides: overrideDetails,
		EffectivePermissions: decision.Permissions, DeniedPermissions: denied,
	}, nil
}

func (s *accessControlService) UpdateUserAccess(ctx context.Context, actorID, userID uuid.UUID, input dto.UpdateUserAccessDTO, metadata dto.AccessChangeMetadata) (*UserAccessDetail, error) {
	roleID, err := uuid.Parse(strings.TrimSpace(input.RoleID))
	if err != nil || input.ExpectedPermissionVersion < 1 {
		return nil, fmt.Errorf("%w: role_id or expected_permission_version is invalid", ErrAccessControlInvalidInput)
	}
	ownerMembershipChanged := false
	err = s.repository.WithinTransaction(ctx, func(repository repositories.AccessControlRepository) error {
		user, err := repository.GetUserByID(ctx, userID, true)
		if err != nil {
			return err
		}
		if user == nil || user.DeletedAt.Valid || user.RoleDefinition == nil {
			return ErrAccessControlNotFound
		}
		if user.PermissionVersion != input.ExpectedPermissionVersion {
			return ErrAccessControlConflict
		}
		role, err := repository.GetRoleByID(ctx, roleID, true)
		if err != nil {
			return err
		}
		if role == nil || !role.IsActive {
			return fmt.Errorf("%w: target role is missing or inactive", ErrAccessControlInvalidInput)
		}
		if user.RoleDefinition.IsOwner && !role.IsOwner && user.IsActive {
			if _, err := repository.GetRoleByID(ctx, user.RoleDefinition.ID, true); err != nil {
				return err
			}
			ownerCount, err := repository.CountActiveOwners(ctx)
			if err != nil {
				return err
			}
			if ownerCount <= 1 {
				return ErrAccessControlLastOwner
			}
		}
		if role.IsOwner && len(input.Overrides) > 0 {
			return fmt.Errorf("%w: owner cannot have permission overrides", ErrAccessControlInvalidInput)
		}

		beforeOverrides, err := repository.GetUserOverrides(ctx, user.ID)
		if err != nil {
			return err
		}
		overrides, _, err := buildUserOverrides(input.Overrides, user.ID, actorID, repository, ctx)
		if err != nil {
			return err
		}
		roleChanged := user.RoleDefinition.ID != role.ID
		ownerMembershipChanged = user.RoleDefinition.IsOwner != role.IsOwner
		criticalChanged := role.IsOwner != user.RoleDefinition.IsOwner || criticalOverridesChanged(beforeOverrides, overrides)
		if criticalChanged && strings.TrimSpace(input.Reason) == "" {
			return ErrAccessControlCriticalReason
		}
		for index := range overrides {
			overrides[index].Reason = strings.TrimSpace(input.Reason)
		}
		before := userAccessAuditSnapshot(user, beforeOverrides)
		updated, err := repository.ReplaceUserAccess(ctx, user, role, overrides, input.ExpectedPermissionVersion)
		if err != nil {
			return err
		}
		if !updated {
			return ErrAccessControlConflict
		}
		action := "user_overrides_changed"
		if roleChanged {
			action = "user_role_changed"
		}
		after := map[string]interface{}{"role_id": role.ID, "role_key": role.Key, "overrides": overrideAuditValues(overrides), "permission_version": input.ExpectedPermissionVersion + 1}
		return createAccessAudit(repository, ctx, actorID, "user", user.ID, action, before, after, input.Reason, metadata)
	})
	if err != nil {
		return nil, err
	}
	if ownerMembershipChanged {
		observability.DefaultAccessControl.RecordOwnerChange(actorID, userID, "user_role_changed")
	}
	return s.GetUserAccess(ctx, userID)
}

func (s *accessControlService) ResetUserOverrides(ctx context.Context, actorID, userID uuid.UUID, input dto.ResetUserOverridesDTO, metadata dto.AccessChangeMetadata) (*UserAccessDetail, error) {
	if input.ExpectedPermissionVersion < 1 {
		return nil, fmt.Errorf("%w: expected_permission_version is invalid", ErrAccessControlInvalidInput)
	}
	err := s.repository.WithinTransaction(ctx, func(repository repositories.AccessControlRepository) error {
		user, err := repository.GetUserByID(ctx, userID, true)
		if err != nil {
			return err
		}
		if user == nil || user.DeletedAt.Valid || user.RoleDefinition == nil {
			return ErrAccessControlNotFound
		}
		if user.PermissionVersion != input.ExpectedPermissionVersion {
			return ErrAccessControlConflict
		}
		beforeOverrides, err := repository.GetUserOverrides(ctx, user.ID)
		if err != nil {
			return err
		}
		if hasCriticalOverride(beforeOverrides) && strings.TrimSpace(input.Reason) == "" {
			return ErrAccessControlCriticalReason
		}
		updated, err := repository.ResetUserOverrides(ctx, user.ID, input.ExpectedPermissionVersion)
		if err != nil {
			return err
		}
		if !updated {
			return ErrAccessControlConflict
		}
		return createAccessAudit(repository, ctx, actorID, "user", user.ID, "user_access_restored", userAccessAuditSnapshot(user, beforeOverrides), map[string]interface{}{
			"role_id": user.RoleDefinition.ID, "role_key": user.RoleDefinition.Key, "overrides": []interface{}{}, "permission_version": input.ExpectedPermissionVersion + 1,
		}, input.Reason, metadata)
	})
	if err != nil {
		return nil, err
	}
	return s.GetUserAccess(ctx, userID)
}

func (s *accessControlService) ListAuditLogs(ctx context.Context, page, limit int, filter repositories.AccessAuditFilter) ([]models.AccessAuditLog, int64, error) {
	return s.repository.ListAuditLogs(ctx, page, limit, filter)
}

func (s *accessControlService) ExportAuditLogs(ctx context.Context, filter repositories.AccessAuditFilter) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{
		"created_at", "actor_user_id", "target_type", "target_id", "action",
		"reason", "request_id", "ip_address", "user_agent", "before_json", "after_json",
	}); err != nil {
		return nil, fmt.Errorf("write audit export header: %w", err)
	}

	const pageSize = 500
	for page := 1; ; page++ {
		logs, total, err := s.repository.ListAuditLogs(ctx, page, pageSize, filter)
		if err != nil {
			return nil, fmt.Errorf("list audit logs for export: %w", err)
		}
		for _, audit := range logs {
			if err := writer.Write([]string{
				audit.CreatedAt.UTC().Format(time.RFC3339Nano),
				audit.ActorUserID.String(),
				audit.TargetType,
				audit.TargetID.String(),
				audit.Action,
				audit.Reason,
				audit.RequestID,
				audit.IPAddress,
				audit.UserAgent,
				string(audit.BeforeData),
				string(audit.AfterData),
			}); err != nil {
				return nil, fmt.Errorf("write audit export row: %w", err)
			}
		}
		if len(logs) == 0 || int64(page*pageSize) >= total {
			break
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush audit export: %w", err)
	}
	return buffer.Bytes(), nil
}

func normalizePermissionKeys(keys []string) ([]string, error) {
	normalized := make([]string, 0, len(keys))
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("%w: permission key cannot be empty", ErrAccessControlInvalidInput)
		}
		if seen[key] {
			return nil, fmt.Errorf("%w: duplicate permission key %s", ErrAccessControlInvalidInput, key)
		}
		seen[key] = true
		normalized = append(normalized, key)
	}
	sort.Strings(normalized)
	return normalized, nil
}

func buildUserOverrides(inputs []dto.UserPermissionOverrideDTO, userID, actorID uuid.UUID, repository repositories.AccessControlRepository, ctx context.Context) ([]models.UserPermissionOverride, []models.Permission, error) {
	keys := make([]string, 0, len(inputs))
	effects := make(map[string]string, len(inputs))
	for _, input := range inputs {
		key := strings.TrimSpace(input.PermissionKey)
		effect := strings.ToLower(strings.TrimSpace(input.Effect))
		if key == "" || (effect != models.PermissionEffectAllow && effect != models.PermissionEffectDeny) {
			return nil, nil, fmt.Errorf("%w: invalid permission override", ErrAccessControlInvalidInput)
		}
		if _, exists := effects[key]; exists {
			return nil, nil, fmt.Errorf("%w: duplicate permission override %s", ErrAccessControlInvalidInput, key)
		}
		effects[key] = effect
		keys = append(keys, key)
	}
	permissions, err := repository.FindPermissionsByKeys(ctx, keys)
	if err != nil {
		return nil, nil, err
	}
	if len(permissions) != len(keys) {
		return nil, nil, fmt.Errorf("%w: one or more permission keys are unknown", ErrAccessControlInvalidInput)
	}
	overrides := make([]models.UserPermissionOverride, 0, len(permissions))
	for _, permission := range permissions {
		permissionCopy := permission
		overrides = append(overrides, models.UserPermissionOverride{
			UserID: userID, PermissionID: permission.ID, Effect: effects[permission.Key], CreatedBy: actorID, UpdatedBy: actorID, Permission: &permissionCopy,
		})
	}
	return overrides, permissions, nil
}

func criticalPermissionsChanged(before, after []models.Permission) bool {
	beforeKeys := make(map[string]models.Permission, len(before))
	afterKeys := make(map[string]models.Permission, len(after))
	for _, permission := range before {
		beforeKeys[permission.Key] = permission
	}
	for _, permission := range after {
		afterKeys[permission.Key] = permission
	}
	for key, permission := range beforeKeys {
		if _, exists := afterKeys[key]; !exists && permission.RiskLevel == "critical" {
			return true
		}
	}
	for key, permission := range afterKeys {
		if _, exists := beforeKeys[key]; !exists && permission.RiskLevel == "critical" {
			return true
		}
	}
	return false
}

func criticalOverridesChanged(before, after []models.UserPermissionOverride) bool {
	beforeCritical := make(map[string]bool)
	afterCritical := make(map[string]bool)
	for _, override := range before {
		if override.Permission != nil && override.Permission.RiskLevel == "critical" {
			beforeCritical[override.Permission.Key+":"+override.Effect] = true
		}
	}
	for _, override := range after {
		if override.Permission != nil && override.Permission.RiskLevel == "critical" {
			afterCritical[override.Permission.Key+":"+override.Effect] = true
		}
	}
	if len(beforeCritical) != len(afterCritical) {
		return true
	}
	for key := range beforeCritical {
		if !afterCritical[key] {
			return true
		}
	}
	return false
}

func (s *accessControlService) previewEffectivePermissions(ctx context.Context, user *models.User, rolePermissions []models.Permission, overrides []models.UserPermissionOverride) (*AuthorizationDecision, error) {
	decision := &AuthorizationDecision{
		UserID: user.ID, RoleID: user.RoleDefinition.ID, RoleKey: user.RoleDefinition.Key,
		IsOwner: user.RoleDefinition.IsOwner, PermissionVersion: user.PermissionVersion,
		Permissions: make(map[string]PermissionSource), DeniedPermissions: make(map[string]bool),
	}
	if decision.IsOwner {
		catalog, err := s.repository.ListPermissions(ctx)
		if err != nil {
			return nil, err
		}
		for _, permission := range catalog {
			decision.Permissions[permission.Key] = PermissionSourceOwner
		}
		return decision, nil
	}
	for _, permission := range rolePermissions {
		decision.Permissions[permission.Key] = PermissionSourceRole
	}
	for _, override := range overrides {
		if override.Permission == nil {
			continue
		}
		if override.Effect == models.PermissionEffectDeny {
			decision.DeniedPermissions[override.Permission.Key] = true
			continue
		}
		if override.Effect == models.PermissionEffectAllow {
			decision.Permissions[override.Permission.Key] = PermissionSourceUserAllow
		}
	}
	for key := range decision.DeniedPermissions {
		delete(decision.Permissions, key)
	}
	return decision, nil
}

func hasCriticalOverride(overrides []models.UserPermissionOverride) bool {
	for _, override := range overrides {
		if override.Permission != nil && override.Permission.RiskLevel == "critical" {
			return true
		}
	}
	return false
}

func permissionKeys(permissions []models.Permission) []string {
	keys := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		keys = append(keys, permission.Key)
	}
	sort.Strings(keys)
	return keys
}

func userAccessAuditSnapshot(user *models.User, overrides []models.UserPermissionOverride) map[string]interface{} {
	roleID := uuid.Nil
	roleKey := ""
	if user.RoleDefinition != nil {
		roleID = user.RoleDefinition.ID
		roleKey = user.RoleDefinition.Key
	}
	return map[string]interface{}{
		"role_id": roleID, "role_key": roleKey, "overrides": overrideAuditValues(overrides), "permission_version": user.PermissionVersion,
	}
}

func overrideAuditValues(overrides []models.UserPermissionOverride) []map[string]string {
	values := make([]map[string]string, 0, len(overrides))
	for _, override := range overrides {
		key := ""
		if override.Permission != nil {
			key = override.Permission.Key
		}
		values = append(values, map[string]string{"permission_key": key, "permission_id": override.PermissionID.String(), "effect": override.Effect})
	}
	return values
}

func createAccessAudit(repository repositories.AccessControlRepository, ctx context.Context, actorID uuid.UUID, targetType string, targetID uuid.UUID, action string, before, after interface{}, reason string, metadata dto.AccessChangeMetadata) error {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return err
	}
	return repository.CreateAuditLog(ctx, &models.AccessAuditLog{
		ActorUserID: actorID, TargetType: targetType, TargetID: targetID, Action: action,
		BeforeData: beforeJSON, AfterData: afterJSON, Reason: strings.TrimSpace(reason),
		IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, RequestID: metadata.RequestID, CreatedAt: time.Now(),
	})
}
