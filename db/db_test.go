package db

import (
	"testing"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAutoMigrateIncludesAccessControlFoundation(t *testing.T) {
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_foreign_keys=1"
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := AutoMigrate(database); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	modelsToCheck := []any{
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.UserPermissionOverride{},
		&models.AccessAuditLog{},
	}
	for _, model := range modelsToCheck {
		if !database.Migrator().HasTable(model) {
			t.Errorf("missing migrated table for %T", model)
		}
	}

	userColumns := []string{"role_id", "permission_version", "is_active"}
	for _, column := range userColumns {
		if !database.Migrator().HasColumn(&models.User{}, column) {
			t.Errorf("users missing migrated column %s", column)
		}
	}
}
