package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRecoverOwnerPromotesTargetAndWritesAudit(t *testing.T) {
	database, ownerRole, adminRole := newOwnerRecoveryTestDB(t)
	target := createAccessControlTestUser(t, database, "recover-target@example.com", "admin", adminRole, 7)

	result, err := RecoverOwner(context.Background(), database, OwnerRecoveryRequest{
		TargetEmail: target.Email, OperatorEmail: target.Email, Reason: "INC-2026-071 restore emergency owner",
	})
	if err != nil {
		t.Fatalf("recover owner: %v", err)
	}
	if result.UserID != target.ID || result.PermissionVersion != 8 || result.PreviousRole != "admin" {
		t.Fatalf("unexpected result: %#v", result)
	}
	var reloaded models.User
	if err := database.First(&reloaded, "id = ?", target.ID).Error; err != nil {
		t.Fatalf("reload target: %v", err)
	}
	if reloaded.RoleID == nil || *reloaded.RoleID != ownerRole.ID || reloaded.Role != "owner" || reloaded.PermissionVersion != 8 {
		t.Fatalf("target was not promoted atomically: %#v", reloaded)
	}
	var audit models.AccessAuditLog
	if err := database.First(&audit, "id = ?", result.AuditLogID).Error; err != nil {
		t.Fatalf("load recovery audit: %v", err)
	}
	if audit.Action != "owner_recovered_via_cli" || audit.ActorUserID != target.ID || audit.Reason == "" {
		t.Fatalf("unexpected recovery audit: %#v", audit)
	}
}

func TestRecoverOwnerRefusesWhenActiveOwnerExists(t *testing.T) {
	database, ownerRole, adminRole := newOwnerRecoveryTestDB(t)
	createAccessControlTestUser(t, database, "existing-owner@example.com", "owner", ownerRole, 1)
	target := createAccessControlTestUser(t, database, "blocked-target@example.com", "admin", adminRole, 1)

	_, err := RecoverOwner(context.Background(), database, OwnerRecoveryRequest{
		TargetEmail: target.Email, Reason: "should be blocked",
	})
	if !errors.Is(err, ErrOwnerRecoveryNotRequired) {
		t.Fatalf("error = %v, want active owner refusal", err)
	}
	assertAccessControlAuditCount(t, database, 0)
}

func TestRecoverOwnerRequiresReasonWithoutMutation(t *testing.T) {
	database, _, adminRole := newOwnerRecoveryTestDB(t)
	target := createAccessControlTestUser(t, database, "reason-target@example.com", "admin", adminRole, 3)
	_, err := RecoverOwner(context.Background(), database, OwnerRecoveryRequest{TargetEmail: target.Email})
	if !errors.Is(err, ErrOwnerRecoveryInvalidInput) {
		t.Fatalf("error = %v, want invalid input", err)
	}
	var reloaded models.User
	if err := database.First(&reloaded, "id = ?", target.ID).Error; err != nil {
		t.Fatalf("reload target: %v", err)
	}
	if reloaded.Role != "admin" || reloaded.PermissionVersion != 3 {
		t.Fatalf("target changed after invalid request: %#v", reloaded)
	}
	assertAccessControlAuditCount(t, database, 0)
}

func newOwnerRecoveryTestDB(t *testing.T) (*gorm.DB, models.Role, models.Role) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared&_foreign_keys=1"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := database.AutoMigrate(
		&models.Coverage{}, &models.Role{}, &models.Permission{}, &models.User{},
		&models.UserPermissionOverride{}, &models.AccessAuditLog{},
	); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	ownerRole := models.Role{Key: "owner", Name: "Owner", IsOwner: true, IsSystem: true, IsActive: true}
	adminRole := models.Role{Key: "admin", Name: "Admin", IsSystem: true, IsActive: true}
	if err := database.Create(&ownerRole).Error; err != nil {
		t.Fatalf("create owner role: %v", err)
	}
	if err := database.Create(&adminRole).Error; err != nil {
		t.Fatalf("create admin role: %v", err)
	}
	return database, ownerRole, adminRole
}
