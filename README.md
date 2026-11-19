# Internal Helpdesk System

Aplikasi web helpdesk internal modern untuk manajemen tiket dukungan TI dan operasional perusahaan. Dibangun menggunakan arsitektur Clean/Layered Backend di atas Go dan Frontend reaktif Vue 3.


---

## 1. Tech Stack

- **Monorepo:** Berisi service `backend/` dan `frontend/` dalam satu repository.
- **Backend:** Go 1.22 (build image `golang:1.23-alpine`), Gin v1.10, arsitektur berlapis (`handler` → `service` → `repository`, dengan entity dan kontrak interface di `domain`).
  - Auth: JWT (`golang-jwt/jwt/v5`), bcrypt, middleware `Authorization: Bearer`.
  - Data: PostgreSQL (`lib/pq`), migrasi SQL + seeder otomatis (`SEED_ON_START=true`).
  - Validasi: `go-playground/validator/v10`; konfigurasi: `godotenv`.
- **Frontend:** Vue 3.5 (Composition API `<script setup>`), Vite 5, Vue Router 4 (navigation guards), Pinia 2 (stores: `auth`, `settings`), Axios 1.7 (interceptor 401), Tailwind CSS 3.4 (`darkMode: 'class'`).
  - **i18n:** modul lokal `src/i18n/index.js` (kamus ID/EN + `useI18n()` / `t(key)`) untuk label dan pesan antarmuka. Sebagian label masih tetap, dan format tanggal masih memakai locale `id-ID`.
  - **Theme:** mode gelap/terang via store Pinia `stores/settings.js`, persisten di `localStorage`, diterapkan sebelum paint pertama (`index.html` inline script) + `src/dark.css` untuk remap global utilitas Tailwind.
  - **Toolbar atas (navbar):** tombol fullscreen, switch bahasa ID/EN, switch tema, dan dropdown avatar user.
  - **Loading bar:** indikator progress tipis di bagian paling atas halaman saat perpindahan route.
- **Database:** PostgreSQL 16 (Container port host `5433:5432`).
- **Containerization:** Docker multi-stage build (Go build → Alpine 3.20 non-root, Node 20 build → Nginx 1.27 serve) + Docker Compose.

---

## 2. Akun Demo Bawaan (Database Seeder)

Semua akun menggunakan password yang sama: **`Password123!`**.

| Label / Nama | Email | Role | Hak Akses Utama |
|---|---|---|---|
| **Admin Helpdesk** | `admin@helpdesk.local` | `admin` | Kelola role user, assignee, pelapor, prioritas, dan statistik; perubahan status tetap mengikuti matriks transisi |
| **Agent Helpdesk** | `agent@helpdesk.local` | `agent` | Mengerjakan tiket (open→in_progress→resolved), auto-assign diri saat proses, mengatur prioritas tiket, internal note |
| **Requester A (Budi)** | `requester1@helpdesk.local` | `requester` | Buat tiket, pantau tiket milik sendiri, reopen (resolved→open), cancel tiket (open→closed) |
| **Requester B (Siti)** | `requester2@helpdesk.local` | `requester` | Terisolasi dari Requester A |

*Tombol Demo Accounts di halaman `/login` mengisi kredensial; pengguna tetap menekan tombol login. Seeder menambahkan akun yang belum ada, bukan mereset password/role akun yang sudah tersimpan. Akun demo dan kredensial Compose hanya untuk pengembangan.*

---

## 3. Aturan Bisnis & RBAC (Role-Based Access Control)

### 3.1. Pembuatan & Kepemilikan Tiket
- **Requester & Admin:** Dapat membuat tiket baru (`POST /api/tickets`).
- **Agent:** **Dilarang** membuat tiket baru (tombol disembunyikan di UI, route guarded, backend menolak dengan `403 Forbidden`).
- **Isolasi Data Requester:** Requester hanya dapat membaca tiket miliknya sendiri (`WHERE requester_id = current_user.id`).

### 3.2. Properti Tiket (Assignee & Prioritas)
- **Prioritas (Priority):**
  - Prioritas opsional saat pembuatan. Form create tidak menyediakan field prioritas; prioritas yang dikirim requester melalui API diabaikan, sedangkan admin dapat menentukannya melalui API.
  - Baik **Agent** maupun **Admin** dapat menentukan dan memperbarui prioritas tiket kapan saja via antarmuka detail tiket (`PATCH /api/tickets/{id}`).
  - Requester tidak dapat mengubah prioritas melalui PATCH; pengiriman nilai prioritas non-null ditolak `403 Forbidden`.
- **Assignee:**
  - Saat Agent mengubah status dari `open` ke `in_progress`, tiket **otomatis di-assign** ke Agent tersebut. Transisi oleh admin tidak melakukan auto-assignment.
  - Role Agent **tidak memiliki dropdown** Assignee (hanya-baca / read-only). Jika dicoba kirim via API, backend menolak dengan `403 Forbidden`.
  - Hanya **Admin** yang memiliki dropdown untuk mengubah Assignee secara manual. Untuk melepas assignee melalui API, kirim `{"assignee_id":""}` atau `{"assignee_id":"unassigned"}`; JSON `null` dianggap tidak mengubah field.
- **Pelapor (Requester):**
  - Admin dapat membuat tiket atas nama user lain (`requester_id` pada `POST /api/tickets`) dan mengubah pelapor tiket (`requester_id` pada `PATCH /api/tickets/{id}`).
  - Perubahan pelapor melalui PATCH hanya untuk admin; requester/agent yang mengirim `requester_id` non-null ditolak `403 Forbidden`. Saat create oleh requester, `requester_id` kiriman diabaikan dan pelapor tetap pengguna tersebut.
- **Kolom Requester di Tabel (`/tickets`):**
  - Hanya muncul pada role **Agent** dan **Admin** (menampilkan Nama dan Email sekaligus dalam 1 kolom).
  - Disembunyikan untuk role Requester.

### 3.3. Matriks Transisi Status (State Machine)

| Status Asal (`current`) | Status Tujuan (`target`) | Role yang Diizinkan | Keterangan |
|---|---|---|---|
| `open` | `in_progress` | `agent`, `admin` | Mulai memproses; auto-assignment hanya jika aktornya agent |
| `open` | `closed` | `requester`, `admin` | Pembatalan tiket oleh pembuat atau admin |
| `in_progress` | `resolved` | `agent`, `admin` | Penanganan masalah selesai |
| `resolved` | `closed` | `requester`, `agent`, `admin` | Konfirmasi tiket ditutup permanen |
| `resolved` | `open` | `requester` | Tiket dibuka kembali oleh requester jika kendala berulang |

*Transisi di luar matriks ini ditolak backend dengan status HTTP `422 Unprocessable Entity`. Mengirim status yang sama bukan transisi baru. Admin tidak dapat reopen tiket; `closed` tidak memiliki transisi keluar, tetapi komentar, lampiran, dan perubahan metadata belum dibatasi berdasarkan status tersebut.*

### 3.4. Catatan Internal (Internal Notes)
- Komentar dengan `is_internal = true` hanya dapat dibuat dan dibaca oleh `agent` dan `admin`.
- Requester tidak menerima isi catatan internal pada payload detail tiket (`GET /api/tickets/{id}`). Event `comment_added` tetap dikirim ke pemilik tiket, tetapi hanya berisi `ticket_id`.
- Lampiran terhubung ke tiket, bukan komentar, dan tidak memiliki flag internal. File yang dikirim bersama internal note tetap dapat diakses requester pemilik tiket.

---

### 3.5. Navigasi & Halaman Default Berdasarkan Role
- **Admin:**
  - Halaman default/landing saat login atau mengakses root URL (`/`) adalah **Statistik Admin** (`/admin/stats`).
  - Menu **Statistik** ditempatkan pada posisi teratas di sidebar desktop dan drawer mobile.
- **Agent & Requester:**
  - Karena tidak memiliki akses ke halaman statistik khusus, halaman default saat login maupun root URL (`/`) adalah **Daftar Tiket** (`/tickets`).

---

## 4. Fitur Tambahan & Bonus (Bonus Features)

1. **Lampiran Berkas (File Attachments):**
   - Handler backend membatasi ukuran file hingga 5 MiB (5.242.880 byte) dan memakai whitelist ekstensi: JPG/JPEG, PNG, GIF, WEBP, PDF, TXT, CSV, DOC/DOCX, XLS/XLSX, ZIP.
   - MIME type ditetapkan dari ekstensi; isi file belum diperiksa melalui MIME sniffing atau antivirus.
   - Enkapsulasi nama berkas dengan UUID pada storage server.
   - Endpoint `POST /api/tickets/{id}/attachments` dan `GET /api/attachments/{id}/download`.
   - UI menyediakan lampiran saat membuat tiket maupun mengirim komentar/balasan, tetapi upload merupakan request terpisah. Tiket/komentar dapat berhasil tersimpan meskipun upload gagal.
2. **Notifikasi Real-Time Berbasis Role & Kepemilikan (SSE with Audience Scoping):**
   - In-memory event broker tanpa dependensi eksternal via endpoint `GET /api/events`.
   - Event `ticket_created` **hanya dikirimkan dan ditampilkan kepada `agent` dan `admin`** (tidak bocor ke requester lain).
   - Event `ticket_updated`, `comment_added`, dan `attachment_added` discoped khusus untuk staf (`agent`, `admin`) dan pembuat tiket terkait (`requester_id`).
   - Event memicu toast serta refetch daftar/detail yang berlangganan. Halaman statistik admin belum melakukan auto-refetch.
   - Broker hanya berada di memori satu proses, tanpa replay; event dapat dibuang jika buffer klien penuh dan belum tersinkron lintas instance backend.
3. **Pelacakan SLA (Service Level Agreement):**
   - Tiket overdue hanya jika berprioritas `urgent`, berstatus `open`, dan umur sejak `created_at` melebihi `SLA_HOURS` (default **2 jam**).
   - Belum ada SLA terpisah untuk high/medium/low, histori breach, atau eskalasi otomatis. Reopen tetap menggunakan waktu pembuatan awal.
   - Badge **Terlambat / Overdue** dihitung saat daftar/detail dimuat, bukan melalui timer atau job background.
4. **Halaman Statistik Admin Terperinci:**
   - Menampilkan jumlah tiket total, open, in progress, resolved, closed, dan overdue, distribusi kategori/prioritas, serta beban kerja agent.
   - Kategori mencakup seluruh tiket. Distribusi prioritas mencakup tiket bukan closed yang prioritasnya terisi, termasuk resolved. Workload hanya menghitung agent dengan tiket bukan resolved/closed.
   - Belum ada metrik khusus resolution rate atau SLA breach rate.
   - Menjadi default landing route untuk akun `admin`, ditempatkan di posisi teratas navigasi sidebar.
   - Endpoint khusus `GET /api/admin/stats` dengan proteksi role `admin`.
5. **Pengaturan Pelapor Tiket oleh Admin:**
   - Admin dapat memilih requester saat membuat tiket baru atas nama pengguna lain.
   - Admin dapat mengubah requester pada tiket yang sudah berjalan langsung dari halaman detail tiket.
6. **Continuous Integration (GitHub Actions):**
   - Workflow `.github/workflows/ci.yml` berjalan pada push/pull request ke `master` dan `main`: backend memakai `go vet ./...` dan `go test -v -race ./...`; frontend memakai `npm ci` dan `npm run build`.
   - Belum ada deployment otomatis (CD) atau langkah typecheck frontend terpisah.

---

## 5. Cara Menjalankan Aplikasi

### Opsi A: Menggunakan Docker Compose (Direkomendasikan)

Pastikan Docker Engine aktif:

```bash
docker compose up --build
```

Layanan yang berjalan:
- **Frontend:** [http://localhost:3000](http://localhost:3000) (Nginx reverse-proxy ke `/api`)
- **Backend API:** [http://localhost:8080](http://localhost:8080)
- **PostgreSQL:** `localhost:5433` (DB: `helpdesk`, user: `postgres`, pass: `postgres`)

### Opsi B: Pengembangan Lokal (Bare-metal)

1. **Jalankan Database PostgreSQL (via Docker):**
   ```bash
   docker compose up -d db
   ```
   *PostgreSQL akan terekspos di host port `5433`.*

2. **Jalankan Backend (Go):**
   ```bash
   cd backend
   cp .env.example .env
   # Edit .env: DATABASE_URL dan JWT_SECRET wajib terisi.
   # DATABASE_URL=postgres://postgres:postgres@localhost:5433/helpdesk?sslmode=disable
   # Gunakan JWT_SECRET acak yang kuat, bukan nilai demo Compose.
   go run ./cmd/api
   ```
   *Server berjalan di `http://localhost:8080`. Migrasi dijalankan pada startup; jika `SEED_ON_START=true` (default), seeder menambahkan empat akun dan 13 tiket sampel yang belum ada.*

3. **Jalankan Frontend (Vite Dev Server):**
   ```bash
   cd frontend
   npm ci
   VITE_API_BASE_URL=/api npm run dev
   ```
   *Aplikasi dapat diakses di `http://localhost:5173`. Base URL `/api` memakai proxy Vite ke `http://localhost:8080`; target proxy dapat diubah melalui environment proses `VITE_BACKEND_URL`. Tanpa `VITE_API_BASE_URL`, klien memakai URL absolut `http://localhost:8080/api` dan melewati proxy.*

### Konfigurasi Backend

| Variabel | Default | Keterangan |
|---|---|---|
| `PORT` | `8080` | Port HTTP server. |
| `DATABASE_URL` | Wajib diisi | DSN PostgreSQL; gunakan `localhost:5433` dari host atau `db:5432` dari container Compose. |
| `JWT_SECRET` | Wajib diisi | Kunci penandatanganan token. |
| `JWT_EXPIRY_HOURS` | `24` | Masa berlaku JWT dalam jam. |
| `SLA_HOURS` | `2` | Batas umur urgent-open sebelum overdue. |
| `SEED_ON_START` | `true` | Seed aktif hanya jika nilai efektif sama dengan string `true`. |
| `UPLOAD_DIR` | `./uploads` | Direktori file; Compose memakai `/app/uploads` dengan volume `uploads_data`. |
| `ALLOWED_ORIGINS` | `http://localhost:5173` | Dibaca config tetapi belum diterapkan; router masih mengirim CORS origin `*`. |

`VITE_API_BASE_URL` frontend ditetapkan saat build, bukan saat container Nginx mulai berjalan. Database dan file upload disimpan pada volume terpisah; backup keduanya dan hindari `docker compose down -v` jika data masih diperlukan.

---

## 6. Ringkasan API Endpoints

### Autentikasi
- `POST /api/auth/register` – Pendaftaran Requester baru (default role: `requester`)
- `POST /api/auth/login` – Login & penerbitan Bearer Token JWT
- `GET  /api/me` – Profil dari claim JWT, bukan lookup user terbaru di database

### Tiket & Komentar
- `GET   /api/tickets` – Daftar tiket dengan `page`, `limit`, `status`, `priority` (termasuk `unset`), `assignee_id` (termasuk `me`/`unassigned`), `search` pada judul, `sort_by` (`created_at`/`updated_at`/`priority`), dan `order`
- `POST  /api/tickets` – Buat tiket baru (khusus `requester` & `admin`)
- `GET   /api/tickets/{id}` – Detail tiket beserta komentar dan lampiran sesuai hak akses
- `PATCH /api/tickets/{id}` – Update status; prioritas oleh agent/admin; assignee & requester oleh admin. Tidak menyediakan edit judul, deskripsi, atau kategori
- `POST  /api/tickets/{id}/comments` – Tambah komentar (mendukung `is_internal: true` untuk agent/admin)

### Lampiran & Realtime
- `POST /api/tickets/{id}/attachments` – Unggah berkas lampiran (multipart/form-data, field `file`, maks 5 MiB pada handler backend)
- `GET  /api/attachments/{id}/download` – Unduh berkas lampiran (validasi otorisasi tiket)
- `GET  /api/events` – Server-Sent Events (SSE) stream notifikasi real-time

### Statistik (Role-Based)
- `GET /api/stats` – Jumlah tiket total/per status dan urgent aktif (bukan resolved/closed); requester hanya melihat miliknya. Field `overdue_tickets` belum dihitung pada endpoint ini dan bernilai nol
- `GET /api/tickets/stats` – Alias statistik ringkasan tiket
- `GET /api/admin/stats` – Statistik komprehensif operasional & beban kerja (khusus role `admin`)

### Pengguna (Agent/Admin)
- `GET   /api/users` – Admin melihat seluruh user dengan filter `role` opsional; agent hanya menerima daftar agent/admin; requester ditolak
- `PATCH /api/users/{id}/role` – Khusus admin; mencegah admin terakhir menurunkan role dirinya sendiri pada alur biasa, tetapi belum memakai lock/transaksi untuk perubahan serentak

Belum tersedia endpoint hapus tiket, edit/hapus komentar, reset password, refresh token, atau logout/revoke token server.

---

## 7. Pengujian Otomatis (Automated Tests)

Suite backend menguji service, state machine, RBAC, broker realtime, serta router/middleware/handler HTTP dengan repository mock. Test ini tidak menjalankan migrasi atau query repository terhadap PostgreSQL nyata:

```bash
cd backend
go test -v -race ./...
```

Terdapat 18 fungsi test top-level:
1. `TestAPI_AuthEndpoints` – Pendaftaran, login, dan akses `/api/me` memakai token
2. `TestAPI_TicketsEndpoints` – Create, list, detail, update tiket, serta pembatasan role; bukan operasi delete
3. `TestAPI_CommentsEndpoint` – Komentar publik dan proteksi catatan internal
4. `TestAPI_AttachmentsUploadAndDownload` – Upload multipart, download berkas, dan penolakan ekstensi `.exe`
5. `TestAPI_StatsEndpoints` – Otorisasi endpoint statistik umum dan statistik admin
6. `TestAPI_UsersEndpoints` – Manajemen pengguna dan proteksi hak akses role admin
7. `TestAPI_EventsEndpoint` – Koneksi SSE streaming `/api/events`
8. `TestAuthRegisterAndLogin` – Pendaftaran dan autentikasi JWT di layer service
9. `TestTicketIsolation` – Isolasi data tiket antar-requester
10. `TestValidStatusTransitions` – Alur siklus hidup tiket yang sah
11. `TestInvalidStatusTransitions` – Penolakan transisi ilegal di state machine
12. `TestInternalNoteSecurity` – Pencegahan kebocoran catatan internal ke requester
13. `TestAgentCannotCreateTicket` – Verifikasi larangan role agent membuat tiket
14. `TestAgentCanManagePriority` – Agent dapat mengatur prioritas tiket yang awalnya kosong; agent tetap tidak dapat mengubah assignee
15. `TestAdminCanChangeRequester` – Admin dapat mengubah pelapor tiket; requester/agent ditolak
16. `TestSLAOverdueCalculation` – Tiket urgent-open berumur tiga jam menjadi overdue dengan SLA dua jam
17. `TestAttachmentUploadAndAccess` – Penyimpanan metadata dan isolasi akses lampiran antar-requester pada service
18. `TestBrokerAudienceScoping` – Agent menerima `ticket_created`, requester tidak; requester menerima `comment_added` yang ditujukan ke ID-nya

Mock belum meniru seluruh filter, pagination, dan agregasi SQL. Daftar test di atas bukan jaminan cakupan seluruh skenario keamanan atau integrasi database.

Validasi tambahan:

```bash
# Dari backend/
go vet ./...

# Dari frontend/
npm ci
npm run build
```

Frontend belum memiliki script unit/E2E test; build Vite bukan pengujian perilaku browser atau typecheck terpisah.

---

## 8. Struktur Direktori (Monorepo)

```
simple-journey-indonesia/
├── backend/
│   ├── cmd/api/                 # Entrypoint server (main.go)
│   ├── internal/
│   │   ├── config/              # Konfigurasi via environment variables
│   │   ├── database/            # Koneksi, skrip migrasi SQL & seeder
│   │   ├── domain/              # Entities, DTO, interfaces, dan custom errors
│   │   ├── handler/             # HTTP controllers (Gin)
│   │   ├── middleware/          # Auth JWT & helper role guard (CORS ada di router)
│   │   ├── realtime/            # Broker SSE in-memory dengan audience scoping
│   │   ├── repository/          # PostgreSQL data access layer
│   │   └── service/             # Core business logic & state machine
│   ├── tests/                   # Test service, HTTP, dan broker dengan repository mock
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/shared/   # StatusBadge, PriorityBadge, ConfirmDialog, Toast, dll
│   │   ├── constants/           # Definisi status & matriks transisi
│   │   ├── i18n/                # Kamus Indonesia/Inggris dan useI18n()
│   │   ├── router/              # Vue Router & Navigation Guards
│   │   ├── services/            # Axios API wrappers
│   │   ├── stores/              # Pinia auth dan settings
│   │   └── views/               # Login, Register, Tickets, CreateTicket, Detail, Users, AdminStats
│   ├── Dockerfile
│   ├── nginx.conf
│   └── package.json
├── docs/                        # Dokumentasi arsitektur & Postman collection
├── docker-compose.yml
└── README.md
```

---

## 9. Batasan Implementasi & Catatan Produksi

- **Sesi dan perubahan role:** middleware dan `/api/me` menggunakan claim JWT tanpa lookup database. Token lama mempertahankan role lama hingga kedaluwarsa; login ulang menerbitkan token dengan role terbaru. Logout browser tidak mencabut token di server.
- **Lampiran melalui Nginx:** `frontend/nginx.conf` belum menetapkan `client_max_body_size`. Default Nginx `1m` dapat menolak upload dengan 413 sebelum batas 5 MiB backend; periksa konfigurasi efektif proxy dan sesuaikan batas beserta overhead multipart.
- **Unassign melalui UI:** pilihan kosong saat ini dikirim sebagai `null`, sehingga backend tidak melepas assignee. API menerima string kosong atau `unassigned` sebagai alternatif.
- **Konsistensi data:** update tiket masih read-modify-write tanpa version/row lock. Create+upload dan komentar+upload bukan transaksi tunggal.
- **Validasi:** nilai role/prioritas belum sepenuhnya dibatasi whitelist backend atau CHECK database. Opsi UI bukan pengganti validasi API.
- **Kesiapan produksi:** ganti secret/kredensial demo, nonaktifkan seed, gunakan HTTPS, terapkan whitelist CORS yang benar, dan tinjau rate limiting, validasi isi file, backup, serta observability. Konfigurasi demo bukan konfigurasi produksi yang sudah diperkeras.
