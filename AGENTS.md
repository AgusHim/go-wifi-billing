# AGENTS.md

## Scope
Instruksi di file ini berlaku untuk seluruh folder `go_wifi_billing/`.

## Ringkasan Proyek
- Backend billing internet berbasis Go.
- HTTP framework: Fiber.
- ORM: GORM.
- Database utama: PostgreSQL (`POSTGRES_URL`), fallback SQLite (`test.db`) jika env kosong.
- Entry point aplikasi: `main.go`.

## Struktur Folder
```text
go_wifi_billing/
|- main.go                  # bootstrap app, wiring dependency, start Fiber
|- go.mod, go.sum           # dependency management
|- .env                     # konfigurasi environment lokal
|- Dockerfile, docker-compose.yml
|
|- controllers/             # layer HTTP handler (request/response)
|- services/                # business logic utama
|- repositories/            # query database via GORM
|- models/                  # definisi model entity + relasi GORM
|- dto/                     # request/response contract (payload API)
|- routes/                  # registrasi route semua controller
|- midlewares/              # middleware Fiber (nama folder memang "midlewares")
|- db/
|  |- db.go                 # init DB + AutoMigrate
|  |- seed/                 # seed data awal
|- lib/                     # helper lintas modul (library internal)
|- utils/                   # utilitas umum
```

## Alur Arsitektur
1. Request masuk ke `controllers/`.
2. Controller parse payload (DTO), validasi dasar, lalu panggil `services/`.
3. Service jalankan business rule dan orkestrasi dependency.
4. Service akses data lewat `repositories/`.
5. Repository berinteraksi dengan DB menggunakan `models/`.
6. Hasil dikembalikan ke controller untuk response JSON.

Prinsip:
- `controllers`: tipis, tanpa business logic berat.
- `services`: tempat utama aturan bisnis.
- `repositories`: fokus ke persistensi data.

## Konvensi Penambahan Fitur
Saat menambah domain baru (contoh: `invoice`), ikuti urutan:
1. Tambah model di `models/invoice.go`.
2. Tambah DTO di `dto/invoice_dto.go`.
3. Tambah repository di `repositories/invoice_repository.go`.
4. Tambah service di `services/invoice_service.go`.
5. Tambah controller di `controllers/invoice_controller.go`.
6. Daftarkan wiring di `main.go`.
7. Daftarkan route di `routes/routes.go`.
8. Pastikan dimasukkan ke `db.AutoMigrate(...)` bila perlu.

## Konvensi Coding
- Jangan panic untuk error input user; kembalikan HTTP 4xx/5xx yang konsisten.
- Error message harus jelas dan bisa dipakai frontend.
- Hindari nil-pointer dereference dari input payload.
- Untuk field opsional FK (`uuid`), gunakan pointer/null jika memang boleh kosong.
- Jika ubah relasi/kolom model, cek dampak FK dan data existing.
- Pertahankan penamaan file per domain: `customer_controller.go`, `customer_service.go`, dst.

## Validasi Sebelum Selesai
- Format kode:
  - `gofmt -w .`
- Compile/test:
  - `go test ./...`
- Jika ada perubahan DB/model:
  - cek `db.AutoMigrate(...)` di `db/db.go`
  - review constraint FK agar tidak trigger error insert/update.

## Common Commands
- Jalankan app: `go run main.go`
- Jalankan test: `go test ./...`
- Rapikan format: `gofmt -w .`
- Rapikan dependency: `go mod tidy`

## Environment Penting
- `POSTGRES_URL`: DSN PostgreSQL.
- `PORT`: port server (default `8080`).
- `MIDTRANS_SERVER_KEY`: konfigurasi payment.
- `WHATSAPP_BOT_URL`: endpoint WhatsApp service.
- `WHATSAPP_API_KEY`: API key untuk header `X-API-KEY` ke WhatsApp service.

## Guardrails
- Jangan edit file di luar scope task tanpa alasan kuat.
- Jangan melakukan destructive command (`reset --hard`, delete massal) tanpa instruksi user.
- Jika menemukan perubahan tak terduga yang bukan bagian task aktif, hentikan dan konfirmasi dulu.
