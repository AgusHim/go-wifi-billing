# Route-Permission Matrix Phase 5

Sumber kebenaran executable matrix berada di `midlewares/route_permission_middleware.go` pada `RoutePermissionPolicies`.
Setiap record memuat method, path pattern, permission atau owner/auth-only exemption, data scope, dan risk.

## Aturan CI/Test

- Semua route dalam namespace `/admin_api/*`, `/user_api/*`, `/api/users*`, dan `/api/auth/me` wajib mempunyai record matrix.
- Route privat baru tanpa policy ditolak fail-closed dengan HTTP 403 dan menyebabkan `TestEveryPrivateRouteHasPermissionPolicy` gagal.
- Permission pada matrix wajib tersedia dalam permission catalog; `TestRoutePermissionCatalogCoverage` akan gagal bila key belum di-seed.
- Public exemption harus eksplisit. Saat ini hanya `GET /user_api/bills/public/:public_id` yang berada di namespace privat tetapi sah sebagai public route.
- Endpoint access control menggunakan hard-check owner, bukan permission delegable.

## Ringkasan Matrix

| Area | Read | Create/Action | Update | Delete | Data scope |
| --- | --- | --- | --- | --- | --- |
| Bills | `bills.read`, `dashboard.read`, `self.bills.read` | `bills.create`, `bills.generate`, `bills.mark_overdue`, `bills.send_reminder` | `bills.update` | `bills.delete` | global untuk owner/admin; assigned user untuk operasional; self untuk customer |
| Payments | `payments.read`, `self.payments.read` | `payments.create`, `self.payments.create` | `payments.update` | `payments.delete` | global/assigned/self |
| Customers | `customers.read` | `customers.create`, `customers.import`, `customers.export` | `customers.update` | `customers.delete` | global/assigned |
| Subscriptions | `subscriptions.read` | `subscriptions.create`, `subscriptions.renew` | `subscriptions.update` | `subscriptions.delete` | global/assigned dan self endpoint terikat JWT |
| Complaints | `complaints.read` atau `self.complaints.manage` | `complaints.create` atau self | `complaints.update` atau self | `complaints.delete` atau self | ownership customer tetap diperiksa controller/service |
| Coverage/ODC/ODP/Package | `*.read` | `*.create` | `*.update` | `*.delete` | global |
| Router/Network plan | `routers.read`, `network_plans.read` | aksi granular test/import/health/sync | `*.update` | `*.delete` | global |
| Service account | `service_accounts.read` | create/provision/suspend/unsuspend/terminate/change-plan | `service_accounts.update` | `service_accounts.delete` | global |
| NOC | `noc.read` | collect/evaluate/manage/reconcile/run action | granular action | - | global |
| User administration | `users.read` | `users.create` | `users.update` | `users.delete` | global; owner tidak dapat dihapus lewat API lama |
| Finance/expense/settings/WhatsApp/voucher | permission modul dan aksi masing-masing | granular | granular | granular | global |
| Inventory | `inventory.read` | master/purchase/receive/transfer/use/opname/approve | granular | master | global; accounting memakai permission terpisah |
| Access control | hard-check owner | hard-check owner | hard-check owner | hard-check owner | owner-only |

Matrix lengkap sengaja tidak diduplikasi sebagai tabel statis agar dokumentasi tidak menyimpang dari enforcement. Registry Go adalah fixture ter-versioning dan dibaca langsung oleh test coverage.

## Integrasi Frontend

- `/api/auth/me` menjadi sumber role canonical, flag owner, effective permission, dan `permission_version`.
- Sidebar dan seluruh route `/admin/*` memakai effective permission; route create/edit/import memakai permission aksi, bukan sekadar permission read.
- Komponen `Can`, page guard, delete guard, dan permission fieldset inventory menyembunyikan atau menonaktifkan aksi yang tidak tersedia.
- Profil sesi disegarkan saat login, startup, window kembali aktif, serta secara berkala agar perubahan owner tercermin tanpa login ulang.
- Cookie role tidak lagi dipakai untuk memutuskan akses; proxy hanya memvalidasi keberadaan dan expiry token sebelum page guard mengambil profil server.
