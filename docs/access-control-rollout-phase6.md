# Access-Control Phase 6: Rollout dan Observability

Dokumen ini adalah runbook deployment RBAC. Seluruh tahap tetap melakukan
autentikasi, validasi user/role aktif, hard-check owner, ownership check, dan
fail-closed untuk route privat yang belum mempunyai policy.

## Konfigurasi

| Variabel | Nilai | Fungsi |
| --- | --- | --- |
| `AUTHZ_ENFORCEMENT_MODE` | `shadow`, `warning`, `enforce` | Mode evaluasi permission. Default aman adalah `enforce`. |
| `AUTHZ_ENFORCED_MODULES` | daftar CSV, mis. `customers,bills` | Modul yang sudah di-enforce saat mode `warning`. |
| `AUTHZ_ROLLBACK_BASELINE` | `true`/`false` | Mengabaikan deny override hanya untuk permission yang memang berasal dari baseline role. |
| `AUTHZ_FORBIDDEN_ALERT_THRESHOLD` | integer, default `20` | Jumlah 403 per route dalam window sebelum alert. |
| `AUTHZ_FORBIDDEN_ALERT_WINDOW` | Go duration, default `5m` | Window dan cooldown alert lonjakan 403. |

Mode `shadow` dan `warning` hanya melonggarkan penolakan permission fitur.
Route owner-only tidak pernah dilonggarkan. Token invalid, user terhapus/nonaktif,
role hilang/nonaktif, dan route tanpa policy tetap ditolak.

## Tahapan rollout

1. Jalankan migration/seed dan preflight role audit. Pastikan minimal satu owner aktif.
2. Aktifkan `AUTHZ_ENFORCEMENT_MODE=shadow`. Pantau decision log dan endpoint
   owner-only `GET /admin_api/access-control/metrics`.
3. Pindah ke `warning`, isi `AUTHZ_ENFORCED_MODULES` dengan satu modul risiko rendah.
   Perluas modul setelah 401/403, support ticket, dan route-policy gap bersih.
4. Jalankan seluruh backend test, frontend access-control test, build, dan staging E2E.
5. Aktifkan `AUTHZ_ENFORCEMENT_MODE=enforce`, kosongkan daftar modul, dan pastikan
   `AUTHZ_ROLLBACK_BASELINE=false`.

Metrics dikelompokkan berdasarkan HTTP method, route pattern, permission, dan status
401/403 agar ID resource tidak membuat cardinality tak terbatas. Metrics dan alert
in-memory di-reset saat proses restart; security alert juga ditulis ke structured log.
Perubahan membership owner menghasilkan alert `owner_change`.

Audit dapat diunduh owner melalui dashboard atau
`GET /admin_api/access-control/audit-logs/export` dengan filter audit yang sama.

## Staging E2E

Gunakan user staging khusus dan permission yang mempunyai probe endpoint aman:

```sh
E2E_API_URL=https://staging.example.com \
E2E_OWNER_TOKEN=... \
E2E_TARGET_TOKEN=... \
E2E_TARGET_USER_ID=... \
E2E_PERMISSION_KEY=customers.read \
E2E_PROBE_URL=/admin_api/customers \
E2E_CONFIRM_MUTATION=MUTATE_AND_RESTORE_ACCESS \
npm run e2e:access-control
```

Script membuktikan owner diterima, non-owner ditolak dari API pengelolaan, mengubah
allow/deny target, memeriksa perubahan pada request berikutnya, lalu mengembalikan
role dan override awal dalam blok `finally`.

## Rollback

Rollback pertama adalah kembali ke `warning` dengan hanya modul yang stabil. Jika
deny override menyebabkan gangguan operasional, set
`AUTHZ_ROLLBACK_BASELINE=true`: user kembali mendapat permission baseline role,
sedangkan permission baru di luar baseline tidak diberikan. Owner-only tetap ketat.

Jangan menghapus role, permission, override, `permission_version`, atau audit log
saat rollback. Setelah incident selesai, perbaiki override/policy melalui dashboard,
validasi metrics, lalu nonaktifkan baseline rollback. Rollback binary/database hanya
dilakukan dengan backup dan migration kompatibel; audit table tidak boleh di-truncate.
