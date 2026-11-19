# Internal Helpdesk System

Aplikasi helpdesk internal untuk membuat dan menangani tiket dukungan, dengan tiga peran: **requester**, **agent**, dan **admin**. Backend menggunakan Go + Gin, frontend menggunakan Vue 3 + Vite, dan database menggunakan PostgreSQL.

## Daftar Isi

1. [Cara Menjalankan Aplikasi](#1-cara-menjalankan-aplikasi)
2. [Akun Seed untuk Tiap Peran](#2-akun-seed-untuk-tiap-peran)
3. [Daftar Endpoint API](#3-daftar-endpoint-api)
4. [Cara Menjalankan Test](#4-cara-menjalankan-test)

## 1. Cara Menjalankan Aplikasi

Jalankan perintah dari direktori utama repository, kecuali jika ada instruksi berpindah direktori.

### Opsi A: Docker Compose (direkomendasikan)

**Prasyarat:** Docker Engine aktif dan Docker Compose v2 tersedia. Go, Node.js, dan PostgreSQL tidak perlu dipasang di host untuk opsi ini.

```bash
docker compose up --build -d
```

Compose menjalankan PostgreSQL, backend, dan frontend. Backend otomatis menjalankan migrasi database, lalu menambahkan data seed karena `SEED_ON_START=true`.

| Layanan | Alamat / koneksi |
|---|---|
| Frontend | [http://localhost:3000](http://localhost:3000) |
| Backend API | `http://localhost:8080/api` |
| PostgreSQL dari host | `localhost:5433` |
| Kredensial database demo | Database: `helpdesk`, user: `postgres`, password: `postgres` |

Buka frontend lalu login menggunakan salah satu [akun seed](#2-akun-seed-untuk-tiap-peran). Frontend memakai `/api`, yang diteruskan Nginx ke backend.

Perintah pengelolaan dari direktori utama:

```bash
# Lihat status container
docker compose ps

# Lihat log startup backend, migrasi, dan seeder
docker compose logs --tail=100 backend

# Hentikan dan hapus container tanpa menghapus volume data
docker compose down
```

Database disimpan pada volume `db_data`, sedangkan lampiran pada `uploads_data`. **Jangan gunakan `docker compose down -v` jika data masih diperlukan**, karena opsi `-v` menghapus kedua volume tersebut.

### Opsi B: Pengembangan lokal

**Prasyarat:** Go 1.22 atau lebih baru, Node.js 20 + npm, serta PostgreSQL. Contoh berikut memakai PostgreSQL 16 dari Compose, sementara backend dan frontend berjalan langsung di host. Untuk `go test -race`, Go juga membutuhkan CGO dan compiler C yang kompatibel, misalnya GCC atau Clang.

#### Langkah 1 — Jalankan database

Dari direktori utama:

```bash
docker compose up -d db
```

Tunggu database siap. Perintah berikut seharusnya menampilkan pesan `accepting connections`:

```bash
docker compose exec db pg_isready -U postgres -d helpdesk
```

Jika menggunakan PostgreSQL sendiri, buat database `helpdesk` dan sesuaikan `DATABASE_URL` pada langkah berikutnya.

#### Langkah 2 — Jalankan backend

Buka terminal untuk backend:

```bash
cd backend
cp .env.example .env
```

Sesuaikan nilai pada `backend/.env`, terutama `DATABASE_URL` dan `JWT_SECRET`. Konfigurasi lokal yang digunakan:

```dotenv
PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5433/helpdesk?sslmode=disable
JWT_SECRET=ganti-dengan-secret-acak-yang-kuat
JWT_EXPIRY_HOURS=24
SLA_HOURS=2
SEED_ON_START=true
UPLOAD_DIR=./uploads
```

Ganti nilai contoh `JWT_SECRET` sebelum digunakan. Jalankan backend dari direktori `backend/` agar file `.env` termuat:

```bash
go mod download
go run ./cmd/api
```

API tersedia di `http://localhost:8080/api`. Migrasi berjalan otomatis sebelum server menerima request. Jika seed aktif, akun dan tiket contoh yang belum ada akan ditambahkan.

#### Langkah 3 — Jalankan frontend

Buka terminal lain dari direktori utama:

```bash
cd frontend
npm ci
VITE_API_BASE_URL=/api npm run dev
```

Buka [http://localhost:5173](http://localhost:5173). Perintah di atas menggunakan sintaks shell Linux/macOS. Vite meneruskan request `/api` ke `http://localhost:8080`; target proxy dapat diubah melalui environment proses `VITE_BACKEND_URL`.

Tanpa `VITE_API_BASE_URL`, klien frontend menggunakan URL absolut `http://localhost:8080/api` dan tidak melewati proxy Vite. Untuk build produksi, `VITE_API_BASE_URL` harus ditetapkan **saat build**, bukan ketika container Nginx sudah berjalan.

### Ringkasan konfigurasi backend

| Variabel | Default | Fungsi |
|---|---|---|
| `PORT` | `8080` | Port HTTP backend. |
| `DATABASE_URL` | Wajib diisi | Koneksi PostgreSQL. Dari host gunakan `localhost:5433`; antar-container Compose gunakan `db:5432`. |
| `JWT_SECRET` | Wajib diisi | Kunci penandatanganan token login. |
| `JWT_EXPIRY_HOURS` | `24` | Masa berlaku token dalam jam. |
| `SLA_HOURS` | `2` | Batas umur tiket berprioritas `urgent` dan berstatus `open` sebelum dianggap overdue. |
| `SEED_ON_START` | `true` | Seeder aktif jika nilainya tepat `true`; gunakan `false` untuk menonaktifkannya. |
| `UPLOAD_DIR` | `./uploads` | Direktori penyimpanan lampiran; Compose memakai `/app/uploads`. |
| `ALLOWED_ORIGINS` | `http://localhost:5173` | Dibaca konfigurasi, tetapi belum diterapkan oleh router; CORS saat ini masih memakai origin `*`. |

**Catatan penggunaan:** kredensial dan secret Compose hanya untuk demo/pengembangan. Untuk produksi, ganti kredensial dan secret, nonaktifkan seed, gunakan HTTPS, serta tinjau konfigurasi CORS dan backup database/lampiran. Pada frontend Nginx, `client_max_body_size` belum diatur; default Nginx `1m` dapat menyebabkan upload ditolak dengan HTTP 413 sebelum mencapai batas backend 5 MiB.

## 2. Akun Seed untuk Tiap Peran

Semua akun seed menggunakan password **`Password123!`**.

| Peran | Nama | Email | Password | Hak akses utama |
|---|---|---|---|---|
| `admin` | Admin Helpdesk | `admin@helpdesk.local` | `Password123!` | Melihat seluruh tiket, membuat tiket, mengatur prioritas/assignee/pelapor, mengelola role pengguna, dan melihat statistik admin. |
| `agent` | Agent Helpdesk | `agent@helpdesk.local` | `Password123!` | Melihat seluruh tiket, memproses tiket, mengatur prioritas, serta membuat komentar publik dan catatan internal. Tidak dapat membuat tiket. |
| `requester` | Requester A (Budi) | `requester1@helpdesk.local` | `Password123!` | Membuat tiket, melihat tiket milik sendiri, mengirim komentar/lampiran, serta mengubah status sesuai alur yang diizinkan. |
| `requester` | Requester B (Siti) | `requester2@helpdesk.local` | `Password123!` | Sama seperti requester A, tetapi tidak dapat mengakses tiket milik requester A. |

Seeder berada di `backend/internal/database/seeder.go` dan dijalankan saat startup jika `SEED_ON_START=true`. Pada database kosong, seeder membuat **empat akun dan 13 tiket contoh**.

- Akun yang emailnya sudah ada tidak dibuat ulang; password dan role akun lama **tidak direset**.
- Tiket contoh yang judulnya sudah ada dilewati agar tidak ditambahkan ulang.
- Tombol akun demo pada halaman login hanya mengisi kredensial; pengguna tetap perlu menekan tombol login.
- Pendaftaran melalui `/api/auth/register` selalu membuat akun `requester`, bukan agent atau admin.

## 3. Daftar Endpoint API

Base URL untuk akses langsung ke backend: **`http://localhost:8080/api`**. Jika melalui frontend Compose, gunakan `http://localhost:3000/api`.

Seluruh endpoint selain register dan login membutuhkan JWT:

```http
Authorization: Bearer <token-hasil-login>
```

### Autentikasi

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| POST | `/api/auth/register` | Publik | Mendaftarkan requester baru. Body JSON: `name`, `email`, `password`. |
| POST | `/api/auth/login` | Publik | Login dengan `email` dan `password`; mengembalikan JWT dan data pengguna. |
| GET | `/api/me` | Semua peran login | Mengembalikan profil dari claim JWT, bukan pembacaan ulang data pengguna dari database. |

Contoh login menggunakan akun seed:

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@helpdesk.local","password":"Password123!"}'
```

Token login terdapat pada field `data.token` dalam respons JSON.

### Tiket dan komentar

`{id}` adalah ID tiket yang dituju.

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| GET | `/api/tickets` | Semua peran login | Daftar tiket dengan filter, pagination, pencarian, dan pengurutan. Requester hanya melihat tiket miliknya. |
| POST | `/api/tickets` | Requester, admin | Membuat tiket. Body utama: `title`, `description`, `category`. Prioritas dari requester diabaikan; admin dapat mengirim `priority` dan `requester_id`. |
| GET | `/api/tickets/{id}` | Pemilik tiket, agent, admin | Detail tiket beserta komentar dan lampiran; catatan internal tidak dikirim kepada requester. |
| PATCH | `/api/tickets/{id}` | Sesuai peran dan kepemilikan tiket | Mengubah `status` sesuai transisi; `priority` hanya agent/admin; `assignee_id` dan `requester_id` hanya admin. Tidak menyediakan edit judul, deskripsi, atau kategori. |
| POST | `/api/tickets/{id}/comments` | Pemilik tiket, agent, admin | Menambah komentar dengan `body`; `is_internal: true` hanya untuk agent/admin. |

Parameter query untuk `GET /api/tickets`:

- `page`, `limit`: pagination.
- `status`: filter status tiket.
- `priority`: filter prioritas; `unset` untuk tiket tanpa prioritas.
- `assignee_id`: ID pengguna, `me`, atau `unassigned`.
- `search`: pencarian pada judul tiket.
- `sort_by`: `created_at`, `updated_at`, atau `priority`.
- `order`: `asc` atau `desc`.

Perubahan status mengikuti alur: `open → in_progress` (agent/admin), `open → closed` (requester/admin), `in_progress → resolved` (agent/admin), `resolved → closed` (semua peran), dan `resolved → open` (requester saja). Akses requester tetap terbatas pada tiket miliknya. Saat agent memulai proses, tiket otomatis ditugaskan kepada agent tersebut; perubahan oleh admin tidak melakukan auto-assignment.

### Lampiran dan realtime

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| POST | `/api/tickets/{id}/attachments` | Pemilik tiket, agent, admin | Upload `multipart/form-data` dengan field `file`; batas backend 5 MiB dan whitelist ekstensi. |
| GET | `/api/attachments/{id}/download` | Pemilik tiket terkait, agent, admin | Mengunduh lampiran; `{id}` di sini adalah ID lampiran, bukan ID tiket. |
| GET | `/api/events` | Semua peran login | Stream Server-Sent Events (SSE), dengan penerima event dibatasi berdasarkan peran dan kepemilikan tiket. |

Browser menggunakan `?token=...` untuk SSE dan unduhan karena alur tersebut tidak memakai interceptor Axios. Utamakan header Bearer untuk klien yang mendukungnya; token pada URL berpotensi masuk ke log atau riwayat. Lampiran terhubung ke tiket, bukan komentar: file yang dikirim bersama catatan internal tetap dapat diakses requester pemilik tiket.

### Statistik dan pengguna

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| GET | `/api/stats` | Semua peran login | Ringkasan jumlah tiket per status dan urgent aktif; requester hanya memperoleh statistik tiket miliknya. Field `overdue_tickets` pada endpoint ini belum dihitung dan bernilai nol. |
| GET | `/api/tickets/stats` | Semua peran login | Alias `/api/stats`. |
| GET | `/api/admin/stats` | Admin | Statistik operasional, overdue, distribusi kategori/prioritas, dan beban kerja agent. |
| GET | `/api/users` | Agent, admin | Admin dapat melihat seluruh pengguna dan memakai filter `role`; agent hanya menerima daftar agent/admin. |
| PATCH | `/api/users/{id}/role` | Admin | Mengubah peran pengguna melalui body `{"role":"agent"}`, misalnya. `{id}` adalah ID pengguna. |

Registrasi route dapat diperiksa di `backend/internal/handler/router.go`. Belum tersedia endpoint hapus tiket, edit/hapus komentar, reset password, refresh token, atau logout/revoke token di server. Perubahan role tidak memperbarui claim token lama; login ulang diperlukan untuk memperoleh token dengan role terbaru.

## 4. Cara Menjalankan Test

### Prasyarat dan cakupan

Test backend menggunakan Go, repository mock, dan HTTP test server dalam proses. **Tidak perlu menjalankan PostgreSQL, backend, frontend, Docker Compose, atau menyiapkan `.env` untuk menjalankan suite ini.** Dependensi Go perlu tersedia; pengunduhan pertama membutuhkan koneksi internet.

Test mencakup autentikasi, isolasi tiket antar-requester, aturan peran, transisi status, catatan internal, lampiran, SLA, endpoint HTTP, dan pembatasan penerima event realtime. Suite ini **bukan pengujian integrasi query SQL terhadap PostgreSQL nyata**.

### Jalankan seluruh test backend

Dari direktori utama:

```bash
cd backend
go test -v ./...
```

Untuk pemeriksaan tambahan seperti yang dijalankan CI:

```bash
# Jalankan dari backend/
go vet ./...
go test -v -race ./...
```

Opsi `-race` memerlukan CGO aktif serta compiler C yang kompatibel. Jika toolchain lokal belum tersedia, jalankan test biasa terlebih dahulu; hasil tersebut belum mencakup pemeriksaan race.

### Jalankan test tertentu dan lihat coverage

Semua perintah berikut dijalankan dari `backend/`:

```bash
# Seluruh test endpoint HTTP
go test -v ./tests -run '^TestAPI_'

# Isolasi data tiket antar-requester
go test -v ./tests -run '^TestTicketIsolation$'

# Pembatasan penerima notifikasi realtime
go test -v ./tests -run '^TestBrokerAudienceScoping$'

# Ringkasan coverage kode aplikasi backend
go test -coverpkg=./internal/... ./tests
```

### Validasi frontend

Frontend belum memiliki script unit test atau E2E test. Validasi yang tersedia adalah build produksi, sama seperti langkah frontend di CI.

Buka terminal dari direktori utama:

```bash
cd frontend
npm ci
npm run build
```

Build menghasilkan direktori `frontend/dist`. **Build berhasil bukan berarti perilaku antarmuka sudah diuji di browser**, dan script ini tidak menjalankan typecheck terpisah.

Workflow `.github/workflows/ci.yml` menjalankan `go vet`, backend test dengan `-race`, serta instalasi dependensi dan build frontend pada push/pull request ke branch `main` atau `master`.
