# NOC Production Operations

## Backup

- PostgreSQL: jalankan `pg_dump --format=custom --file=wifi_billing_$(date +%Y%m%d%H%M).dump "$POSTGRES_DSN"` dari host yang memiliki akses database.
- Simpan minimal 7 backup harian, 4 backup mingguan, dan 12 backup bulanan.
- Simpan file backup di storage terpisah dari server aplikasi.
- Backup `.env` production secara terpisah dengan akses terbatas karena berisi secret.

## Restore

- Buat database kosong untuk restore test.
- Jalankan `pg_restore --clean --if-exists --dbname="$POSTGRES_DSN" wifi_billing_YYYYMMDDHHMM.dump`.
- Setelah restore, jalankan aplikasi satu kali agar `AutoMigrate` menambahkan kolom/tabel baru yang kompatibel.
- Validasi minimal:
  - login admin berhasil
  - data router terbaca
  - endpoint `/admin_api/noc/overview` berhasil
  - endpoint `/admin_api/noc/metrics` berhasil

## RouterOS API Hardening

- Gunakan user RouterOS khusus API, bukan user admin utama.
- Permission minimal untuk monitoring: `read`, `api`.
- Permission tambahan untuk runbook write command: `write`, `api`, hanya jika teknisi memang membutuhkan reconnect/disable/enable.
- Aktifkan API-SSL pada router yang mendukung, lalu set `NOC_REQUIRE_ROUTER_TLS=true` di backend.
- Batasi service API/API-SSL RouterOS hanya dari IP backend.
- Rotasi password API secara berkala dan update credential router di aplikasi.

## Runtime Knobs

- `FEATURE_NOC_COLLECTOR_ENABLED=true` untuk mengaktifkan scheduler collector.
- `NOC_COLLECTOR_INTERVAL=5m` untuk interval collection.
- `NOC_COLLECTOR_WORKERS=4` untuk batas worker collector paralel.
- `NOC_ROUTER_COMMAND_INTERVAL_SECONDS=5` untuk rate limit command langsung per router.
- `NOC_REQUIRE_ROUTER_TLS=true` untuk menolak collector/runbook pada router yang belum memakai TLS.

## Retention

- Raw telemetry snapshot disimpan 7 hari.
- Aggregate hourly disimpan 90 hari.
- Aggregate daily disimpan 1 tahun.
- Retention dijalankan otomatis setelah collector selesai.
