package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/Agushim/go_wifi_billing/observability"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrOwnerRecoveryInvalidInput = errors.New("invalid owner recovery input")
	ErrOwnerRecoveryNotRequired  = errors.New("owner recovery is blocked while an active owner exists")
)

type OwnerRecoveryRequest struct {
	TargetEmail   string
	OperatorEmail string
	Reason        string
}

type OwnerRecoveryResult struct {
	UserID            uuid.UUID `json:"user_id"`
	OperatorUserID    uuid.UUID `json:"operator_user_id"`
	Email             string    `json:"email"`
	PreviousRole      string    `json:"previous_role"`
	PermissionVersion int64     `json:"permission_version"`
	AuditLogID        uuid.UUID `json:"audit_log_id"`
}

// RecoverOwner is an emergency-only recovery path. It is intentionally kept
// outside HTTP routing and refuses to run while an active owner still exists.
func RecoverOwner(ctx context.Context, database *gorm.DB, request OwnerRecoveryRequest) (*OwnerRecoveryResult, error) {
	if database == nil {
		return nil, fmt.Errorf("%w: database is required", ErrOwnerRecoveryInvalidInput)
	}
	targetEmail := strings.ToLower(strings.TrimSpace(request.TargetEmail))
	operatorEmail := strings.ToLower(strings.TrimSpace(request.OperatorEmail))
	reason := strings.TrimSpace(request.Reason)
	if operatorEmail == "" {
		operatorEmail = targetEmail
	}
	if targetEmail == "" || operatorEmail == "" || reason == "" {
		return nil, fmt.Errorf("%w: target email, operator email, and reason are required", ErrOwnerRecoveryInvalidInput)
	}

	var result OwnerRecoveryResult
	err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ownerRole models.Role
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("is_owner = ? AND is_active = ?", true, true).
			First(&ownerRole).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: active owner role is missing", ErrOwnerRecoveryInvalidInput)
			}
			return fmt.Errorf("load owner role: %w", err)
		}

		// Locking the single owner role serializes concurrent recovery attempts.
		var activeOwners int64
		if err := tx.Model(&models.User{}).
			Joins("JOIN roles ON roles.id = users.role_id").
			Where("users.is_active = ? AND roles.is_active = ? AND roles.is_owner = ?", true, true, true).
			Count(&activeOwners).Error; err != nil {
			return fmt.Errorf("count active owners: %w", err)
		}
		if activeOwners > 0 {
			return ErrOwnerRecoveryNotRequired
		}

		var target models.User
		if err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("LOWER(email) = ?", targetEmail).
			First(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: target user not found", ErrOwnerRecoveryInvalidInput)
			}
			return fmt.Errorf("load target user: %w", err)
		}
		if target.DeletedAt.Valid || !target.IsActive {
			return fmt.Errorf("%w: target user must be active", ErrOwnerRecoveryInvalidInput)
		}

		var operator models.User
		if err := tx.Unscoped().Where("LOWER(email) = ?", operatorEmail).First(&operator).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: operator user not found", ErrOwnerRecoveryInvalidInput)
			}
			return fmt.Errorf("load operator user: %w", err)
		}
		if operator.DeletedAt.Valid || !operator.IsActive {
			return fmt.Errorf("%w: operator user must be active", ErrOwnerRecoveryInvalidInput)
		}

		before := map[string]interface{}{
			"role_id": target.RoleID, "role_key": target.Role,
			"permission_version": target.PermissionVersion,
		}
		previousRole := target.Role
		nextVersion := target.PermissionVersion + 1
		if err := tx.Model(&models.User{}).Where("id = ?", target.ID).Updates(map[string]interface{}{
			"role_id": ownerRole.ID, "role": ownerRole.Key, "permission_version": nextVersion,
		}).Error; err != nil {
			return fmt.Errorf("promote recovered owner: %w", err)
		}
		if err := tx.Where("user_id = ?", target.ID).Delete(&models.UserPermissionOverride{}).Error; err != nil {
			return fmt.Errorf("clear recovered owner overrides: %w", err)
		}

		after := map[string]interface{}{
			"role_id": ownerRole.ID, "role_key": ownerRole.Key,
			"permission_version": nextVersion,
		}
		beforeJSON, err := json.Marshal(before)
		if err != nil {
			return err
		}
		afterJSON, err := json.Marshal(after)
		if err != nil {
			return err
		}
		audit := models.AccessAuditLog{
			ActorUserID: operator.ID,
			TargetType:  "user",
			TargetID:    target.ID,
			Action:      "owner_recovered_via_cli",
			BeforeData:  beforeJSON,
			AfterData:   afterJSON,
			Reason:      reason,
			RequestID:   "owner-recovery-cli",
			UserAgent:   "rbac-owner-recovery",
			CreatedAt:   time.Now(),
		}
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("create owner recovery audit: %w", err)
		}
		result = OwnerRecoveryResult{
			UserID: target.ID, OperatorUserID: operator.ID, Email: target.Email, PreviousRole: previousRole,
			PermissionVersion: nextVersion, AuditLogID: audit.ID,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	observability.DefaultAccessControl.RecordOwnerChange(result.OperatorUserID, result.UserID, "owner_recovered_via_cli")
	return &result, nil
}
