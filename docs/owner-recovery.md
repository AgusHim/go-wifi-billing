# Emergency Owner Recovery

Recovery ini hanya digunakan ketika tidak ada owner aktif. CLI menolak berjalan
jika owner aktif masih tersedia, target/operator tidak aktif, alasan kosong, owner
role tidak tersedia, atau konfirmasi eksplisit belum diberikan.

## Checklist sebelum eksekusi

1. Buka incident ticket dan catat penyebab hilangnya akses owner.
2. Ambil backup/snapshot database dan pastikan restore point berhasil dibuat.
3. Lakukan peer review atas email target dan operator.
4. Pastikan target adalah user aktif yang identitasnya telah diverifikasi.
5. Jalankan dari host administrasi dengan `POSTGRES_URL` production yang eksplisit.

## Eksekusi

```sh
POSTGRES_URL='postgres://...' \
RECOVERY_OWNER_EMAIL='target@example.com' \
RECOVERY_OPERATOR_EMAIL='operator@example.com' \
RECOVERY_REASON='INC-2026-071: owner terkunci setelah rotasi akun' \
RECOVERY_CONFIRM=RECOVER_OWNER \
go run ./cmd/rbac-owner-recovery
```

Jika operator adalah target yang sama, `RECOVERY_OPERATOR_EMAIL` boleh dikosongkan.
CLI tidak pernah memakai fallback SQLite. Promosi role, penghapusan override target,
kenaikan `permission_version`, dan audit `owner_recovered_via_cli` terjadi dalam satu
transaksi. Kegagalan audit membatalkan seluruh perubahan.

## Verifikasi dan penutupan

1. Simpan output `user_id`, `operator_user_id`, `permission_version`, dan
   `audit_log_id` pada incident ticket.
2. Login sebagai owner yang dipulihkan dan verifikasi
   `/admin/access-control`, daftar permission, serta audit log.
3. Pastikan entry audit mempunyai actor, target, alasan incident, before/after,
   request ID `owner-recovery-cli`, dan waktu yang benar.
4. Rotasi credential/token yang terdampak dan buat owner cadangan aktif sesuai
   kebijakan organisasi.
5. Tutup incident hanya setelah backup, audit export, dan akar masalah tersimpan.

Jangan mempromosikan owner dengan `UPDATE users` manual karena jalur tersebut tidak
menjamin invariant, version increment, penghapusan override, dan audit atomik.
