package seed

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Agushim/go_wifi_billing/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type roleSeed struct {
	Key         string
	Name        string
	Description string
	IsOwner     bool
}

type permissionSeed struct {
	Key         string
	Name        string
	RiskLevel   string
	Description string
}

type RoleAuditUser struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	LegacyRole    string    `json:"legacy_role"`
	CanonicalRole string    `json:"canonical_role,omitempty"`
	Issue         string    `json:"issue,omitempty"`
	IsDeleted     bool      `json:"is_deleted"`
	IsActive      *bool     `json:"is_active,omitempty"`
}

type RoleAuditReport struct {
	Users             []RoleAuditUser `json:"users"`
	UnknownRoleUsers  []RoleAuditUser `json:"unknown_role_users"`
	OwnerCandidates   []RoleAuditUser `json:"owner_candidates"`
	InitialOwnerEmail string          `json:"initial_owner_email,omitempty"`
}

var canonicalRoles = []roleSeed{
	{Key: "owner", Name: "Owner", Description: "Pemilik sistem dengan kontrol penuh termasuk pengaturan hak akses.", IsOwner: true},
	{Key: "admin", Name: "Admin", Description: "Administrator operasional dan administrasi harian."},
	{Key: "petugas", Name: "Petugas", Description: "Petugas lapangan atau back office sesuai penugasan."},
	{Key: "loket", Name: "Loket", Description: "Petugas tagihan, pembayaran, customer, dan subscription sesuai scope."},
	{Key: "teknisi", Name: "Teknisi", Description: "Teknisi instalasi, komplain, NOC, dan pekerjaan jaringan."},
	{Key: "customer", Name: "Customer", Description: "Pelanggan dengan akses ke portal dan datanya sendiri."},
}

var permissionCatalog = []permissionSeed{
	{Key: "dashboard.read", Name: "Lihat dashboard", RiskLevel: "low"},
	{Key: "bills.read", Name: "Lihat tagihan", RiskLevel: "low"},
	{Key: "bills.create", Name: "Buat tagihan", RiskLevel: "medium"},
	{Key: "bills.update", Name: "Ubah tagihan", RiskLevel: "high"},
	{Key: "bills.delete", Name: "Hapus tagihan", RiskLevel: "critical"},
	{Key: "bills.generate", Name: "Generate tagihan", RiskLevel: "high"},
	{Key: "bills.mark_overdue", Name: "Tandai tagihan overdue", RiskLevel: "high"},
	{Key: "bills.send_reminder", Name: "Kirim pengingat tagihan", RiskLevel: "medium"},
	{Key: "bills.export", Name: "Export tagihan", RiskLevel: "medium"},
	{Key: "payments.read", Name: "Lihat pembayaran", RiskLevel: "low"},
	{Key: "payments.create", Name: "Catat pembayaran", RiskLevel: "high"},
	{Key: "payments.update", Name: "Ubah pembayaran", RiskLevel: "high"},
	{Key: "payments.delete", Name: "Hapus pembayaran", RiskLevel: "critical"},
	{Key: "payments.export", Name: "Export pembayaran", RiskLevel: "medium"},
	{Key: "finance.read", Name: "Lihat laporan keuangan", RiskLevel: "high"},
	{Key: "finance.export", Name: "Export laporan keuangan", RiskLevel: "high"},
	{Key: "expenses.read", Name: "Lihat pengeluaran", RiskLevel: "medium"},
	{Key: "expenses.create", Name: "Buat pengeluaran", RiskLevel: "high"},
	{Key: "expenses.update", Name: "Ubah pengeluaran", RiskLevel: "high"},
	{Key: "expenses.delete", Name: "Hapus pengeluaran", RiskLevel: "critical"},
	{Key: "customers.read", Name: "Lihat customer", RiskLevel: "low"},
	{Key: "customers.create", Name: "Buat customer", RiskLevel: "medium"},
	{Key: "customers.update", Name: "Ubah customer", RiskLevel: "medium"},
	{Key: "customers.delete", Name: "Hapus customer", RiskLevel: "critical"},
	{Key: "customers.import", Name: "Import customer", RiskLevel: "high"},
	{Key: "customers.export", Name: "Export customer", RiskLevel: "medium"},
	{Key: "subscriptions.read", Name: "Lihat subscription", RiskLevel: "low"},
	{Key: "subscriptions.create", Name: "Buat subscription", RiskLevel: "medium"},
	{Key: "subscriptions.update", Name: "Ubah subscription", RiskLevel: "high"},
	{Key: "subscriptions.delete", Name: "Hapus subscription", RiskLevel: "critical"},
	{Key: "subscriptions.renew", Name: "Perpanjang subscription", RiskLevel: "high"},
	{Key: "complaints.read", Name: "Lihat komplain", RiskLevel: "low"},
	{Key: "complaints.create", Name: "Buat komplain", RiskLevel: "low"},
	{Key: "complaints.update", Name: "Ubah komplain", RiskLevel: "medium"},
	{Key: "complaints.delete", Name: "Hapus komplain", RiskLevel: "high"},
	{Key: "complaints.assign", Name: "Assign komplain", RiskLevel: "medium"},
	{Key: "packages.read", Name: "Lihat paket", RiskLevel: "low"},
	{Key: "packages.create", Name: "Buat paket", RiskLevel: "high"},
	{Key: "packages.update", Name: "Ubah paket", RiskLevel: "high"},
	{Key: "packages.delete", Name: "Hapus paket", RiskLevel: "critical"},
	{Key: "vouchers.read", Name: "Lihat voucher", RiskLevel: "low"},
	{Key: "vouchers.create", Name: "Buat voucher", RiskLevel: "high"},
	{Key: "vouchers.redeem", Name: "Redeem voucher", RiskLevel: "medium"},
	{Key: "vouchers.manage", Name: "Kelola voucher", RiskLevel: "high"},
	{Key: "coverages.read", Name: "Lihat coverage", RiskLevel: "low"},
	{Key: "coverages.create", Name: "Buat coverage", RiskLevel: "high"},
	{Key: "coverages.update", Name: "Ubah coverage", RiskLevel: "high"},
	{Key: "coverages.delete", Name: "Hapus coverage", RiskLevel: "critical"},
	{Key: "odcs.read", Name: "Lihat ODC", RiskLevel: "low"},
	{Key: "odcs.create", Name: "Buat ODC", RiskLevel: "high"},
	{Key: "odcs.update", Name: "Ubah ODC", RiskLevel: "high"},
	{Key: "odcs.delete", Name: "Hapus ODC", RiskLevel: "critical"},
	{Key: "odps.read", Name: "Lihat ODP", RiskLevel: "low"},
	{Key: "odps.create", Name: "Buat ODP", RiskLevel: "high"},
	{Key: "odps.update", Name: "Ubah ODP", RiskLevel: "high"},
	{Key: "odps.delete", Name: "Hapus ODP", RiskLevel: "critical"},
	{Key: "routers.read", Name: "Lihat router", RiskLevel: "medium"},
	{Key: "routers.create", Name: "Buat router", RiskLevel: "critical"},
	{Key: "routers.update", Name: "Ubah router", RiskLevel: "critical"},
	{Key: "routers.delete", Name: "Hapus router", RiskLevel: "critical"},
	{Key: "routers.test_connection", Name: "Test koneksi router", RiskLevel: "high"},
	{Key: "routers.health_check", Name: "Jalankan health check router", RiskLevel: "high"},
	{Key: "routers.import", Name: "Import resource router", RiskLevel: "critical"},
	{Key: "network_plans.read", Name: "Lihat network plan", RiskLevel: "low"},
	{Key: "network_plans.create", Name: "Buat network plan", RiskLevel: "high"},
	{Key: "network_plans.update", Name: "Ubah network plan", RiskLevel: "high"},
	{Key: "network_plans.delete", Name: "Hapus network plan", RiskLevel: "critical"},
	{Key: "network_plans.sync", Name: "Sinkronkan network plan", RiskLevel: "critical"},
	{Key: "service_accounts.read", Name: "Lihat service account", RiskLevel: "medium"},
	{Key: "service_accounts.create", Name: "Buat service account", RiskLevel: "high"},
	{Key: "service_accounts.update", Name: "Ubah service account", RiskLevel: "high"},
	{Key: "service_accounts.delete", Name: "Hapus service account", RiskLevel: "critical"},
	{Key: "service_accounts.provision", Name: "Provision service account", RiskLevel: "critical"},
	{Key: "service_accounts.suspend", Name: "Suspend service account", RiskLevel: "critical"},
	{Key: "service_accounts.unsuspend", Name: "Unsuspend service account", RiskLevel: "critical"},
	{Key: "service_accounts.terminate", Name: "Terminate service account", RiskLevel: "critical"},
	{Key: "service_accounts.change_plan", Name: "Ganti plan service account", RiskLevel: "critical"},
	{Key: "provisioning_logs.read", Name: "Lihat provisioning log", RiskLevel: "medium"},
	{Key: "noc.read", Name: "Lihat NOC", RiskLevel: "medium"},
	{Key: "noc.collect", Name: "Jalankan collector NOC", RiskLevel: "high"},
	{Key: "noc.evaluate_alerts", Name: "Evaluasi alert NOC", RiskLevel: "high"},
	{Key: "noc.manage_alerts", Name: "Kelola alert NOC", RiskLevel: "high"},
	{Key: "noc.reconcile", Name: "Rekonsiliasi NOC", RiskLevel: "critical"},
	{Key: "noc.run_action", Name: "Jalankan aksi NOC", RiskLevel: "critical"},
	{Key: "users.read", Name: "Lihat user", RiskLevel: "medium"},
	{Key: "users.create", Name: "Buat user", RiskLevel: "high"},
	{Key: "users.update", Name: "Ubah user", RiskLevel: "high"},
	{Key: "users.delete", Name: "Hapus user", RiskLevel: "critical"},
	{Key: "settings.read", Name: "Lihat settings", RiskLevel: "medium"},
	{Key: "settings.update", Name: "Ubah settings", RiskLevel: "critical"},
	{Key: "whatsapp.read", Name: "Lihat WhatsApp", RiskLevel: "medium"},
	{Key: "whatsapp.send", Name: "Kirim WhatsApp", RiskLevel: "high"},
	{Key: "whatsapp_templates.manage", Name: "Kelola template WhatsApp", RiskLevel: "high"},
	{Key: "inventory.read", Name: "Lihat inventory", RiskLevel: "low"},
	{Key: "inventory.manage_master", Name: "Kelola master inventory", RiskLevel: "high"},
	{Key: "inventory.purchase", Name: "Kelola pembelian inventory", RiskLevel: "high"},
	{Key: "inventory.receive", Name: "Terima barang", RiskLevel: "high"},
	{Key: "inventory.transfer", Name: "Transfer stok", RiskLevel: "high"},
	{Key: "inventory.use_material", Name: "Gunakan material", RiskLevel: "high"},
	{Key: "inventory.stock_opname", Name: "Kelola stock opname", RiskLevel: "high"},
	{Key: "inventory.approve", Name: "Approve inventory", RiskLevel: "critical"},
	{Key: "inventory.accounting.read", Name: "Lihat accounting inventory", RiskLevel: "high"},
	{Key: "inventory.accounting.manage", Name: "Kelola accounting inventory", RiskLevel: "critical"},
	{Key: "audit_logs.read", Name: "Lihat audit log", RiskLevel: "high"},
	{Key: "access_control.read", Name: "Lihat hak akses", RiskLevel: "critical"},
	{Key: "access_control.manage", Name: "Kelola hak akses", RiskLevel: "critical"},
	{Key: "self.bills.read", Name: "Lihat tagihan sendiri", RiskLevel: "low"},
	{Key: "self.payments.read", Name: "Lihat pembayaran sendiri", RiskLevel: "low"},
	{Key: "self.payments.create", Name: "Buat pembayaran sendiri", RiskLevel: "medium"},
	{Key: "self.complaints.manage", Name: "Kelola komplain sendiri", RiskLevel: "low"},
	{Key: "self.profile.update", Name: "Ubah profil sendiri", RiskLevel: "low"},
}

var defaultRolePermissions = map[string][]string{
	"admin": {
		"dashboard.read", "bills.read", "bills.create", "bills.update", "bills.generate", "bills.mark_overdue", "bills.send_reminder", "bills.export",
		"payments.read", "payments.create", "payments.update", "payments.export", "finance.read", "finance.export",
		"expenses.read", "expenses.create", "expenses.update", "customers.read", "customers.create", "customers.update", "customers.import", "customers.export",
		"subscriptions.read", "subscriptions.create", "subscriptions.update", "subscriptions.renew", "complaints.read", "complaints.create", "complaints.update", "complaints.assign",
		"packages.read", "packages.create", "packages.update", "vouchers.read", "vouchers.create", "vouchers.manage",
		"coverages.read", "coverages.create", "coverages.update", "odcs.read", "odcs.create", "odcs.update", "odps.read", "odps.create", "odps.update",
		"routers.read", "routers.create", "routers.update", "routers.test_connection", "routers.health_check", "routers.import",
		"network_plans.read", "network_plans.create", "network_plans.update", "network_plans.sync",
		"service_accounts.read", "service_accounts.create", "service_accounts.update", "service_accounts.provision", "service_accounts.suspend", "service_accounts.unsuspend", "service_accounts.change_plan",
		"provisioning_logs.read", "noc.read", "noc.collect", "noc.evaluate_alerts", "noc.manage_alerts", "noc.reconcile", "noc.run_action",
		"users.read", "users.create", "users.update", "settings.read", "whatsapp.read", "whatsapp.send", "whatsapp_templates.manage",
		"inventory.read", "inventory.manage_master", "inventory.purchase", "inventory.receive", "inventory.transfer", "inventory.use_material", "inventory.stock_opname", "inventory.approve", "inventory.accounting.read",
	},
	"petugas": {
		"dashboard.read", "bills.read", "bills.create", "bills.update", "payments.read", "payments.create", "payments.update",
		"customers.read", "customers.create", "customers.update", "subscriptions.read", "subscriptions.create", "subscriptions.update",
		"complaints.read", "complaints.create", "complaints.update", "packages.read", "coverages.read", "odcs.read", "odps.read",
		"noc.read", "routers.read", "network_plans.read", "service_accounts.read", "inventory.read", "inventory.transfer", "inventory.use_material",
	},
	"loket": {
		"dashboard.read", "bills.read", "bills.create", "bills.update", "payments.read", "payments.create", "payments.update",
		"customers.read", "customers.create", "customers.update", "subscriptions.read", "subscriptions.create", "subscriptions.update",
		"complaints.read", "complaints.create", "complaints.update", "packages.read", "coverages.read", "odcs.read", "odps.read", "inventory.read",
	},
	"teknisi": {
		"dashboard.read", "customers.read", "customers.update", "subscriptions.read", "complaints.read", "complaints.update",
		"packages.read", "coverages.read", "odcs.read", "odps.read", "routers.read", "network_plans.read", "service_accounts.read",
		"service_accounts.provision", "service_accounts.suspend", "service_accounts.unsuspend", "noc.read", "noc.manage_alerts", "noc.run_action",
		"inventory.read", "inventory.transfer", "inventory.use_material",
	},
	"customer": {"self.bills.read", "self.payments.read", "self.payments.create", "self.complaints.manage", "self.profile.update", "vouchers.redeem"},
}

// CanonicalRoleKey normalizes legacy role names without mutating the legacy
// users.role column. Keeping that column intact preserves compatibility until
// all existing role checks have moved to the authorization service.
func CanonicalRoleKey(role string) (string, bool) {
	return models.CanonicalRoleKey(role)
}

// AuditLegacyUserRoles performs a read-only preflight that is safe to run
// before the RBAC migration because it only selects legacy users columns.
func AuditLegacyUserRoles(db *gorm.DB, initialOwnerEmail string) (RoleAuditReport, error) {
	type legacyUser struct {
		ID        uuid.UUID
		Email     string
		Role      string
		DeletedAt gorm.DeletedAt
		IsActive  *bool
	}

	var rows []legacyUser
	columns := []string{"id", "email", "role", "deleted_at"}
	if db.Migrator().HasColumn("users", "is_active") {
		columns = append(columns, "is_active")
	}
	if err := db.Unscoped().Table("users").Select(columns).Scan(&rows).Error; err != nil {
		return RoleAuditReport{}, err
	}

	report := RoleAuditReport{InitialOwnerEmail: strings.TrimSpace(initialOwnerEmail)}
	for _, row := range rows {
		canonical, known := CanonicalRoleKey(row.Role)
		item := RoleAuditUser{
			ID: row.ID, Email: row.Email, LegacyRole: row.Role, CanonicalRole: canonical,
			IsDeleted: row.DeletedAt.Valid, IsActive: row.IsActive,
		}
		if !known {
			item.Issue = "unknown_role"
			report.UnknownRoleUsers = append(report.UnknownRoleUsers, item)
		}
		if canonical == "owner" && !row.DeletedAt.Valid && (row.IsActive == nil || *row.IsActive) {
			report.OwnerCandidates = append(report.OwnerCandidates, item)
		}
		report.Users = append(report.Users, item)
	}

	if len(report.OwnerCandidates) == 0 && report.InitialOwnerEmail != "" {
		for _, item := range report.Users {
			if strings.EqualFold(strings.TrimSpace(item.Email), report.InitialOwnerEmail) && item.Issue == "" && !item.IsDeleted && (item.IsActive == nil || *item.IsActive) {
				report.OwnerCandidates = append(report.OwnerCandidates, item)
			}
		}
	}
	return report, nil
}

// SeedAccessControl seeds the versioned RBAC catalog, backfills user role IDs,
// and validates the owner invariants in a single transaction.
func SeedAccessControl(db *gorm.DB, initialOwnerEmail string) error {
	if db == nil {
		return errors.New("seed access control: db is nil")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		rolesByKey, err := upsertRoles(tx)
		if err != nil {
			return err
		}
		permissionsByKey, err := upsertPermissions(tx)
		if err != nil {
			return err
		}
		if err := seedDefaultRolePermissions(tx, rolesByKey, permissionsByKey); err != nil {
			return err
		}
		if err := backfillUserRoles(tx, rolesByKey, initialOwnerEmail); err != nil {
			return err
		}
		return validateAccessControlFoundation(tx)
	})
}

func upsertRoles(tx *gorm.DB) (map[string]models.Role, error) {
	for _, item := range canonicalRoles {
		role := models.Role{Key: item.Key, Name: item.Name, Description: item.Description, IsSystem: true, IsOwner: item.IsOwner, IsActive: true}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"name", "description", "is_system", "is_owner", "is_active", "updated_at"}),
		}).Create(&role).Error; err != nil {
			return nil, fmt.Errorf("seed role %s: %w", item.Key, err)
		}
	}

	var roles []models.Role
	if err := tx.Where("key IN ?", canonicalRoleKeys()).Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("load canonical roles: %w", err)
	}
	result := make(map[string]models.Role, len(roles))
	for _, role := range roles {
		result[role.Key] = role
	}
	if len(result) != len(canonicalRoles) {
		return nil, fmt.Errorf("seed roles incomplete: got %d, want %d", len(result), len(canonicalRoles))
	}
	return result, nil
}

func upsertPermissions(tx *gorm.DB) (map[string]models.Permission, error) {
	for index, item := range permissionCatalog {
		parts := strings.SplitN(item.Key, ".", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid permission key %q", item.Key)
		}
		permission := models.Permission{
			Key: item.Key, Module: parts[0], Action: parts[1], Name: item.Name,
			Description: item.Description, RiskLevel: item.RiskLevel, SortOrder: index + 1, IsSystem: true,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"module", "action", "name", "description", "risk_level", "sort_order", "is_system", "updated_at"}),
		}).Create(&permission).Error; err != nil {
			return nil, fmt.Errorf("seed permission %s: %w", item.Key, err)
		}
	}

	var permissions []models.Permission
	if err := tx.Find(&permissions).Error; err != nil {
		return nil, fmt.Errorf("load permissions: %w", err)
	}
	result := make(map[string]models.Permission, len(permissions))
	for _, permission := range permissions {
		result[permission.Key] = permission
	}
	for _, item := range permissionCatalog {
		if _, exists := result[item.Key]; !exists {
			return nil, fmt.Errorf("seed permission missing after upsert: %s", item.Key)
		}
	}
	return result, nil
}

func seedDefaultRolePermissions(tx *gorm.DB, roles map[string]models.Role, permissions map[string]models.Permission) error {
	owner := roles["owner"]
	for _, permission := range permissions {
		if err := createRolePermission(tx, owner.ID, permission.ID); err != nil {
			return err
		}
	}
	for roleKey, permissionKeys := range defaultRolePermissions {
		role := roles[roleKey]
		for _, permissionKey := range permissionKeys {
			permission, exists := permissions[permissionKey]
			if !exists {
				return fmt.Errorf("default role %s references unknown permission %s", roleKey, permissionKey)
			}
			if err := createRolePermission(tx, role.ID, permission.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func createRolePermission(tx *gorm.DB, roleID, permissionID uuid.UUID) error {
	row := models.RolePermission{RoleID: roleID, PermissionID: permissionID}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return fmt.Errorf("seed role permission: %w", err)
	}
	return nil
}

func backfillUserRoles(tx *gorm.DB, roles map[string]models.Role, initialOwnerEmail string) error {
	var users []models.User
	if err := tx.Unscoped().Find(&users).Error; err != nil {
		return fmt.Errorf("load users for role backfill: %w", err)
	}
	if len(users) == 0 {
		return errors.New("owner bootstrap failed: no users exist; create the intended owner user first")
	}

	validRoleIDs := make(map[uuid.UUID]bool, len(roles))
	for _, role := range roles {
		validRoleIDs[role.ID] = true
	}
	ownerRole := roles["owner"]
	hasActiveOwner := false
	var unknown []string

	for _, user := range users {
		canonical, known := CanonicalRoleKey(user.Role)
		if !known && (user.RoleID == nil || !validRoleIDs[*user.RoleID]) {
			unknown = append(unknown, fmt.Sprintf("%s <%s>", user.ID, user.Email))
			continue
		}
		if user.DeletedAt.Valid || !user.IsActive {
			continue
		}
		if canonical == "owner" || (user.RoleID != nil && *user.RoleID == ownerRole.ID) {
			hasActiveOwner = true
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("role backfill blocked; users with unknown role: %s", strings.Join(unknown, ", "))
	}

	bootstrapEmail := strings.TrimSpace(initialOwnerEmail)
	var bootstrapOwnerID uuid.UUID
	if !hasActiveOwner {
		if bootstrapEmail == "" {
			return errors.New("owner bootstrap failed: INITIAL_OWNER_EMAIL is required because no active owner exists")
		}
		matches := 0
		for _, user := range users {
			if !user.DeletedAt.Valid && user.IsActive && strings.EqualFold(strings.TrimSpace(user.Email), bootstrapEmail) {
				bootstrapOwnerID = user.ID
				matches++
			}
		}
		if matches != 1 {
			return fmt.Errorf("owner bootstrap failed: INITIAL_OWNER_EMAIL must identify exactly one active user; found %d", matches)
		}
	}

	for _, user := range users {
		if user.ID == bootstrapOwnerID {
			if err := tx.Unscoped().Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
				"role_id": ownerRole.ID, "permission_version": gorm.Expr("CASE WHEN permission_version < 1 THEN 1 ELSE permission_version END"),
			}).Error; err != nil {
				return fmt.Errorf("bootstrap owner %s: %w", user.Email, err)
			}
			continue
		}
		if user.RoleID != nil && validRoleIDs[*user.RoleID] {
			continue
		}
		canonical, _ := CanonicalRoleKey(user.Role)
		role := roles[canonical]
		if err := tx.Unscoped().Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{
			"role_id": role.ID, "permission_version": gorm.Expr("CASE WHEN permission_version < 1 THEN 1 ELSE permission_version END"),
		}).Error; err != nil {
			return fmt.Errorf("backfill role for user %s: %w", user.Email, err)
		}
	}
	return nil
}

func validateAccessControlFoundation(tx *gorm.DB) error {
	var ownerRoleCount int64
	if err := tx.Model(&models.Role{}).Where("is_owner = ?", true).Count(&ownerRoleCount).Error; err != nil {
		return fmt.Errorf("count owner roles: %w", err)
	}
	if ownerRoleCount != 1 {
		return fmt.Errorf("access control invalid: got %d owner roles, want exactly 1", ownerRoleCount)
	}

	var unmappedUsers int64
	if err := tx.Unscoped().Model(&models.User{}).Where("role_id IS NULL").Count(&unmappedUsers).Error; err != nil {
		return fmt.Errorf("count users without role mapping: %w", err)
	}
	if unmappedUsers != 0 {
		return fmt.Errorf("access control invalid: %d users have no role mapping", unmappedUsers)
	}

	var invalidMappings int64
	if err := tx.Unscoped().Table("users").
		Joins("LEFT JOIN roles ON roles.id = users.role_id").
		Where("roles.id IS NULL").Count(&invalidMappings).Error; err != nil {
		return fmt.Errorf("count invalid user role mappings: %w", err)
	}
	if invalidMappings != 0 {
		return fmt.Errorf("access control invalid: %d users reference an invalid role", invalidMappings)
	}

	var activeOwnerCount int64
	if err := tx.Table("users").
		Joins("JOIN roles ON roles.id = users.role_id").
		Where("roles.is_owner = ? AND users.is_active = ? AND users.deleted_at IS NULL", true, true).
		Count(&activeOwnerCount).Error; err != nil {
		return fmt.Errorf("count active owners: %w", err)
	}
	if activeOwnerCount < 1 {
		return errors.New("access control invalid: no active owner exists")
	}
	return nil
}

func canonicalRoleKeys() []string {
	keys := make([]string, 0, len(canonicalRoles))
	for _, role := range canonicalRoles {
		keys = append(keys, role.Key)
	}
	return keys
}
