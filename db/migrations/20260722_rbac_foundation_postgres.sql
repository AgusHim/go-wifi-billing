-- RBAC foundation and user-role backfill for PostgreSQL.
--
-- This migration is intentionally fail-closed. It does not guess the owner.
-- Run the read-only audit first:
--   INITIAL_OWNER_EMAIL=owner@example.com go run ./cmd/rbac-role-audit
--
-- Then run after a database backup, passing the exact existing owner email:
--   psql "$POSTGRES_URL" -v ON_ERROR_STOP=1 \
--     -v initial_owner_email='owner@example.com' \
--     -f db/migrations/20260722_rbac_foundation_postgres.sql

BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS roles (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  key varchar(50) NOT NULL CONSTRAINT uni_roles_key UNIQUE,
  name varchar(100) NOT NULL,
  description text NOT NULL DEFAULT '',
  is_system boolean NOT NULL DEFAULT false,
  is_owner boolean NOT NULL DEFAULT false,
  is_active boolean NOT NULL DEFAULT true,
  created_by uuid,
  updated_by uuid,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_roles_owner_is_system CHECK (NOT is_owner OR is_system)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_single_owner
ON roles (is_owner)
WHERE is_owner = true;

CREATE INDEX IF NOT EXISTS idx_roles_is_active ON roles (is_active);

CREATE TABLE IF NOT EXISTS permissions (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  key varchar(100) NOT NULL CONSTRAINT uni_permissions_key UNIQUE,
  module varchar(50) NOT NULL,
  action varchar(50) NOT NULL,
  name varchar(150) NOT NULL,
  description text NOT NULL DEFAULT '',
  risk_level varchar(20) NOT NULL DEFAULT 'low',
  sort_order integer NOT NULL DEFAULT 0,
  is_system boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_permissions_risk_level CHECK (risk_level IN ('low', 'medium', 'high', 'critical'))
);

CREATE INDEX IF NOT EXISTS idx_permissions_module_sort
ON permissions (module, sort_order);

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS role_id uuid,
  ADD COLUMN IF NOT EXISTS permission_version bigint NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS is_active boolean NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS idx_users_role_id ON users (role_id);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users (is_active);

CREATE TABLE IF NOT EXISTS role_permissions (
  role_id uuid NOT NULL,
  permission_id uuid NOT NULL,
  created_by uuid,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (role_id, permission_id),
  CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES roles(id) ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT fk_role_permissions_permission FOREIGN KEY (permission_id) REFERENCES permissions(id) ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT fk_role_permissions_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_role
ON role_permissions (permission_id, role_id);

CREATE TABLE IF NOT EXISTS user_permission_overrides (
  user_id uuid NOT NULL,
  permission_id uuid NOT NULL,
  effect varchar(10) NOT NULL,
  reason text NOT NULL DEFAULT '',
  created_by uuid NOT NULL,
  updated_by uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, permission_id),
  CONSTRAINT chk_user_permission_overrides_effect CHECK (effect IN ('allow', 'deny')),
  CONSTRAINT fk_user_permission_overrides_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT fk_user_permission_overrides_permission FOREIGN KEY (permission_id) REFERENCES permissions(id) ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT fk_user_permission_overrides_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT,
  CONSTRAINT fk_user_permission_overrides_updated_by FOREIGN KEY (updated_by) REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_user_permission_overrides_permission_user
ON user_permission_overrides (permission_id, user_id);

CREATE TABLE IF NOT EXISTS access_audit_logs (
  id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
  actor_user_id uuid NOT NULL,
  target_type varchar(20) NOT NULL,
  target_id uuid NOT NULL,
  action varchar(80) NOT NULL,
  before_data jsonb,
  after_data jsonb,
  reason text NOT NULL DEFAULT '',
  ip_address varchar(64) NOT NULL DEFAULT '',
  user_agent text NOT NULL DEFAULT '',
  request_id varchar(100) NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT chk_access_audit_target_type CHECK (target_type IN ('role', 'user')),
  CONSTRAINT fk_access_audit_actor FOREIGN KEY (actor_user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_access_audit_actor_created
ON access_audit_logs (actor_user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_access_audit_target_created
ON access_audit_logs (target_type, target_id, created_at);
CREATE INDEX IF NOT EXISTS idx_access_audit_action ON access_audit_logs (action);
CREATE INDEX IF NOT EXISTS idx_access_audit_request_id ON access_audit_logs (request_id);

INSERT INTO roles (key, name, description, is_system, is_owner, is_active)
VALUES
  ('owner', 'Owner', 'Pemilik sistem dengan kontrol penuh termasuk pengaturan hak akses.', true, true, true),
  ('admin', 'Admin', 'Administrator operasional dan administrasi harian.', true, false, true),
  ('petugas', 'Petugas', 'Petugas lapangan atau back office sesuai penugasan.', true, false, true),
  ('loket', 'Loket', 'Petugas tagihan, pembayaran, customer, dan subscription sesuai scope.', true, false, true),
  ('teknisi', 'Teknisi', 'Teknisi instalasi, komplain, NOC, dan pekerjaan jaringan.', true, false, true),
  ('customer', 'Customer', 'Pelanggan dengan akses ke portal dan datanya sendiri.', true, false, true)
ON CONFLICT (key) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  is_system = EXCLUDED.is_system,
  is_owner = EXCLUDED.is_owner,
  is_active = EXCLUDED.is_active,
  updated_at = now();

-- Keep this catalog synchronized with db/seed/access_control_seed.go.
INSERT INTO permissions (key, module, action, name, risk_level, sort_order, is_system)
VALUES
  ('dashboard.read', 'dashboard', 'read', 'Lihat dashboard', 'low', 1, true),
  ('bills.read', 'bills', 'read', 'Lihat tagihan', 'low', 2, true),
  ('bills.create', 'bills', 'create', 'Buat tagihan', 'medium', 3, true),
  ('bills.update', 'bills', 'update', 'Ubah tagihan', 'high', 4, true),
  ('bills.delete', 'bills', 'delete', 'Hapus tagihan', 'critical', 5, true),
  ('bills.generate', 'bills', 'generate', 'Generate tagihan', 'high', 6, true),
  ('bills.mark_overdue', 'bills', 'mark_overdue', 'Tandai tagihan overdue', 'high', 7, true),
  ('bills.send_reminder', 'bills', 'send_reminder', 'Kirim pengingat tagihan', 'medium', 8, true),
  ('bills.export', 'bills', 'export', 'Export tagihan', 'medium', 9, true),
  ('payments.read', 'payments', 'read', 'Lihat pembayaran', 'low', 10, true),
  ('payments.create', 'payments', 'create', 'Catat pembayaran', 'high', 11, true),
  ('payments.update', 'payments', 'update', 'Ubah pembayaran', 'high', 12, true),
  ('payments.delete', 'payments', 'delete', 'Hapus pembayaran', 'critical', 13, true),
  ('payments.export', 'payments', 'export', 'Export pembayaran', 'medium', 14, true),
  ('finance.read', 'finance', 'read', 'Lihat laporan keuangan', 'high', 15, true),
  ('finance.export', 'finance', 'export', 'Export laporan keuangan', 'high', 16, true),
  ('expenses.read', 'expenses', 'read', 'Lihat pengeluaran', 'medium', 17, true),
  ('expenses.create', 'expenses', 'create', 'Buat pengeluaran', 'high', 18, true),
  ('expenses.update', 'expenses', 'update', 'Ubah pengeluaran', 'high', 19, true),
  ('expenses.delete', 'expenses', 'delete', 'Hapus pengeluaran', 'critical', 20, true),
  ('customers.read', 'customers', 'read', 'Lihat customer', 'low', 21, true),
  ('customers.create', 'customers', 'create', 'Buat customer', 'medium', 22, true),
  ('customers.update', 'customers', 'update', 'Ubah customer', 'medium', 23, true),
  ('customers.delete', 'customers', 'delete', 'Hapus customer', 'critical', 24, true),
  ('customers.import', 'customers', 'import', 'Import customer', 'high', 25, true),
  ('customers.export', 'customers', 'export', 'Export customer', 'medium', 26, true),
  ('subscriptions.read', 'subscriptions', 'read', 'Lihat subscription', 'low', 27, true),
  ('subscriptions.create', 'subscriptions', 'create', 'Buat subscription', 'medium', 28, true),
  ('subscriptions.update', 'subscriptions', 'update', 'Ubah subscription', 'high', 29, true),
  ('subscriptions.delete', 'subscriptions', 'delete', 'Hapus subscription', 'critical', 30, true),
  ('subscriptions.renew', 'subscriptions', 'renew', 'Perpanjang subscription', 'high', 31, true),
  ('complaints.read', 'complaints', 'read', 'Lihat komplain', 'low', 32, true),
  ('complaints.create', 'complaints', 'create', 'Buat komplain', 'low', 33, true),
  ('complaints.update', 'complaints', 'update', 'Ubah komplain', 'medium', 34, true),
  ('complaints.delete', 'complaints', 'delete', 'Hapus komplain', 'high', 35, true),
  ('complaints.assign', 'complaints', 'assign', 'Assign komplain', 'medium', 36, true),
  ('packages.read', 'packages', 'read', 'Lihat paket', 'low', 37, true),
  ('packages.create', 'packages', 'create', 'Buat paket', 'high', 38, true),
  ('packages.update', 'packages', 'update', 'Ubah paket', 'high', 39, true),
  ('packages.delete', 'packages', 'delete', 'Hapus paket', 'critical', 40, true),
  ('vouchers.read', 'vouchers', 'read', 'Lihat voucher', 'low', 41, true),
  ('vouchers.create', 'vouchers', 'create', 'Buat voucher', 'high', 42, true),
  ('vouchers.redeem', 'vouchers', 'redeem', 'Redeem voucher', 'medium', 43, true),
  ('vouchers.manage', 'vouchers', 'manage', 'Kelola voucher', 'high', 44, true),
  ('coverages.read', 'coverages', 'read', 'Lihat coverage', 'low', 45, true),
  ('coverages.create', 'coverages', 'create', 'Buat coverage', 'high', 46, true),
  ('coverages.update', 'coverages', 'update', 'Ubah coverage', 'high', 47, true),
  ('coverages.delete', 'coverages', 'delete', 'Hapus coverage', 'critical', 48, true),
  ('odcs.read', 'odcs', 'read', 'Lihat ODC', 'low', 49, true),
  ('odcs.create', 'odcs', 'create', 'Buat ODC', 'high', 50, true),
  ('odcs.update', 'odcs', 'update', 'Ubah ODC', 'high', 51, true),
  ('odcs.delete', 'odcs', 'delete', 'Hapus ODC', 'critical', 52, true),
  ('odps.read', 'odps', 'read', 'Lihat ODP', 'low', 53, true),
  ('odps.create', 'odps', 'create', 'Buat ODP', 'high', 54, true),
  ('odps.update', 'odps', 'update', 'Ubah ODP', 'high', 55, true),
  ('odps.delete', 'odps', 'delete', 'Hapus ODP', 'critical', 56, true),
  ('routers.read', 'routers', 'read', 'Lihat router', 'medium', 57, true),
  ('routers.create', 'routers', 'create', 'Buat router', 'critical', 58, true),
  ('routers.update', 'routers', 'update', 'Ubah router', 'critical', 59, true),
  ('routers.delete', 'routers', 'delete', 'Hapus router', 'critical', 60, true),
  ('routers.test_connection', 'routers', 'test_connection', 'Test koneksi router', 'high', 61, true),
  ('routers.health_check', 'routers', 'health_check', 'Jalankan health check router', 'high', 62, true),
  ('routers.import', 'routers', 'import', 'Import resource router', 'critical', 63, true),
  ('network_plans.read', 'network_plans', 'read', 'Lihat network plan', 'low', 64, true),
  ('network_plans.create', 'network_plans', 'create', 'Buat network plan', 'high', 65, true),
  ('network_plans.update', 'network_plans', 'update', 'Ubah network plan', 'high', 66, true),
  ('network_plans.delete', 'network_plans', 'delete', 'Hapus network plan', 'critical', 67, true),
  ('network_plans.sync', 'network_plans', 'sync', 'Sinkronkan network plan', 'critical', 68, true),
  ('service_accounts.read', 'service_accounts', 'read', 'Lihat service account', 'medium', 69, true),
  ('service_accounts.create', 'service_accounts', 'create', 'Buat service account', 'high', 70, true),
  ('service_accounts.update', 'service_accounts', 'update', 'Ubah service account', 'high', 71, true),
  ('service_accounts.delete', 'service_accounts', 'delete', 'Hapus service account', 'critical', 72, true),
  ('service_accounts.provision', 'service_accounts', 'provision', 'Provision service account', 'critical', 73, true),
  ('service_accounts.suspend', 'service_accounts', 'suspend', 'Suspend service account', 'critical', 74, true),
  ('service_accounts.unsuspend', 'service_accounts', 'unsuspend', 'Unsuspend service account', 'critical', 75, true),
  ('service_accounts.terminate', 'service_accounts', 'terminate', 'Terminate service account', 'critical', 76, true),
  ('service_accounts.change_plan', 'service_accounts', 'change_plan', 'Ganti plan service account', 'critical', 77, true),
  ('provisioning_logs.read', 'provisioning_logs', 'read', 'Lihat provisioning log', 'medium', 78, true),
  ('noc.read', 'noc', 'read', 'Lihat NOC', 'medium', 79, true),
  ('noc.collect', 'noc', 'collect', 'Jalankan collector NOC', 'high', 80, true),
  ('noc.evaluate_alerts', 'noc', 'evaluate_alerts', 'Evaluasi alert NOC', 'high', 81, true),
  ('noc.manage_alerts', 'noc', 'manage_alerts', 'Kelola alert NOC', 'high', 82, true),
  ('noc.reconcile', 'noc', 'reconcile', 'Rekonsiliasi NOC', 'critical', 83, true),
  ('noc.run_action', 'noc', 'run_action', 'Jalankan aksi NOC', 'critical', 84, true),
  ('users.read', 'users', 'read', 'Lihat user', 'medium', 85, true),
  ('users.create', 'users', 'create', 'Buat user', 'high', 86, true),
  ('users.update', 'users', 'update', 'Ubah user', 'high', 87, true),
  ('users.delete', 'users', 'delete', 'Hapus user', 'critical', 88, true),
  ('settings.read', 'settings', 'read', 'Lihat settings', 'medium', 89, true),
  ('settings.update', 'settings', 'update', 'Ubah settings', 'critical', 90, true),
  ('whatsapp.read', 'whatsapp', 'read', 'Lihat WhatsApp', 'medium', 91, true),
  ('whatsapp.send', 'whatsapp', 'send', 'Kirim WhatsApp', 'high', 92, true),
  ('whatsapp_templates.manage', 'whatsapp_templates', 'manage', 'Kelola template WhatsApp', 'high', 93, true),
  ('inventory.read', 'inventory', 'read', 'Lihat inventory', 'low', 94, true),
  ('inventory.manage_master', 'inventory', 'manage_master', 'Kelola master inventory', 'high', 95, true),
  ('inventory.purchase', 'inventory', 'purchase', 'Kelola pembelian inventory', 'high', 96, true),
  ('inventory.receive', 'inventory', 'receive', 'Terima barang', 'high', 97, true),
  ('inventory.transfer', 'inventory', 'transfer', 'Transfer stok', 'high', 98, true),
  ('inventory.use_material', 'inventory', 'use_material', 'Gunakan material', 'high', 99, true),
  ('inventory.stock_opname', 'inventory', 'stock_opname', 'Kelola stock opname', 'high', 100, true),
  ('inventory.approve', 'inventory', 'approve', 'Approve inventory', 'critical', 101, true),
  ('inventory.accounting.read', 'inventory', 'accounting.read', 'Lihat accounting inventory', 'high', 102, true),
  ('inventory.accounting.manage', 'inventory', 'accounting.manage', 'Kelola accounting inventory', 'critical', 103, true),
  ('audit_logs.read', 'audit_logs', 'read', 'Lihat audit log', 'high', 104, true),
  ('access_control.read', 'access_control', 'read', 'Lihat hak akses', 'critical', 105, true),
  ('access_control.manage', 'access_control', 'manage', 'Kelola hak akses', 'critical', 106, true),
  ('self.bills.read', 'self', 'bills.read', 'Lihat tagihan sendiri', 'low', 107, true),
  ('self.payments.read', 'self', 'payments.read', 'Lihat pembayaran sendiri', 'low', 108, true),
  ('self.payments.create', 'self', 'payments.create', 'Buat pembayaran sendiri', 'medium', 109, true),
  ('self.complaints.manage', 'self', 'complaints.manage', 'Kelola komplain sendiri', 'low', 110, true),
  ('self.profile.update', 'self', 'profile.update', 'Ubah profil sendiri', 'low', 111, true)
ON CONFLICT (key) DO UPDATE SET
  module = EXCLUDED.module,
  action = EXCLUDED.action,
  name = EXCLUDED.name,
  risk_level = EXCLUDED.risk_level,
  sort_order = EXCLUDED.sort_order,
  is_system = EXCLUDED.is_system,
  updated_at = now();

-- Seed owner with every catalog permission. Other default role mappings are
-- applied idempotently by SeedAccessControl at application startup.
INSERT INTO role_permissions (role_id, permission_id)
SELECT roles.id, permissions.id
FROM roles CROSS JOIN permissions
WHERE roles.key = 'owner'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Abort before mutating mappings when a legacy role cannot be normalized.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM users
    WHERE lower(trim(role)) NOT IN ('root', 'owner', 'admin', 'petugas', 'loket', 'technician', 'teknisi', 'noc', 'user', 'customer')
  ) THEN
    RAISE EXCEPTION 'RBAC migration blocked: users with unknown legacy roles exist; run cmd/rbac-role-audit';
  END IF;
END $$;

-- Preserve users.role for compatibility. Only role_id is canonical in Phase 1.
UPDATE users
SET role_id = roles.id,
    permission_version = GREATEST(permission_version, 1)
FROM roles
WHERE users.role_id IS NULL
  AND roles.key = CASE lower(trim(users.role))
    WHEN 'root' THEN 'owner'
    WHEN 'owner' THEN 'owner'
    WHEN 'admin' THEN 'admin'
    WHEN 'petugas' THEN 'petugas'
    WHEN 'loket' THEN 'loket'
    WHEN 'technician' THEN 'teknisi'
    WHEN 'teknisi' THEN 'teknisi'
    WHEN 'noc' THEN 'teknisi'
    WHEN 'user' THEN 'customer'
    WHEN 'customer' THEN 'customer'
  END;

-- psql substitutes this value before PostgreSQL executes the migration.
CREATE TEMP TABLE rbac_bootstrap_config (initial_owner_email text) ON COMMIT DROP;
INSERT INTO rbac_bootstrap_config (initial_owner_email)
VALUES (NULLIF(trim(:'initial_owner_email'), ''));

UPDATE users
SET role_id = (SELECT id FROM roles WHERE key = 'owner'),
    permission_version = GREATEST(permission_version, 1)
WHERE NOT EXISTS (
    SELECT 1
    FROM users owner_users
    JOIN roles owner_roles ON owner_roles.id = owner_users.role_id
    WHERE owner_roles.is_owner = true
      AND owner_users.is_active = true
      AND owner_users.deleted_at IS NULL
  )
  AND users.deleted_at IS NULL
  AND users.is_active = true
  AND lower(trim(users.email)) = lower((SELECT initial_owner_email FROM rbac_bootstrap_config));

DO $$
DECLARE
  owner_role_count bigint;
  active_owner_count bigint;
  unmapped_user_count bigint;
BEGIN
  SELECT count(*) INTO owner_role_count FROM roles WHERE is_owner = true;
  SELECT count(*) INTO active_owner_count
  FROM users JOIN roles ON roles.id = users.role_id
  WHERE roles.is_owner = true AND users.is_active = true AND users.deleted_at IS NULL;
  SELECT count(*) INTO unmapped_user_count FROM users WHERE role_id IS NULL;

  IF owner_role_count <> 1 THEN
    RAISE EXCEPTION 'RBAC migration failed: expected exactly one owner role, found %', owner_role_count;
  END IF;
  IF active_owner_count < 1 THEN
    RAISE EXCEPTION 'RBAC migration failed: no active owner; pass a valid existing initial_owner_email';
  END IF;
  IF unmapped_user_count <> 0 THEN
    RAISE EXCEPTION 'RBAC migration failed: % users have no role mapping', unmapped_user_count;
  END IF;
END $$;

ALTER TABLE users
  DROP CONSTRAINT IF EXISTS fk_users_role_definition;
ALTER TABLE users
  ADD CONSTRAINT fk_users_role_definition FOREIGN KEY (role_id) REFERENCES roles(id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE roles
  DROP CONSTRAINT IF EXISTS fk_roles_created_by,
  DROP CONSTRAINT IF EXISTS fk_roles_updated_by;
ALTER TABLE roles
  ADD CONSTRAINT fk_roles_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL,
  ADD CONSTRAINT fk_roles_updated_by FOREIGN KEY (updated_by) REFERENCES users(id) ON UPDATE CASCADE ON DELETE SET NULL;

COMMIT;
