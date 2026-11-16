# Product Requirements Document (PRD): Internal Helpdesk System
**Target Audience:** AI Coding Agent / Autonomous Developer  
**Project Context:** Technical Test – Shift Engineer (Fullstack) – PT Simple Journey Indonesia  
**Objective:** Membangun aplikasi web helpdesk internal untuk menggantikan pelacakan tiket dukungan karyawan berbasis email dan spreadsheet.

---

## 1. System Architecture & Tech Stack

AI Agent **WAJIB** menggunakan tech stack dan arsitektur berikut tanpa penyimpangan:

* **Backend:** Go (Golang) versi `>= 1.22`.
  * Framework/Router: `Gin`, `Echo`, `Fiber`, atau `net/http` standar (disarankan `Gin` atau `Echo` untuk produktivitas dan middleware yang bersih).
  * Arsitektur: Clean / Layered Architecture (`handler/controller` -> `service/usecase` -> `repository` -> `model/domain`).
  * Konfigurasi: Menggunakan Environment Variables (`.env`), tanpa *hardcoded credentials*.
* **Frontend:** Vue 3 dengan Composition API (`<script setup>`).
  * Build Tool: Vite.
  * State Management: Pinia.
  * Routing: Vue Router (dengan Navigation Guards).
  * Styling: Tailwind CSS (disarankan untuk kecepatan dan konsistensi desain responsif desktop & mobile).
  * HTTP Client: Axios atau Fetch wrapper terpusat dengan interceptor untuk injeksi Bearer Token dan penanganan otomatis HTTP `401 Unauthorized`.
* **Database:** PostgreSQL (disarankan v15+) atau MySQL (v8+).
  * Migrasi: Wajib menyediakan skrip/mekanisme migrasi database yang *idempotent* (dapat dijalankan ulang dengan aman) beserta *database seeder*.
* **Containerization & Deployment:**
  * `Dockerfile` terpisah untuk Backend dan Frontend.
  * `docker-compose.yml` di root direktori yang orkestrasi service: `db`, `backend`, dan `frontend` agar seluruh sistem berjalan hanya dengan perintah `docker compose up --build`.

---

## 2. Database Schema & Seed Requirements

### 2.1. Skema Tabel
Gunakan UUID atau Auto-Increment BigInt untuk Primary Key (UUID sangat disarankan untuk keamanan IDOR).

1. **`users`**
   * `id`: PK (UUID / BIGINT)
   * `name`: VARCHAR(100), NOT NULL
   * `email`: VARCHAR(150), UNIQUE, NOT NULL
   * `password_hash`: VARCHAR(255), NOT NULL (bcrypt/argon2)
   * `role`: ENUM/VARCHAR (`requester`, `agent`, `admin`), DEFAULT `'requester'`, NOT NULL
   * `created_at`: TIMESTAMP, DEFAULT CURRENT_TIMESTAMP

2. **`tickets`**
   * `id`: PK (UUID / BIGINT)
   * `title`: VARCHAR(255), NOT NULL
   * `description`: TEXT, NOT NULL
   * `category`: VARCHAR(100), NOT NULL (contoh: `IT Support`, `Hardware`, `Software`, `Network`, `HR/Facility`)
   * `priority`: ENUM/VARCHAR (`low`, `medium`, `high`, `urgent`), DEFAULT `'medium'`, NOT NULL
   * `status`: ENUM/VARCHAR (`open`, `in_progress`, `resolved`, `closed`), DEFAULT `'open'`, NOT NULL
   * `requester_id`: FK -> `users.id`, NOT NULL
   * `assignee_id`: FK -> `users.id`, NULLABLE
   * `created_at`: TIMESTAMP, DEFAULT CURRENT_TIMESTAMP
   * `updated_at`: TIMESTAMP, DEFAULT CURRENT_TIMESTAMP

3. **`comments`**
   * `id`: PK (UUID / BIGINT)
   * `ticket_id`: FK -> `tickets.id` (ON DELETE CASCADE), NOT NULL
   * `author_id`: FK -> `users.id`, NOT NULL
   * `body`: TEXT, NOT NULL
   * `is_internal`: BOOLEAN, DEFAULT `FALSE`, NOT NULL
   * `created_at`: TIMESTAMP, DEFAULT CURRENT_TIMESTAMP

### 2.2. Seed Data Wajib
Saat aplikasi dijalankan pertama kali (atau melalui command seed), sistem harus memastikan minimal terdapat **3 akun default** (1 per role) dan beberapa sampel tiket:
* **Admin:** `admin@simplejourney.id` | Password: `Password123!` | Role: `admin`
* **Agent:** `agent@simplejourney.id` | Password: `Password123!` | Role: `agent`
* **Requester:** `requester@simplejourney.id` | Password: `Password123!` | Role: `requester`

---

## 3. Core Business Logic & Security Rules (CRITICAL)

### 3.1. Role-Based Access Control (RBAC)
1. **`requester` (Karyawan):**
   * Dapat membuat tiket baru (`status` awal otomatis `open`, `requester_id` diambil dari JWT).
   * **HANYA** dapat melihat daftar tiket dan detail tiket miliknya sendiri (`ticket.requester_id == current_user.id`).
   * **TIDAK BOLEH** melihat komentar dengan `is_internal == true`.
   * Dapat menambahkan komentar publik (`is_internal = false`) pada tiket miliknya.
   * Dapat menutup atau membuka kembali tiket miliknya sesuai Matriks Transisi Status.
   * **TIDAK BOLEH** mengubah `priority` atau `assignee_id` setelah tiket dibuat (atau saat update).
2. **`agent`:**
   * Dapat melihat seluruh tiket dari semua pengguna.
   * Dapat melakukan *assign* tiket (ke dirinya sendiri atau agent lain), mengubah `priority`, dan mengubah `status` sesuai Matriks Transisi.
   * Dapat melihat seluruh komentar (publik & internal) dan membuat komentar publik maupun catatan internal (`is_internal = true`).
3. **`admin`:**
   * Memiliki seluruh hak akses `agent`.
   * Dapat melihat daftar seluruh pengguna (`GET /api/users`) dan mengubah role pengguna (`PATCH /api/users/{id}/role`).

### 3.2. Matriks Transisi Status Tiket (State Machine)
Backend **WAJIB** memvalidasi setiap perubahan status tiket di layer Service/Usecase. Transisi di luar tabel di bawah ini **HARUS DITOLAK** dengan HTTP status `400 Bad Request` atau `422 Unprocessable Entity` (atau `403 Forbidden` jika role tidak berhak).

| Status Awal (`current`) | Status Tujuan (`target`) | Role yang Diizinkan | Keterangan |
| :--- | :--- | :--- | :--- |
| `open` | `in_progress` | `agent`, `admin` | Agent mulai mengerjakan tiket |
| `in_progress` | `resolved` | `agent`, `admin` | Masalah selesai ditangani |
| `resolved` | `closed` | `requester`, `agent`, `admin` | Tiket ditutup permanen |
| `resolved` | `open` | `requester` | Reopen tiket jika solusi belum berhasil |
| `open` | `closed` | `requester`, `admin` | Pembatalan tiket sebelum dikerjakan |

*Catatan Validasi:*
* Jika `requester` mencoba mengubah status tiket milik `requester` lain -> Return `403 Forbidden` (atau `404 Not Found`).
* Jika `requester` mencoba mengirim payload `is_internal: true` pada komentar -> Backend harus menolak (`403 Forbidden`) atau memaksa nilai menjadi `false`.

---

## 4. API Specification

Semua endpoint (kecuali `/auth/*`) memerlukan header `Authorization: Bearer <token>`. Format respons JSON harus konsisten (misal: `{"data": ..., "meta": ...}` untuk sukses, dan `{"error": "pesan error jelas"}` untuk gagal).

### 4.1. Auth & Profile
* `POST /api/auth/register` (Public)
  * Body: `{ "name": "...", "email": "...", "password": "..." }`
  * Validasi: Email valid & unik, password min 8 karakter. Role otomatis di-set `requester`.
* `POST /api/auth/login` (Public)
  * Body: `{ "email": "...", "password": "..." }`
  * Response: `{ "token": "<jwt>", "user": { "id", "name", "email", "role" } }`
* `GET /api/me` (Authenticated)
  * Response: Profil user yang sedang login berdasarkan claims JWT.

### 4.2. Tickets
* `GET /api/tickets` (Authenticated)
  * Query Params:
    * `page` (default: 1), `limit` (default: 10)
    * `status` (`open`, `in_progress`, `resolved`, `closed`)
    * `priority` (`low`, `medium`, `high`, `urgent`)
    * `assignee_id` (UUID/ID atau keyword `unassigned` / `me`)
    * `search` (pencarian case-insensitive berdasarkan `title`)
    * `sort_by` (`created_at`, `updated_at`, `priority`), `order` (`asc`, `desc`)
  * Behavior: Jika role `requester`, filter otomatis `WHERE requester_id = current_user.id`.
  * Response: List tiket + metadata pagination (`total_items`, `total_pages`, `current_page`, `limit`).
* `POST /api/tickets` (Authenticated)
  * Body: `{ "title": "...", "description": "...", "category": "...", "priority": "..." }`
  * Validasi: Field wajib tidak boleh kosong, `priority` harus valid enum.
* `GET /api/tickets/{id}` (Owner, Agent, Admin)
  * Behavior: Mengembalikan detail tiket (termasuk data requester & assignee) beserta array `comments`. Jika user adalah `requester`, filter out komentar dengan `is_internal == true`.
* `PATCH /api/tickets/{id}` (Role-based)
  * Body (partial update): `{ "status": "...", "priority": "...", "assignee_id": "..." }`
  * Behavior:
    * Jika `status` dikirim, validasi menggunakan Matriks Transisi Status.
    * Jika `priority` atau `assignee_id` dikirim oleh `requester`, tolak dengan `403 Forbidden`.

### 4.3. Comments
* `POST /api/tickets/{id}/comments` (Owner, Agent, Admin)
  * Body: `{ "body": "...", "is_internal": boolean }`
  * Behavior: Pastikan user berhak mengakses tiket `{id}`. Tolak `is_internal: true` jika role adalah `requester`.

### 4.4. User Management (Admin Only)
* `GET /api/users` (Admin)
  * Query Params: `role` (opsional, berguna bagi frontend saat memuat daftar `agent`/`admin` untuk dropdown assignee).
* `PATCH /api/users/{id}/role` (Admin)
  * Body: `{ "role": "requester" | "agent" | "admin" }`
  * Validasi: Mencegah admin menurunkan role dirinya sendiri jika ia satu-satunya admin (opsional namun praktik yang baik).

---

## 5. Frontend Requirements (Vue 3)

### 5.1. Arsitektur & State Management
* **Pinia Stores:**
  * `useAuthStore`: Menyimpan `token`, `user`, `isAuthenticated`, `isAdmin`, `isAgent`, `isRequester`, serta action `login()`, `register()`, `fetchMe()`, `logout()`. Sinkronisasi dengan `localStorage`.
* **Axios Instance (`src/services/api.js`):**
  * Request interceptor: Menambahkan `Authorization: Bearer <token>`.
  * Response interceptor: Jika status `401`, otomatis panggil `authStore.logout()` dan redirect ke `/login`.
* **Vue Router (`src/router/index.js`):**
  * Meta routes: `requiresAuth: true`, `guestOnly: true`, `roles: ['admin']`.
  * Navigation guard (`beforeEach`) memverifikasi status login dan otorisasi role sebelum merender komponen.

### 5.2. Halaman & Komponen Utama
1. **Login (`/login`) & Register (`/register`):**
   * Validasi form sisi klien (required fields, format email, panjang password).
   * Tampilkan pesan error dari API dengan jelas (misal: "Email atau password salah").
2. **Dashboard / Daftar Tiket (`/tickets`):**
   * Filter bar: Input pencarian judul (dengan *debounce*), dropdown filter Status, Prioritas, dan Assignee (khusus Agent/Admin).
   * Tabel/Card list responsif menampilkan ID singkat, Judul, Kategori, Badge Prioritas (warna berbeda: Low=Abu, Medium=Biru, High=Oranye, Urgent=Merah), Badge Status, Requester, Assignee, dan Waktu dibuat.
   * Kontrol Pagination (Previous, Next, info halaman).
   * Tombol CTA `+ Buat Tiket Baru`.
3. **Buat Tiket (`/tickets/create` atau Modal):**
   * Form input: Title, Category (dropdown), Priority (dropdown), Description (textarea).
4. **Detail Tiket (`/tickets/:id`):**
   * Panel Informasi Tiket: Judul, deskripsi lengkap, status saat ini, prioritas, pembuat tiket, dan assignee.
   * **Panel Aksi Dinamis (Role-Aware):**
     * Render tombol transisi status **HANYA** yang valid berdasarkan status saat ini dan role user (sesuai Matriks Transisi). Contoh: Jika status `open` dan user `agent`, tampilkan tombol `"Proses Tiket (In Progress)"`. Jika status `resolved` dan user `requester`, tampilkan tombol `"Tutup Tiket"` dan `"Buka Kembali (Reopen)"`.
     * Dropdown ubah Prioritas dan Assignee (hanya muncul untuk `agent` & `admin`).
   * **Thread Komentar:**
     * Membedakan tampilan visual antara Komentar Publik dan Catatan Internal (misal: background kuning lembut dengan label badge `"Internal Note"` untuk catatan internal).
     * Form tambah komentar di bagian bawah, dilengkapi checkbox `"Jadikan catatan internal (hanya dilihat Agent & Admin)"` khusus untuk `agent` dan `admin`.
5. **Manajemen Pengguna (`/users` - Khusus Admin):**
   * Menu navigasi hanya muncul jika `user.role === 'admin'`.
   * Tabel daftar pengguna (Nama, Email, Role saat ini, Tanggal terdaftar) dengan dropdown langsung untuk mengubah role pengguna beserta notifikasi sukses/gagal.

### 5.3. UX & UI States
Setiap halaman yang memuat data API **WAJIB** menangani 3 state visual:
* **Loading State:** Spinner atau skeleton loader saat request berlangsung.
* **Empty State:** Ilustrasi/pesan ramah saat daftar tiket kosong atau pencarian tidak ditemukan.
* **Error State:** Alert box dengan pesan error yang informatif dan tombol "Coba Lagi".

---

## 6. Testing & Quality Assurance

AI Agent **WAJIB** membuat suite pengujian otomatis di Backend Go menggunakan package `testing` standar (dan `testify` jika diperlukan). Minimal terdapat **6 test cases bermakna** yang mencakup:
1. **Auth Test:** Register berhasil & Login mengembalikan JWT valid.
2. **RBAC / Isolation Test:** Requester A mencoba mengakses `GET /api/tickets/{id}` milik Requester B -> Harus ditolak (`403` atau `404`).
3. **Status Transition Valid Test:** Agent mengubah status tiket dari `open` -> `in_progress` -> `resolved` -> Berhasil (`200`).
4. **Status Transition Invalid Test:**
   * Requester mencoba mengubah status dari `open` -> `resolved` (melompati alur) -> Ditolak (`400`/`422`/`403`).
   * Agent mencoba melakukan `reopen` dari `resolved` -> `open` (hanya boleh Requester) -> Ditolak (`403`).
5. **Internal Note Security Test:**
   * Requester mencoba membuat komentar dengan `is_internal: true` -> Ditolak (`403`).
   * Requester mengambil detail tiket yang memiliki catatan internal -> Catatan internal tidak muncul di respons JSON.

---

## 7. Bonus Features (Implementasikan Jika Waktu & Konteks Memungkinkan)
Untuk memaksimalkan nilai evaluasi teknis, prioritaskan implementasi fitur bonus berikut secara bertahap setelah fitur inti selesai 100%:
1. **Simple SLA Indicator:** Tandai tiket berstatus `open` dengan prioritas `urgent` yang berusia lebih dari 2 jam (atau parameter konfigurasi `.env`) dengan flag `is_overdue: true` di API dan tampilkan badge merah `"SLA Overdue"` di UI.
2. **Admin Statistics Summary:** Tambahkan endpoint `GET /api/stats` (khusus Agent/Admin) dan widget ringkasan di atas halaman Daftar Tiket (Total Open, In Progress, Resolved, Urgent Tickets).
3. **OpenAPI / Swagger atau Postman Collection:** Sertakan file `api-docs.yaml` atau `postman_collection.json` di root repository.

---

## 8. Deliverables & Repository Structure

Struktur direktori akhir yang harus dihasilkan oleh AI Agent:

```text
/
├── backend/
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── config/
│   │   ├── database/         # Koneksi DB, migrasi SQL, dan seeder
│   │   ├── domain/           # Struct entity & interface
│   │   ├── handler/          # HTTP handlers
│   │   ├── middleware/       # JWT Auth & RBAC middlewares
│   │   ├── repository/       # Query database
│   │   └── service/          # Business logic & state machine validation
│   ├── tests/                # Unit & Integration tests
│   ├── .env.example
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── router/
│   │   ├── services/
│   │   ├── stores/
│   │   └── views/
│   ├── .env.example
│   ├── Dockerfile
│   ├── nginx.conf            # Untuk serve build Vite di production container
│   └── package.json
├── docs/
│   ├── postman_collection.json
│   └── ARCHITECTURE.md       # Catatan keputusan arsitektur & trade-offs (maks 1 halaman)
├── docker-compose.yml
└── README.md                 # Panduan lengkap setup, akun seed, endpoint, dan cara run test
```

---

## 9. Step-by-Step Execution Plan for AI Agent

1. **Inisialisasi Infrastruktur:** Buat struktur folder, `docker-compose.yml` (PostgreSQL + Backend + Frontend), dan `.env.example`.
2. **Backend Core & Database:** Buat skema migrasi SQL, fungsi seeder (3 akun wajib), koneksi database, dan konfigurasi environment.
3. **Backend Business Logic:** Implementasikan Auth (JWT + bcrypt), Middleware RBAC, State Machine Transisi Status, CRUD Tiket (dengan filter/pagination/search), Komentar (publik vs internal), dan Manajemen User.
4. **Backend Automated Testing:** Tulis dan jalankan `go test ./...` untuk memastikan seluruh skenario transisi status dan isolasi data lulus 100%.
5. **Frontend Implementation:** Setup Vue 3 + Vite + Pinia + Vue Router + Tailwind CSS. Bangun Axios interceptor, Auth store, dan seluruh halaman utama dengan penanganan loading/empty/error state.
6. **End-to-End Verification & Documentation:** Jalankan `docker compose up --build`, uji seluruh alur dari Login hingga Close/Reopen tiket, lalu lengkapi `README.md`, `docs/ARCHITECTURE.md`, dan `docs/postman_collection.json`.