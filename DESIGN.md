# System & UI Design Document (`DESIGN.md`)

**Project:** Internal Helpdesk System – PT Simple Journey Indonesia

**Companion Document:** `PRD.md`

**Target Audience:** AI Coding Agent

**Design Philosophy:** Strict Clean Architecture (Backend & Frontend) + Modern Enterprise Minimalism (`shadcn-vue` + Tailwind CSS).

## 1. Backend Clean Architecture (Go `>= 1.22`)

Backend harus menerapkan **Dependency Inversion Principle** dan pemisahan tanggung jawab (*Separation of Concerns*) yang ketat. Ketergantungan kode hanya boleh mengarah ke dalam (`Handler` -> `Service` -> `Repository` -> `Domain`).

### 1.1. Dependency Rules & Layer Responsibilities

1. **Domain Layer (`internal/domain/`)**

   * **Aturan:** Tidak boleh mengimpor package dari layer lain (`handler`, `service`, `repository`) maupun library framework HTTP (`gin`/`echo`). Hanya berisi pure Go structs, custom error sentinel, enum constants, dan interface kontrak.

   * **Isi:**

     * `entity.go`: Struct `User`, `Ticket`, `Comment`.

     * `dto.go`: Request/Response structs (`CreateTicketReq`, `UpdateTicketReq`, `TicketFilter`, `PaginationMeta`).

     * `errors.go`: Domain errors (`ErrNotFound`, `ErrForbidden`, `ErrInvalidTransition`, `ErrUnauthorized`, `ErrConflict`).

     * `interfaces.go`: Kontrak interface untuk `TicketRepository`, `UserRepository`, `CommentRepository`, serta `TicketService`, `AuthService`, `UserService`.

2. **Repository Layer (`internal/repository/`)**

   * **Aturan:** Bertanggung jawab murni atas operasi I/O database (SQL queries). Tidak boleh ada business logic (seperti validasi transisi status atau pengecekan role) di layer ini.

   * **Implementasi:** Mengimplementasikan interface repository dari `domain`. Menggunakan parameterized queries untuk mencegah SQL Injection dan mendukung dynamic query building untuk pagination, search, dan filter.

3. **Service / Usecase Layer (`internal/service/`)**

   * **Aturan:** Pusat dari seluruh *business logic*. Layer ini bergantung pada interface `Repository` (bukan struct konkret) agar mudah di-mock saat unit testing.

   * **Implementasi:**

     * `TicketStateMachine`: Fungsi murni `ValidateTransition(currentStatus, targetStatus, actorRole)` yang memvalidasi aturan matriks transisi sebelum memanggil `repository.Update()`.

     * `RBAC Guard`: Memastikan `requester` hanya bisa membaca/mengubah tiketnya sendiri dan memfilter `is_internal` comments.

4. **Delivery / Handler Layer (`internal/handler/`)**

   * **Aturan:** Bertanggung jawab parsing HTTP request (JSON body, query params, URL params), validasi input sintaksis dasar, memanggil `Service`, dan memetakan `domain.Errors` menjadi HTTP Status Code yang tepat (`ErrInvalidTransition` -> `422`, `ErrForbidden` -> `403`, `ErrNotFound` -> `404`).

### 1.2. Struktur Direktori Backend

```
backend/
├── cmd/
│   └── api/
│       └── main.go                 # Composition Root (Dependency Injection wiring)
├── internal/
│   ├── config/
│   │   └── config.go               # Load & validate .env variables
│   ├── database/
│   │   ├── postgres.go             # DB connection pool setup
│   │   ├── migrations/             # Up/Down SQL files
│   │   └── seeder.go               # Idempotent default users & sample tickets
│   ├── domain/
│   │   ├── entity.go
│   │   ├── dto.go
│   │   ├── errors.go
│   │   └── interfaces.go
│   ├── repository/
│   │   ├── user_repo.go
│   │   ├── ticket_repo.go
│   │   └── comment_repo.go
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── ticket_service.go
│   │   ├── state_machine.go        # Isolated ticket status transition logic
│   │   └── user_service.go
│   ├── handler/
│   │   ├── response.go             # Standardized JSON envelope & error mapper
│   │   ├── auth_handler.go
│   │   ├── ticket_handler.go
│   │   └── user_handler.go
│   └── middleware/
│       ├── auth_middleware.go      # JWT verification & context injection
│       └── rbac_middleware.go      # Role guard helper (e.g., RequireRoles("admin"))
└── tests/
    ├── state_machine_test.go
    └── ticket_integration_test.go

```

## 2. Frontend Clean Architecture (Vue 3 + TypeScript/JS)

Frontend mengadopsi **Feature-Driven Clean Architecture** agar komponen UI (`shadcn-vue`) tetap "bodoh" (hanya bertugas menampilkan data dan memancarkan event) sementara logika bisnis, transformasi data, dan komunikasi API terisolasi rapi.

### 2.1. Layering Frontend

1. **Core / Infrastructure (`src/core/`)**

   * `api/client.js`: Instance Axios tunggal dengan request interceptor (inject JWT) dan response interceptor (handle `401` auto-logout & format error message).

   * `constants/ticket.js`: Definisi enum status, prioritas, kategori, dan **Client-side Transition Map** (untuk menentukan tombol aksi mana yang dirender di UI).

2. **Services / Repositories (`src/services/`)**

   * Abstraksi pemanggilan endpoint API (`auth.service.js`, `ticket.service.js`, `user.service.js`). Komponen Vue **DILARANG** mengimpor `axios` secara langsung; wajib melalui service layer atau store.

3. **State Management (`src/stores/`)**

   * Pinia stores (`useAuthStore`, `useTicketStore`) mengelola state global, caching sementara, dan state `loading` / `error`.

4. **Composables (`src/composables/`)**

   * Logika UI yang reusable (contoh: `useDebounce.js` untuk input pencarian, `useTicketPermissions.js` untuk mengecek hak aksi user pada tiket tertentu).

5. **UI & Feature Components (`src/components/` & `src/views/`)**

   * `src/components/ui/`: Komponen primitif dari **shadcn-vue** (jangan dimodifikasi logika bisnisnya).

   * `src/components/shared/`: Komponen reusable aplikasi (`AppSidebar`, `EmptyState`, `ErrorAlert`, `LoadingSkeleton`, `StatusBadge`, `PriorityBadge`).

   * `src/views/`: Halaman tingkat rute yang merakit komponen shared dan feature.

## 3. UI/UX Design System dengan `shadcn-vue`

Aplikasi menggunakan estetika **Clean Enterprise Dashboard** (terinspirasi dari Linear & Zendesk modern) dengan palet warna netral (`Slate` / `Zinc`) agar kontras status dan prioritas tiket langsung terbaca.

### 3.1. Setup & Konfigurasi `shadcn-vue`

AI Agent wajib menginisialisasi `shadcn-vue` (menggunakan `radix-vue`, `lucide-vue-next`, `class-variance-authority`, `clsx`, `tailwind-merge`) dengan tema dasar **Zinc** atau **Slate**, border radius `0.5rem`, dan menginstal komponen berikut:

* **Layout & Navigasi:** `Button`, `Separator`, `Sheet` (untuk mobile drawer sidebar), `DropdownMenu`, `Avatar`, `Breadcrumb`.

* **Data Display:** `Table`, `Badge`, `Card`, `Skeleton` (untuk loading state).

* **Forms & Input:** `Input`, `Textarea`, `Label`, `Select`, `Checkbox`, `Form`.

* **Feedback & Overlay:** `Dialog` (untuk modal konfirmasi/buat tiket), `Alert`, `AlertDialog` (untuk konfirmasi pembatalan/penutupan tiket), `Sonner` atau `Toast` (untuk notifikasi aksi sukses/gagal).

### 3.2. Sistem Warna & Semantik Badge

Gunakan komponen `<Badge>` dari `shadcn-vue` dengan styling kustom yang konsisten di seluruh halaman:

#### A. Status Tiket (`StatusBadge.vue`)

| Status | Label UI | Styling Tailwind (`Badge` variant/class) | Icon (`lucide-vue-next`) | 
| ----- | ----- | ----- | ----- | 
| `open` | Open | `bg-sky-500/15 text-sky-700 border-sky-200 hover:bg-sky-500/20` | `CircleDot` | 
| `in_progress` | In Progress | `bg-amber-500/15 text-amber-700 border-amber-200 hover:bg-amber-500/20` | `Clock` | 
| `resolved` | Resolved | `bg-emerald-500/15 text-emerald-700 border-emerald-200 hover:bg-emerald-500/20` | `CheckCircle2` | 
| `closed` | Closed | `bg-slate-500/15 text-slate-700 border-slate-200 hover:bg-slate-500/20` | `Archive` | 

#### B. Prioritas Tiket (`PriorityBadge.vue`)

| Priority | Label UI | Styling Tailwind (`Badge` variant/class) | Icon (`lucide-vue-next`) | 
| ----- | ----- | ----- | ----- | 
| `low` | Low | `bg-slate-100 text-slate-600 border-slate-200` | `ArrowDown` | 
| `medium` | Medium | `bg-blue-50 text-blue-700 border-blue-200` | `Minus` | 
| `high` | High | `bg-orange-50 text-orange-700 border-orange-200` | `ArrowUp` | 
| `urgent` | Urgent | `bg-rose-50 text-rose-700 border-rose-200 font-semibold` | `AlertTriangle` | 

### 3.3. Wireframe & Layout Blueprint per Halaman

#### 1. App Shell (`MainLayout.vue`)

* **Desktop (`>= 1024px`):** Fixed Left Sidebar (`w-64`, `border-r bg-muted/40`) berisi Logo PT Simple Journey Indonesia, menu navigasi (`Tiket Saya` / `Semua Tiket`, `Manajemen Pengguna` khusus Admin), dan User Profile Card di bagian bawah dengan tombol Logout.

* **Mobile (`< 1024px`):** Top Header bar dengan tombol Hamburger yang memicu komponen `shadcn` `<Sheet side="left">` berisi menu sidebar yang sama.

#### 2. Halaman Auth (`/login` & `/register`)

* Centered single-column layout (`min-h-screen flex items-center justify-center bg-muted/30 p-4`).

* Menggunakan komponen `<Card class="w-full max-w-md">` dengan `<CardHeader>`, `<CardContent>`, dan `<CardFooter>`.

* Di halaman Login, sediakan helper collapsible atau quick-fill button kecil untuk akun demo/seed (`Admin`, `Agent`, `Requester`) guna mempermudah evaluator menguji aplikasi.

#### 3. Halaman Daftar Tiket (`/tickets`)

* **Header Area:** Judul halaman di kiri + tombol `<Button>` `+ Buat Tiket` di kanan.

* **Filter Toolbar:** Menggunakan flex/grid responsif berisi `<Input>` search (dengan icon `Search`), `<Select>` filter Status, `<Select>` filter Prioritas, dan `<Select>` filter Assignee (khusus Agent/Admin).

* **Content Area:**

  * *Loading:* `<Table>` dengan 5 baris `<Skeleton class="h-12 w-full" />`.

  * *Empty:* `<Card>` dashed border dengan icon `Inbox`, pesan deskriptif, dan tombol reset filter / buat tiket.

  * *Data:* `<Table>` dari `shadcn-vue` (pada mobile menggunakan `overflow-x-auto` atau berubah menjadi stack `<Card>` list). Kolom: `ID`, `Subjek & Kategori`, `Status`, `Prioritas`, `Assignee`, `Dibuat`.

* **Pagination Footer:** Informasi `"Menampilkan X-Y dari Z tiket"` di kiri, dan grup `<Button variant="outline" size="sm">` `Previous` / `Next` di kanan.

#### 4. Halaman Detail Tiket (`/tickets/:id`)

Menggunakan layout **2-Kolom Asimetris** pada desktop (`grid-cols-1 lg:grid-cols-3 gap-6`):

* **Kolom Kiri (`lg:col-span-2`) – Konten Utama & Diskusi:**

  * `<Card>` Detail Masalah: Judul besar (`text-xl font-semibold`), metadata pembuat & timestamp, `<Separator />`, dan isi `description` (`whitespace-pre-wrap`).

  * **Section Komentar:**

    * Komentar Publik: `<Card>` standar (`bg-card`).

    * Catatan Internal (`is_internal: true`): `<Card class="bg-amber-50/70 border-amber-200 dark:bg-amber-950/20">` dengan `<Badge variant="outline" class="border-amber-400 text-amber-800">` bertuliskan `Internal Note (Agent & Admin Only)` dan icon `Lock`.

  * **Form Balas Komentar:** `<Textarea>` + `<Checkbox id="internal">` (hanya render jika `isAgent || isAdmin`) + `<Button>` Kirim.

* **Kolom Kanan (`lg:col-span-1`) – Panel Kontrol & Properti Tiket:**

  * `<Card>` Status & Aksi:

    * Menampilkan status saat ini.

    * **Action Buttons:** Render tombol aksi berdasarkan Matriks Transisi. Contoh:

      * Jika transisi ke `in_progress` atau `resolved`: `<Button class="w-full">` (Primary).

      * Jika transisi ke `closed` (batal/tutup): `<Button variant="destructive" class="w-full">` yang memicu `<AlertDialog>` konfirmasi.

      * Jika transisi `reopen`: `<Button variant="outline" class="w-full">`.

  * `<Card>` Properti Tiket:

    * **Assignee:** Jika `Agent/Admin`, render `<Select>` berisi daftar Agent/Admin (plus opsi "Unassigned"). Jika `Requester`, tampilkan teks nama assignee saja.

    * **Prioritas:** Jika `Agent/Admin`, render `<Select>` (`low`, `medium`, `high`, `urgent`) yang langsung memanggil `PATCH /api/tickets/:id` saat diubah. Jika `Requester`, tampilkan `<PriorityBadge />` read-only.

#### 5. Halaman Manajemen Pengguna (`/users` - Admin Only)

* `<Card>` berisi `<Table>` daftar seluruh user (`Nama`, `Email`, `Role`, `Tanggal Bergabung`).

* Kolom `Role` merender komponen `<Select>` (`requester`, `agent`, `admin`). Perubahan langsung memicu `PATCH /api/users/:id/role` dan menampilkan notifikasi `<Sonner />` / `<Toast />`.

## 4. Coding Standards & Clean Code Checklist for AI Agent

1. **No God Functions / Components:** Setiap file komponen Vue tidak boleh melebihi ~200 baris. Pecah bagian kompleks (seperti `TicketCommentThread.vue` dan `TicketSidebarControls.vue`) menjadi sub-komponen terpisah.

2. **Explicit Error Handling:** Backend tidak boleh mengabaikan `err != nil`. Semua error SQL atau validasi harus dibungkus dengan konteks yang jelas.

3. **Zero Magic Strings:** Gunakan konstanta untuk semua nilai role (`RoleAdmin`, `RoleAgent`, `RoleRequester`), status tiket (`StatusOpen`, dll.), dan prioritas.

4. **Strict Separation of DTO and Entity:** Jangan pernah mengembalikan struct `domain.User` (yang memiliki field `PasswordHash`) secara langsung ke JSON response meskipun sudah diberi tag `json:"-"`; tetap gunakan Response DTO atau pastikan serialisasi aman.