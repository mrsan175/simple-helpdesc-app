# Simple Journey Indonesia – Internal Helpdesk System

Aplikasi web helpdesk internal modern untuk manajemen tiket dukungan TI dan operasional PT Simple Journey Indonesia. Dibangun menggunakan arsitektur Clean/Layered Backend di atas Go dan Frontend reaktif Vue 3.

---

## 1. Tech Stack

- **Monorepo:** Berisi service `backend/` dan `frontend/` dalam satu repository.
- **Backend:** Go 1.22 (build image `golang:1.23-alpine`), Gin v1.10, Clean Layered Architecture (`handler` → `service` → `repository` → `domain`).
  - Auth: JWT (`golang-jwt/jwt/v5`), bcrypt, middleware `Authorization: Bearer`.
  - Data: PostgreSQL (`lib/pq`), migrasi SQL + seeder otomatis (`SEED_ON_START=true`).
  - Validasi: `go-playground/validator/v10`; konfigurasi: `godotenv`.
- **Frontend:** Vue 3.5 (Composition API `<script setup>`), Vite 5, Vue Router 4 (navigation guards), Pinia 2 (stores: `auth`, `settings`), Axios 1.7 (interceptor 401), Tailwind CSS 3.4 (`darkMode: 'class'`).
  - **i18n:** module lokal `src/i18n/index.js` (kamus ID/EN + `useI18n()` / `t(key)`), mencakup seluruh UI — bukan hanya chrome.
  - **Theme:** mode gelap/terang via toko Pinia `stores/settings.js`, persisten di `localStorage`, diterapkan sebelum paint pertama (`index.html` inline script) + `src/dark.css` untuk remap global utilitas Tailwind.
  - **Toolbar atas (navbar):** tombol fullscreen, switch bahasa ID/EN, switch tema, info user — sticky di atas sidebar & konten.
- **Database:** PostgreSQL 16 (Container port host `5433:5432`).
- **Containerization:** Docker multi-stage build (Go distroless-like alpine, Node 20 build → Nginx 1.27 serve) + `docker-compose`.

---

## 2. Akun Demo Bawaan (Database Seeder)

Semua akun menggunakan password yang sama: **`Password123!`**.

| Label / Nama | Email | Role | Hak Akses Utama |
|---|---|---|---|
| **Admin Simple Journey** | `admin@simplejourney.id` | `admin` | Semua hak akses: kelola role user, ubah assignee/prioritas tiket, full state machine |
| **Agent Simple Journey** | `agent@simplejourney.id` | `agent` | Mengerjakan tiket (open→in_progress→resolved), auto-assign diri saat proses, internal note |
| **Requester A (Budi)** | `requester1@simplejourney.id` | `requester` | Buat tiket, pantau tiket milik sendiri, reopen (resolved→open), cancel tiket (open→closed) |
| **Requester B (Siti)** | `requester2@simplejourney.id` | `requester` | Terisolasi dari Requester A |

*Tersedia tombol login cepat (Demo Accounts) di halaman `/login`.*

---

## 3. Aturan Bisnis & RBAC (Role-Based Access Control)

### 3.1. Pembuatan & Kepemilikan Tiket
- **Requester & Admin:** Dapat membuat tiket baru (`POST /api/tickets`).
- **Agent:** **Dilarang** membuat tiket baru (tombol disembunyikan di UI, route guarded, backend menolak dengan `403 Forbidden`).
- **Isolasi Data Requester:** Requester hanya dapat membaca tiket miliknya sendiri (`WHERE requester_id = current_user.id`).

### 3.2. Properti Tiket (Assignee & Prioritas)
- **Assignee:**
  - Saat Agent mengubah status dari `open` ke `in_progress`, tiket **otomatis di-assign** ke Agent tersebut.
  - Role Agent **tidak memiliki dropdown** Assignee & Prioritas (hanya-baca / read-only). Jika dicoba kirim via API, backend menolak dengan `403 Forbidden`.
  - Hanya **Admin** yang memiliki dropdown untuk mengubah Assignee dan Prioritas secara manual.
- **Pelapor (Requester):**
  - Admin dapat membuat tiket atas nama user lain (`requester_id` pada `POST /api/tickets`) dan mengubah pelapor tiket (`requester_id` pada `PATCH /api/tickets/{id}`).
  - Requester & Agent tidak dapat mengubah pelapor (ditolak `403 Forbidden`).
- **Kolom Requester di Tabel (`/tickets`):**
  - Hanya muncul pada role **Agent** dan **Admin** (menampilkan Nama dan Email sekaligus dalam 1 kolom).
  - Disembunyikan untuk role Requester.

### 3.3. Matriks Transisi Status (State Machine)

| Status Asal (`current`) | Status Tujuan (`target`) | Role yang Diizinkan | Keterangan |
|---|---|---|---|
| `open` | `in_progress` | `agent`, `admin` | Agent memproses tiket (otomatis menjadi assignee) |
| `open` | `closed` | `requester`, `admin` | Pembatalan tiket oleh pembuat atau admin |
| `in_progress` | `resolved` | `agent`, `admin` | Penanganan masalah selesai |
| `resolved` | `closed` | `requester`, `agent`, `admin` | Konfirmasi tiket ditutup permanen |
| `resolved` | `open` | `requester` | Tiket dibuka kembali oleh requester jika kendala berulang |

*Transisi di luar matriks ini ditolak backend dengan status HTTP `422 Unprocessable Entity`.*

### 3.4. Catatan Internal (Internal Notes)
- Komentar dengan `is_internal = true` hanya dapat dibuat dan dibaca oleh `agent` dan `admin`.
- Requester tidak menerima catatan internal pada payload detail tiket (`GET /api/tickets/{id}`).

---

### 3.5. Navigasi & Halaman Default Berdasarkan Role
- **Admin:**
  - Halaman default/landing saat login atau mengakses root URL (`/`) adalah **Statistik Admin** (`/admin/stats`).
  - Menu **Statistik** ditempatkan pada posisi teratas di sidebar desktop dan drawer mobile.
- **Agent & Requester:**
  - Karena tidak memiliki akses ke halaman statistik khusus, halaman default saat login maupun root URL (`/`) adalah **Daftar Tiket** (`/tickets`).

---

## 4. Fitur Bonus (Bonus Features)

1. **Lampiran Berkas (File Attachments):**
   - Mendukung unggah berkas hingga 5MB pada tiket dengan validasi tipe berkas (ekstensi gambar, PDF, dokumen aman) dan enkapsulasi UUID nama file pada storage server.
   - Endpoint `POST /api/tickets/{id}/attachments` dan `GET /api/attachments/{id}/download`.
   - Berkas dapat dilampirkan saat membuat tiket maupun saat mengirim komentar.
2. **Notifikasi Real-Time (Server-Sent Events / SSE):**
   - In-memory event broker tanpa dependensi eksternal via endpoint `GET /api/events`.
   - Fan-out notifikasi instan untuk tiket dibuat, tiket diupdate, komentar baru, dan lampiran baru, langsung memperbarui data di antarmuka pengguna tanpa refresh.
3. **Pelacakan SLA (Service Level Agreement):**
   - Otomatis menghitung status overdue berdasarkan tingkat urgensi tiket (Urgent 4 jam, High 8 jam, Medium 24 jam, Low 48 jam).
   - Indikator visual badge **Terlambat / Overdue** muncul otomatis pada detail dan daftar tiket.
4. **Halaman Statistik Admin Terperinci:**
   - Menyajikan KPI operasional: Total Tiket, Tingkat Penyelesaian (Resolution Rate), SLA Breach Rate, Distribusi Kategori & Prioritas, serta Matriks Beban Kerja Agen (Workload).
   - Endpoint khusus `GET /api/admin/stats` dengan proteksi role `admin`.
5. **CI/CD Pipeline (GitHub Actions):**
   - Workflow `.github/workflows/ci.yml` otomatis menjalankan Go lint & race detector test, serta validasi build Vue 3 pada setiap push dan pull request.

---

## 6. Cara Menjalankan Aplikasi

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
   # Edit backend/.env pastikan DATABASE_URL mengarah ke localhost:5433
   # DATABASE_URL=postgres://postgres:postgres@localhost:5433/helpdesk?sslmode=disable
   go run ./cmd/api
   ```
   *Server berjalan di `http://localhost:8080` (auto-migrate & auto-seed 13 tiket sampel).*

3. **Jalankan Frontend (Vite Dev Server):**
   ```bash
   cd frontend
   npm install
   npm run dev
   ```
   *Aplikasi dapat diakses di `http://localhost:5173`.*

---

## 7. Ringkasan API Endpoints

### Autentikasi
- `POST /api/auth/register` – Pendaftaran Requester baru (default role: `requester`)
- `POST /api/auth/login` – Login & penerbitan Bearer Token JWT
- `GET  /api/me` – Profil pengguna saat ini

### Tiket & Komentar
- `GET   /api/tickets` – Daftar tiket (dukungan: `page`, `limit`, `status`, `priority`, `assignee_id`, `search`, `sort_by`, `order`)
- `POST  /api/tickets` – Buat tiket baru (khusus `requester` & `admin`)
- `GET   /api/tickets/{id}` – Detail tiket beserta thread komentar
- `PATCH /api/tickets/{id}` – Update tiket (transisi status, prioritas/assignee khusus admin)
- `POST  /api/tickets/{id}/comments` – Tambah komentar (mendukung `is_internal: true` untuk agent/admin)

### Lampiran & Realtime
- `POST /api/tickets/{id}/attachments` – Unggah berkas lampiran (multipart/form-data, maks 5MB)
- `GET  /api/attachments/{id}/download` – Unduh berkas lampiran (validasi otorisasi tiket)
- `GET  /api/events` – Server-Sent Events (SSE) stream notifikasi real-time

### Statistik (Role-Based)
- `GET /api/stats` – Statistik ringkasan tiket (Open, In Progress, Resolved, Urgent)
- `GET /api/tickets/stats` – Alias statistik ringkasan tiket
- `GET /api/admin/stats` – Statistik komprehensif operasional & beban kerja (khusus role `admin`)

### Pengguna (Admin Only)
- `GET   /api/users` – Daftar seluruh pengguna (bisa filter role, agent hanya dapat melihat daftar assignee)
- `PATCH /api/users/{id}/role` – Ubah role pengguna (dilengkapi proteksi admin terakhir)

---

## 8. Pengujian Otomatis (Automated Tests)

Suite pengujian backend mencakup seluruh layer arsitektur (service, state machine, RBAC, dan HTTP router API handler):

```bash
cd backend
go test -v -race ./...
```

Cakupan 16 test cases:
1. `TestAPI_AuthEndpoints` – Pendaftaran, login, dan validasi sesi `/api/me`
2. `TestAPI_TicketsEndpoints` – Siklus CRUD tiket, otorisasi role, dan filter
3. `TestAPI_CommentsEndpoint` – Komentar publik dan proteksi catatan internal
4. `TestAPI_AttachmentsUploadAndDownload` – Upload multipart, validasi ekstensi/ukuran, dan download berkas
5. `TestAPI_StatsEndpoints` – Otorisasi endpoint statistik umum dan statistik admin
6. `TestAPI_UsersEndpoints` – Manajemen pengguna dan proteksi hak akses role admin
7. `TestAPI_EventsEndpoint` – Koneksi SSE streaming `/api/events`
8. `TestAuthRegisterAndLogin` – Pendaftaran dan autentikasi JWT di layer service
9. `TestTicketIsolation` – Isolasi data tiket antar-requester
10. `TestValidStatusTransitions` – Alur siklus hidup tiket yang sah
11. `TestInvalidStatusTransitions` – Penolakan transisi ilegal di state machine
12. `TestInternalNoteSecurity` – Pencegahan kebocoran catatan internal ke requester
13. `TestAgentCannotCreateTicket` – Verifikasi larangan role agent membuat tiket
14. `TestAgentCannotChangePriorityOrAssignee` – Verifikasi proteksi field prioritas/assignee bagi agent
15. `TestSLAOverdueCalculation` – Kalkulasi SLA overdue untuk tiket urgent/high/medium/low
16. `TestAttachmentUploadAndAccess` – Validasi tipe berkas dan otorisasi akses lampiran

---

## 9. Struktur Direktori (Monorepo)

```
simple-journey-indonesia/
├── backend/
│   ├── cmd/api/                 # Entrypoint server (main.go)
│   ├── internal/
│   │   ├── config/              # Konfigurasi via environment variables
│   │   ├── database/            # Koneksi, skrip migrasi SQL & seeder
│   │   ├── domain/              # Entities, DTO, interfaces, dan custom errors
│   │   ├── handler/             # HTTP controllers (Gin)
│   │   ├── middleware/          # Auth JWT & CORS middleware
│   │   ├── repository/          # PostgreSQL data access layer
│   │   └── service/             # Core business logic & state machine
│   ├── tests/                   # Automated unit & integration tests
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/shared/   # StatusBadge, PriorityBadge, ConfirmDialog, Toast, dll
│   │   ├── constants/           # Definisi status & matriks transisi
│   │   ├── router/              # Vue Router & Navigation Guards
│   │   ├── services/            # Axios API wrappers
│   │   ├── stores/              # Pinia auth store
│   │   └── views/               # Halaman Login, Register, Tickets, CreateTicket, Detail, Users
│   ├── Dockerfile
│   ├── nginx.conf
│   └── package.json
├── docs/                        # Dokumentasi arsitektur & Postman collection
├── docker-compose.yml
└── README.md
```
