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

## 4. Cara Menjalankan Aplikasi

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

## 5. Ringkasan API Endpoints

### Autentikasi
- `POST /api/auth/register` – Pendaftaran Requester baru (default role: `requester`)
- `POST /api/auth/login` – Login & penerbitan Bearer Token JWT
- `GET  /api/me` – Profil pengguna saat ini

### Tiket
- `GET   /api/tickets` – Daftar tiket (dukungan: `page`, `limit`, `status`, `priority`, `assignee_id`, `search`, `sort_by`, `order`)
- `POST  /api/tickets` – Buat tiket baru (khusus `requester` & `admin`)
- `GET   /api/tickets/{id}` – Detail tiket beserta thread komentar
- `PATCH /api/tickets/{id}` – Update tiket (transisi status, prioritas/assignee khusus admin)
- `POST  /api/tickets/{id}/comments` – Tambah komentar (mendukung `is_internal: true` untuk agent/admin)
- `GET   /api/stats` – Statistik ringkasan tiket (Open, In Progress, Resolved, Urgent)

### Pengguna (Admin Only)
- `GET   /api/users` – Daftar seluruh pengguna (bisa filter role, agent hanya dapat melihat daftar assignee)
- `PATCH /api/users/{id}/role` – Ubah role pengguna

---

## 6. Pengujian Otomatis (Automated Tests)

Suite pengujian unit backend mencakup skenario kritis:

```bash
cd backend
go test -v ./tests/...
```

Cakupan pengujian:
1. `TestAuthRegisterAndLogin` – Pendaftaran dan autentikasi JWT
2. `TestTicketIsolation` – Isolasi data tiket antar-requester
3. `TestValidStatusTransitions` – Alur siklus hidup tiket yang sah
4. `TestInvalidStatusTransitions` – Penolakan transisi ilegal di state machine
5. `TestInternalNoteSecurity` – Pencegahan kebocoran catatan internal ke requester
6. `TestAgentCannotCreateTicket` – Verifikasi larangan role agent membuat tiket
7. `TestAgentCannotChangePriorityOrAssignee` – Verifikasi proteksi field prioritas/assignee bagi agent

---

## 7. Struktur Direktori (Monorepo)

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
