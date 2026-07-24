# Phase 0 Route Access Inventory

Dokumen ini mencatat klasifikasi route setelah hardening awal. Guard ini bersifat transisi sampai seluruh route memakai permission middleware pada Phase 2 dan Phase 5.

## Public routes

| Route | Alasan |
| --- | --- |
| `GET /health` | Health check deployment |
| `POST /api/auth/login` | Membuat sesi |
| `POST /api/auth/register` | Registrasi customer; backend selalu menetapkan role customer |
| `GET /api/bills/:public_id` | Invoice publik berdasarkan opaque public ID |
| `GET /user_api/bills/public/:public_id` | Alias kompatibilitas invoice publik |
| `POST /api/payments/callback` | Callback Midtrans; signature diverifikasi oleh service |
| `POST /api/vouchers/redeem` | Redeem token voucher publik/self-service |

Semua route lain dianggap private. Route baru harus ditambahkan ke tabel public secara eksplisit atau diberi `UserProtected()` dan authorization guard.

## Customer-only routes

- `/user_api/bills/*`
- `/user_api/payments/*`
- `/user_api/subscriptions/*`
- `/user_api/customers/me`

Role transisi yang diterima: `user` dan `customer`.

`/admin_api/complains/*` masih dipakai bersama oleh frontend admin dan portal customer. Route tersebut wajib authenticated; handler membatasi customer ke complain miliknya berdasarkan `user_id` dari JWT. Customer tidak dapat memilih customer lain, mengubah assignment teknisi, resolution, atau status operasional.

## Staff routes

- Billing/payment operasional: `root`, `owner`, `admin`, `petugas`, `loket`.
- Network/NOC/provisioning: role network yang sesuai, termasuk alias transisi `teknisi`, `technician`, dan `noc`.
- Finance, settings, WhatsApp, voucher administration, dan user administration: sementara dibatasi ke `root`, `owner`, dan `admin` sampai permission enforcement menggantikannya.
- Grup `/admin_api/*` lainnya memakai staff guard dan selalu menolak role customer.

## Response contract

- Tidak ada atau token tidak valid: HTTP `401` dengan message `unauthorized`.
- Token valid tetapi role tidak diizinkan: HTTP `403` dengan message `forbidden`.

## Known transition constraints

- Role guard sementara masih membaca claim role lama dari JWT.
- `users.role_id` adalah mapping canonical Phase 1, tetapi resolver database baru diterapkan pada Phase 2.
- Sidebar dan tombol frontend baru menjadi permission-driven pada Phase 4/5.
