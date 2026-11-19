# Dokumentasi Lengkap Cara Kerja Internal Helpdesk System

Dokumen ini menjelaskan cara kerja aplikasi helpdesk pada repositori ini, mulai dari browser, autentikasi, pemrosesan tiket, penyimpanan database, hingga notifikasi real-time dan deployment. Penjelasan ditujukan untuk pengembang, penguji, administrator aplikasi, dan pembaca yang ingin memahami hubungan antara fitur dengan kode implementasinya.

> **Dasar penjelasan:** kode sumber yang tersedia saat dokumen disusun. Perilaku aktual dibedakan dari rencana fitur atau klaim pada dokumentasi lama. Potongan kode merupakan cuplikan dari berkas terkait, bukan berkas lengkap, kecuali dinyatakan lain. Semua path kode pada uraian dihitung dari root repositori.
>
> **Penting:** ini adalah dokumentasi implementasi, bukan pernyataan bahwa aplikasi sudah siap produksi atau seluruh skenario sudah diuji. Batasan yang memengaruhi penggunaan dijelaskan pada bagian akhir.

## Daftar Isi

1. [Gambaran umum sistem](#1-gambaran-umum-sistem)
2. [Teknologi dan dependensi](#2-teknologi-dan-dependensi)
3. [Arsitektur dan aliran data](#3-arsitektur-dan-aliran-data)
4. [Peta direktori dan tanggung jawab kode](#4-peta-direktori-dan-tanggung-jawab-kode)
5. [Proses startup backend](#5-proses-startup-backend)
6. [Konfigurasi aplikasi](#6-konfigurasi-aplikasi)
7. [Model domain dan struktur data](#7-model-domain-dan-struktur-data)
8. [Database, migrasi, dan data awal](#8-database-migrasi-dan-data-awal)
9. [Autentikasi dan pengelolaan sesi](#9-autentikasi-dan-pengelolaan-sesi)
10. [Hak akses berdasarkan role](#10-hak-akses-berdasarkan-role)
11. [Siklus hidup dan transisi status tiket](#11-siklus-hidup-dan-transisi-status-tiket)
12. [Cara kerja layanan tiket](#12-cara-kerja-layanan-tiket)
13. [Query SQL, pencarian, filter, dan pagination](#13-query-sql-pencarian-filter-dan-pagination)
14. [Komentar dan catatan internal](#14-komentar-dan-catatan-internal)
15. [Unggah dan unduh lampiran](#15-unggah-dan-unduh-lampiran)
16. [SLA dan statistik operasional](#16-sla-dan-statistik-operasional)
17. [Notifikasi real-time dengan SSE](#17-notifikasi-real-time-dengan-sse)
18. [Cara kerja frontend Vue](#18-cara-kerja-frontend-vue)
19. [Penjelasan setiap halaman frontend](#19-penjelasan-setiap-halaman-frontend)
20. [Komponen bersama, bahasa, dan tema](#20-komponen-bersama-bahasa-dan-tema)
21. [Kontrak API dan penanganan error](#21-kontrak-api-dan-penanganan-error)
22. [Contoh alur penggunaan menyeluruh](#22-contoh-alur-penggunaan-menyeluruh)
23. [Menjalankan aplikasi dan deployment](#23-menjalankan-aplikasi-dan-deployment)
24. [Pengujian dan pipeline CI](#24-pengujian-dan-pipeline-ci)
25. [Batasan implementasi dan catatan keamanan](#25-batasan-implementasi-dan-catatan-keamanan)
26. [Panduan membaca dan mengembangkan kode](#26-panduan-membaca-dan-mengembangkan-kode)
27. [Troubleshooting dan glosarium](#27-troubleshooting-dan-glosarium)

**Pendalaman kode API dan relasinya dengan frontend:**

28. [Peta lengkap relasi endpoint dan pemakainya](#28-peta-lengkap-relasi-endpoint-dan-pemakainya)
29. [Mekanisme request dan response lintas lapisan](#29-mekanisme-request-dan-response-lintas-lapisan)
30. [Bedah API autentikasi dan integrasi frontend](#30-bedah-api-autentikasi-dan-integrasi-frontend)
31. [Bedah API tiket dan integrasi frontend](#31-bedah-api-tiket-dan-integrasi-frontend)
32. [Bedah API komentar dan relasi diskusi](#32-bedah-api-komentar-dan-relasi-diskusi)
33. [Bedah API lampiran dan relasi berkas](#33-bedah-api-lampiran-dan-relasi-berkas)
34. [Bedah API statistik dan pemetaan dashboard](#34-bedah-api-statistik-dan-pemetaan-dashboard)
35. [Bedah API pengguna dan relasi penugasan](#35-bedah-api-pengguna-dan-relasi-penugasan)
36. [Bedah API SSE dan sinkronisasi frontend](#36-bedah-api-sse-dan-sinkronisasi-frontend)
37. [Peta field database ke JSON dan komponen Vue](#37-peta-field-database-ke-json-dan-komponen-vue)
38. [Ketergantungan request dan propagasi perubahan](#38-ketergantungan-request-dan-propagasi-perubahan)
39. [Penelusuran masalah dan verifikasi kontrak API](#39-penelusuran-masalah-dan-verifikasi-kontrak-api)

---

## 1. Gambaran Umum Sistem

**Internal Helpdesk System** adalah aplikasi web untuk mencatat, mendiskusikan, dan menangani permintaan bantuan internal, misalnya gangguan jaringan, masalah perangkat keras, kebutuhan perangkat lunak, dan fasilitas kantor.

Aplikasi memiliki tiga peran utama:

| Peran | Tanggung jawab |
|---|---|
| `requester` | Melaporkan masalah, melihat tiket sendiri, menambahkan informasi, mengonfirmasi penutupan, atau membuka kembali tiket yang sudah diselesaikan. |
| `agent` | Melihat tiket, menentukan prioritas, memproses dan menyelesaikan masalah, serta menambahkan catatan internal. |
| `admin` | Mengawasi operasional, membuat tiket termasuk atas nama pengguna lain, mengatur penugasan/pelapor, melihat statistik, dan mengubah role pengguna. |

Secara sederhana, alur utamanya adalah:

1. Pengguna mendaftar atau login.
2. Requester/admin membuat tiket berstatus `open`.
3. Agent/admin menentukan prioritas dan memulai pengerjaan.
4. Tiket berpindah menjadi `in_progress`, kemudian `resolved`.
5. Tiket ditutup menjadi `closed`, atau requester membuka kembali tiket dari `resolved` ke `open`.
6. Komentar dan lampiran melengkapi informasi penanganan.
7. Event SSE memberi tahu browser tentang perubahan agar data yang relevan dapat dimuat ulang.

Fitur yang benar-benar memiliki implementasi pada kode saat ini meliputi autentikasi JWT, pembatasan akses tiket, filter/pagination, state machine, komentar internal, edit/hapus komentar sendiri, lampiran tiket dan lampiran komentar, pratinjau berkas lokal di aplikasi, penguncian diskusi saat tiket `closed`, statistik, SSE, tampilan responsif, bahasa Indonesia/Inggris, dan tema terang/gelap.

Tidak tersedia endpoint penghapusan tiket, reset password, pengiriman email, refresh token, maupun riwayat perubahan status terpisah.

## 2. Teknologi dan Dependensi

Sumber versi backend: [`backend/go.mod`](../backend/go.mod). Sumber deklarasi dependensi frontend: [`frontend/package.json`](../frontend/package.json). Versi frontend di bawah merupakan rentang yang dideklarasikan, bukan jaminan versi persis yang terpasang; resolusi dikunci oleh `package-lock.json`.

| Area | Teknologi | Penggunaan |
|---|---|---|
| Backend | Go, directive `go 1.22` | Server HTTP, aturan bisnis, akses database, dan SSE. |
| HTTP framework | Gin `v1.10.0` | Router, request binding, response JSON, middleware. |
| Database access | `database/sql` + `github.com/lib/pq` `v1.10.9` | SQL PostgreSQL tanpa ORM. |
| Token | `github.com/golang-jwt/jwt/v5` `v5.2.1` | Membuat dan memverifikasi JWT. |
| Password | bcrypt dari `golang.org/x/crypto` | Hash password dan pencocokan saat login. |
| Identifier | `github.com/google/uuid` `v1.6.0` | ID user, tiket, komentar, dan lampiran. |
| Environment lokal | `github.com/joho/godotenv` `v1.5.1` | Membaca `.env` secara opsional pada startup. |
| Validasi HTTP | Binding Gin dengan validator | Memeriksa tag seperti `required`, `email`, dan `min=8`. |
| Frontend | Vue `^3.5.0` | SPA dengan Composition API dan `<script setup>`. |
| Routing | Vue Router `^4.4.6` | Navigasi halaman dan pemeriksaan akses UI. |
| Global state | Pinia `^2.2.2` | Sesi autentikasi dan preferensi tampilan. |
| HTTP client | Axios `^1.7.2` | Request API dan interceptor token/error. |
| Build frontend | Vite `^5.4.10` | Dev server, bundling, dan lazy-loaded chunks. |
| Styling | Tailwind CSS `^3.4.14` | Utility CSS, responsivitas, dan dark mode. |
| Database server | PostgreSQL `16-alpine` | Penyimpanan data relasional. |
| Serving produksi | Nginx `1.27-alpine` | Menyajikan frontend dan reverse proxy API/SSE. |
| Orkestrasi lokal | Docker Compose | Menjalankan frontend, backend, dan database. |

Docker backend menggunakan Go `1.23-alpine` untuk build, sementara CI memilih Go `1.22`. Container frontend menggunakan Node `20-alpine` untuk build. Perbedaan toolchain ini berasal dari konfigurasi repositori, bukan dari pemilihan versi dalam dokumen ini.

## 3. Arsitektur dan Aliran Data

### 3.1 Komponen saat aplikasi dijalankan

```mermaid
flowchart TD
    U[Pengguna di browser] --> V[Vue SPA]
    V -->|HTTP JSON dan multipart| N[Nginx atau proxy development]
    N --> G[Gin HTTP API]
    G --> S[Service dan aturan bisnis]
    S --> R[Repository SQL]
    R --> P[(PostgreSQL)]
    G --> F[(Direktori uploads)]
    S --> B[Broker SSE dalam memori]
    B --> E[Endpoint GET /api/events]
    E --> N
    N -->|Event SSE ke browser| V
```

Pada Docker, browser mengakses Nginx melalui port `3000`. Request `/api/...` diteruskan ke backend pada jaringan internal Docker. Dalam pengembangan lokal, frontend dapat langsung mengakses backend `8080` atau menggunakan proxy Vite apabila base URL klien diatur menjadi `/api`.

### 3.2 Lapisan backend

Alur pemrosesan request normal:

```text
HTTP request
  → Router dan middleware
  → Handler
  → Service
  → Interface repository pada domain
  → Implementasi repository PostgreSQL
  → Database
```

Tanggung jawab setiap lapisan:

- **Domain:** mendefinisikan tipe data, request/response, error, dan kontrak interface.
- **Repository:** menjalankan SQL dan memetakan hasilnya menjadi objek domain.
- **Service:** memeriksa kepemilikan, role, transisi status, dan aturan bisnis lain.
- **Handler:** membaca input HTTP, memanggil service, dan menentukan bentuk/status response.
- **Middleware:** memvalidasi token dan menaruh identitas aktor pada context request.
- **Realtime:** mendistribusikan event kepada koneksi browser yang sesuai.
- **Entrypoint:** menyusun semua dependensi menjadi aplikasi yang dapat dijalankan.

Service tidak menerima `*sql.DB` langsung, tetapi interface seperti `domain.TicketRepository`. Karena itu, service yang sama bisa dipakai dengan repository PostgreSQL pada runtime atau repository mock saat test.

**Catatan arsitektur:** pemisahan ini bukan dependency inversion yang sepenuhnya ketat. `ticket_service.go` mengimpor paket `realtime` dan mengakses `realtime.DefaultBroker` secara langsung. Broker belum diinjeksi melalui interface seperti repository.

### 3.3 Alur satu perubahan data

Contoh agent memulai pengerjaan tiket:

```mermaid
sequenceDiagram
    actor A as Agent
    participant V as TicketDetailView
    participant H as Handler dan middleware
    participant S as TicketService
    participant R as TicketRepository
    participant D as PostgreSQL
    participant B as Broker SSE
    A->>V: Konfirmasi mulai pengerjaan
    V->>H: PATCH /api/tickets/:id
    H->>H: Verifikasi JWT dan bind JSON
    H->>S: Update(actor, id, request)
    S->>R: FindByID(id)
    R->>D: SELECT tiket
    D-->>S: Data tiket melalui repository
    S->>S: Validasi transisi dan auto-assign agent
    S->>R: Update(ticket)
    R->>D: UPDATE tickets
    S->>B: Broadcast ticket_updated
    S->>R: Muat ulang detail dan relasi
    S-->>V: Response data tiket melalui handler
    B-->>V: Event SSE untuk memicu refetch
```

Response request dan event SSE merupakan dua jalur berbeda. Event bukan pengganti response HTTP dan tidak membawa seluruh detail tiket.

## 4. Peta Direktori dan Tanggung Jawab Kode

```text
simple-journey-indonesia/
├── backend/
│   ├── cmd/api/main.go                 # Entry point dan penyusunan dependensi
│   ├── internal/
│   │   ├── config/config.go            # Pembacaan konfigurasi environment
│   │   ├── database/
│   │   │   ├── postgres.go             # Koneksi dan connection pool
│   │   │   ├── migrate.go              # Runner migrasi SQL ter-embed
│   │   │   ├── migrations/             # Tiga migrasi skema
│   │   │   └── seeder.go               # Akun dan tiket demo
│   │   ├── domain/
│   │   │   ├── entity.go               # User, Ticket, Comment, Attachment
│   │   │   ├── dto.go                  # DTO request, response, filter
│   │   │   ├── interfaces.go           # Kontrak service/repository
│   │   │   └── errors.go               # Error domain
│   │   ├── handler/
│   │   │   ├── router.go               # Pendaftaran endpoint dan CORS
│   │   │   ├── auth_handler.go         # Register, login, me
│   │   │   ├── ticket_handler.go       # Tiket, komentar, berkas, statistik, SSE
│   │   │   ├── user_handler.go         # Daftar user dan ubah role
│   │   │   └── response.go             # Envelope response dan mapping error
│   │   ├── middleware/auth_middleware.go
│   │   ├── realtime/broker.go          # Pub/sub SSE in-memory
│   │   ├── repository/                 # Empat implementasi repository SQL
│   │   └── service/
│   │       ├── auth_service.go
│   │       ├── ticket_service.go
│   │       ├── user_service.go
│   │       └── state_machine.go
│   ├── tests/                         # Test HTTP/service/broker dengan mock
│   ├── uploads/                       # Lokasi penyimpanan lokal default
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── main.js                    # Bootstrap Vue dan Pinia
│   │   ├── App.vue                    # Outlet root dan progress navigasi
│   │   ├── router/index.js            # Definisi route dan guards
│   │   ├── stores/                    # auth.js dan settings.js
│   │   ├── services/                  # Axios dan wrapper API/SSE
│   │   ├── constants/ticket.js         # Daftar nilai dan peta transisi UI
│   │   ├── i18n/index.js               # Kamus dan fungsi penerjemahan
│   │   ├── components/shared/         # Layout, badge, dialog, toast, dll.
│   │   ├── views/                     # Tujuh halaman aplikasi
│   │   ├── style.css                  # Styling dasar dan kelas bersama
│   │   └── dark.css                   # Override utility untuk tema gelap
│   ├── index.html
│   ├── vite.config.js
│   ├── tailwind.config.js
│   ├── nginx.conf
│   └── Dockerfile
├── docs/
│   ├── ARCHITECTURE.md
│   ├── PENJELASAN_SISTEM.md
│   └── postman_collection.json
├── .github/workflows/ci.yml
├── docker-compose.yml
└── README.md
```

`frontend/dist` adalah hasil build, `frontend/node_modules` berisi dependensi terpasang, dan `backend/api` bukan entrypoint kode sumber yang dipakai oleh `go run ./cmd/api`. Untuk memahami logika aplikasi, mulai dari direktori sumber, bukan artefak build. Direktori `frontend/src/composables` tidak ada pada struktur implementasi saat ini; fungsi reusable `useI18n()` dan `useRealtime()` berada di direktori masing-masing.

## 5. Proses Startup Backend

Sumber utama: [`backend/cmd/api/main.go`](../backend/cmd/api/main.go).

Fungsi `main()` berjalan dengan urutan berikut:

1. Memanggil `godotenv.Load()`. Tidak adanya `.env` tidak langsung menyebabkan error fatal karena hasilnya diabaikan.
2. Memanggil `config.Load()` untuk membaca dan memvalidasi konfigurasi dasar.
3. Membuka koneksi PostgreSQL melalui `database.Connect()`.
4. Menjalankan `database.Migrate()` sebelum menerima request.
5. Menjalankan `database.Seed()` jika `SeedOnStart` aktif.
6. Membuat repository user, tiket, komentar, dan lampiran.
7. Membuat service dan memberikan repository yang dibutuhkan.
8. Membuat handler dan menyusun router.
9. Menjalankan HTTP server pada port konfigurasi.

Cuplikan penyusunan dependensi:

```go
userRepo := repository.NewUserRepository(db)
ticketRepo := repository.NewTicketRepository(db)
commentRepo := repository.NewCommentRepository(db)
attachmentRepo := repository.NewAttachmentRepository(db)

authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiry)
ticketService := service.NewTicketService(ticketRepo, commentRepo, userRepo, attachmentRepo, cfg.SLAHours)
userService := service.NewUserService(userRepo)
```

Keterangan:

- Semua repository memakai connection pool `db` yang sama.
- `AuthService` membutuhkan repository user dan parameter JWT.
- `TicketService` membutuhkan empat repository karena detail tiket melibatkan pengguna, komentar, dan lampiran.
- `UserService` menangani daftar pengguna dan perubahan role.
- Constructor `New...` menyimpan dependensi ke dalam struct; tidak ada framework dependency injection khusus.

Penyusunan router:

```go
r := handler.SetupRouter(handler.RouterConfig{
    JWTSecret:     cfg.JWTSecret,
    AuthHandler:   authHandler,
    TicketHandler: ticketHandler,
    UserHandler:   userHandler,
})
```

`SetupRouter()` menggunakan `gin.New()`, mode release, dan `gin.Recovery()`. Recovery mengubah panic pada penanganan request menjadi kegagalan HTTP, bukan mekanisme pemulihan transaksi bisnis. Middleware logger Gin tidak dipasang secara eksplisit.

Server dijalankan memakai `http.ListenAndServe()`. Belum ada penanganan graceful shutdown maupun pengaturan timeout server secara eksplisit. Kegagalan config, koneksi database, migrasi, atau seed menyebabkan proses berhenti melalui `log.Fatalf()` sebelum server siap melayani.

## 6. Konfigurasi Aplikasi

### 6.1 Environment backend

Sumber perilaku: [`backend/internal/config/config.go`](../backend/internal/config/config.go).

| Variabel | Default di kode | Keterangan |
|---|---|---|
| `PORT` | `8080` | Port HTTP backend. |
| `DATABASE_URL` | Tidak ada; wajib | DSN koneksi PostgreSQL. |
| `JWT_SECRET` | Tidak ada; wajib | Kunci penandatanganan/verifikasi JWT. |
| `JWT_EXPIRY_HOURS` | `24` | Masa berlaku token dalam jam. |
| `SLA_HOURS` | `2` | Ambang umur tiket urgent-open yang dianggap overdue. |
| `ALLOWED_ORIGINS` | `http://localhost:5173` | Dibaca ke config, tetapi belum diterapkan pada middleware CORS. |
| `SEED_ON_START` | `true` | Aktif hanya jika nilai efektif sama persis dengan string `true`. |
| `UPLOAD_DIR` | `./uploads` | Folder penyimpanan byte lampiran. |

`getEnv()` memakai fallback jika variabel tidak ada atau bernilai kosong. `JWT_EXPIRY_HOURS` dan `SLA_HOURS` diparsing memakai `strconv.Atoi()`; teks nonangka ditolak, tetapi nilai nol/negatif belum ditolak secara khusus.

Docker Compose memberi `SLA_HOURS=2`, `SEED_ON_START=true`, `UPLOAD_DIR=/app/uploads`, dan origin frontend `http://localhost:3000`. Secret JWT serta kredensial database pada Compose adalah nilai demo; jangan menggunakannya untuk lingkungan produksi.

> Berkas `.env.example` terdeteksi tersedia, tetapi isinya tidak dibaca dalam penyusunan dokumen ini karena pembatasan berkas privat editor. Nilai dan perilaku pada tabel diambil dari `config.go` dan `docker-compose.yml`, bukan dari asumsi isi template environment.

### 6.2 Environment frontend

| Variabel | Pemakai | Fungsi |
|---|---|---|
| `VITE_API_BASE_URL` | `api.js`, `ticket.service.js`, `realtime.service.js` | Base URL API; fallback kode adalah `http://localhost:8080/api`. |
| `VITE_BACKEND_URL` | `vite.config.js` melalui `process.env` | Target proxy development `/api`; fallback `http://localhost:8080`. |

Keduanya berbeda:

- `VITE_API_BASE_URL=/api` membuat browser mengirim request relatif ke origin frontend, lalu Vite/Nginx dapat mem-proxy-nya.
- `VITE_API_BASE_URL=http://localhost:8080/api` membuat browser mengakses backend secara langsung.
- Proxy Vite tidak dipakai jika klien mengirim request ke URL absolut backend.
- `VITE_BACKEND_URL` dibaca sebagai environment proses yang menjalankan Vite; konfigurasi tidak memanggil `loadEnv()` untuk memuatnya dari `.env` secara eksplisit.
- Nilai `VITE_*` yang dipakai frontend dibundel saat build. Mengubah environment container Nginx setelah build tidak otomatis mengubah JavaScript yang telah dihasilkan.
- Jangan menyimpan secret di variabel `VITE_*`; nilai yang masuk bundle dapat dibaca pengguna browser.

## 7. Model Domain dan Struktur Data

Sumber: [`entity.go`](../backend/internal/domain/entity.go), [`dto.go`](../backend/internal/domain/dto.go), dan [`interfaces.go`](../backend/internal/domain/interfaces.go).

### 7.1 Konstanta bisnis

Backend mendefinisikan tipe string khusus:

```go
type UserRole string
type TicketPriority string
type TicketStatus string
```

Nilai yang digunakan aplikasi:

- Role: `requester`, `agent`, `admin`.
- Prioritas: `low`, `medium`, `high`, `urgent`; string kosong berarti belum ditentukan pada representasi Go/JSON.
- Status: `open`, `in_progress`, `resolved`, `closed`.

Tipe string khusus membantu keterbacaan kode, tetapi **bukan validasi enum otomatis**. Membuat `TicketPriority` dari string lain tetap mungkin; validasi nilai yang diterima API perlu dilihat terpisah dari deklarasi konstanta.

### 7.2 Entity utama

| Entity | Field utama | Relasi/field tambahan |
|---|---|---|
| `User` | `ID`, `Name`, `Email`, `PasswordHash`, `Role`, `CreatedAt` | `PasswordHash` tidak diserialisasi ke JSON. |
| `Ticket` | `ID`, `Title`, `Description`, `Category`, `Priority`, `Status`, `RequesterID`, `AssigneeID`, waktu | `Requester`, `Assignee`, `Comments`, `Attachments`, `IsOverdue`. |
| `Comment` | `ID`, `TicketID`, `AuthorID`, `Body`, `IsInternal`, `CreatedAt` | `Author` untuk nama/email pembuat komentar. |
| `Attachment` | `ID`, `TicketID`, `UploaderID`, `Filename`, `FilePath`, `FileSize`, `ContentType`, `CreatedAt` | `Uploader` untuk informasi pengunggah. |

Contoh tag yang penting:

```go
PasswordHash string `json:"-"`
```

Tag tersebut mencegah hash password ikut terkirim meskipun service menggunakan objek `User` dari repository. Namun `Attachment.FilePath` bertag `json:"file_path,omitempty"`, sehingga path storage yang terisi saat ini masih dapat masuk ke payload API; `omitempty` bukan perlindungan kerahasiaan path.

`IsOverdue` adalah hasil perhitungan saat data diminta, bukan kolom tersimpan pada tabel `tickets`.

### 7.3 DTO dan validasi input

**DTO** memisahkan kontrak request HTTP dari entity tersimpan. Contoh registrasi:

```go
type RegisterReq struct {
    Name     string `json:"name" binding:"required"`
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
}
```

Ketika handler memanggil `ShouldBindJSON()`, binding Gin memeriksa tag di atas. Pemeriksaan ini berada pada jalur HTTP; pemanggilan service secara langsung tidak otomatis menjalankan validator Gin.

`CreateTicketReq` mewajibkan `title`, `description`, dan `category`. `priority` bersifat opsional, sedangkan `requester_id` opsional dan hanya dipakai untuk pembuatan oleh admin.

Update tiket memakai pointer:

```go
type UpdateTicketReq struct {
    Status      *TicketStatus   `json:"status,omitempty"`
    Priority    *TicketPriority `json:"priority,omitempty"`
    AssigneeID  *string         `json:"assignee_id,omitempty"`
    RequesterID *string         `json:"requester_id,omitempty"`
}
```

Maknanya:

- Field tidak dikirim menghasilkan `nil`, sehingga service tidak mengubah field tersebut.
- Field dikirim sebagai string menghasilkan pointer non-`nil`.
- Field dikirim sebagai JSON `null` juga menghasilkan `nil` pada DTO baru ini.
- Karena itu, DTO ini **tidak membedakan field tidak dikirim dengan field bernilai `null`**.
- Untuk melepas assignee melalui implementasi backend saat ini, admin harus mengirim `"assignee_id":""` atau `"assignee_id":"unassigned"`, bukan `null`.

`title`, `description`, dan `category` tidak ada pada `UpdateTicketReq`, sehingga endpoint PATCH saat ini bukan fasilitas mengedit seluruh isi tiket.

### 7.4 Interface dan context

Contoh kontrak repository:

```go
type CommentRepository interface {
    Create(ctx context.Context, comment *Comment) error
    ListByTicket(ctx context.Context, ticketID string, includeInternal bool) ([]*Comment, error)
}
```

Service cukup mengetahui kontrak tersebut. Implementasi PostgreSQL dan mock test memenuhi method yang sama. `context.Context` diteruskan dari request HTTP ke `QueryContext()`/`ExecContext()`, sehingga operasi database dapat menerima sinyal pembatalan request. Aplikasi belum menetapkan deadline query tersendiri secara eksplisit.

## 8. Database, Migrasi, dan Data Awal

### 8.1 Hubungan antartabel

```mermaid
erDiagram
    users ||--o{ tickets : melaporkan
    users o|--o{ tickets : ditugaskan
    users ||--o{ comments : menulis
    users ||--o{ attachments : mengunggah
    tickets ||--o{ comments : memiliki
    tickets ||--o{ attachments : memiliki

    users {
        uuid id PK
        varchar name
        varchar email UK
        varchar password_hash
        varchar role
        timestamptz created_at
    }
    tickets {
        uuid id PK
        varchar title
        text description
        varchar category
        varchar priority
        varchar status
        uuid requester_id FK
        uuid assignee_id FK
        timestamptz created_at
        timestamptz updated_at
    }
    comments {
        uuid id PK
        uuid ticket_id FK
        uuid author_id FK
        text body
        boolean is_internal
        timestamptz created_at
        timestamptz updated_at
    }
    attachments {
        uuid id PK
        uuid ticket_id FK
        uuid comment_id FK "nullable"
        uuid uploader_id FK
        varchar filename
        varchar file_path
        bigint file_size
        varchar content_type
        timestamptz created_at
    }
```

Setiap tiket wajib memiliki satu pelapor, sedangkan assignee boleh belum ada. Lampiran dengan `comment_id` kosong adalah **lampiran tiket utama**, sedangkan lampiran dengan `comment_id` terisi adalah **lampiran komentar**. Keduanya tetap menyimpan `ticket_id` untuk keperluan akses dan pembersihan data.

### 8.2 Tabel `users`

| Kolom | Tipe | Aturan |
|---|---|---|
| `id` | UUID | Primary key; default SQL `gen_random_uuid()`. |
| `name` | VARCHAR(100) | Wajib. |
| `email` | VARCHAR(150) | Wajib dan unik. |
| `password_hash` | VARCHAR(255) | Wajib; berisi bcrypt hash. |
| `role` | VARCHAR(20) | Wajib; default `requester`. |
| `created_at` | TIMESTAMPTZ | Wajib; default waktu database. |

Service menormalisasi email menjadi huruf kecil sebelum simpan/login. Constraint unik database sendiri berlaku pada isi kolom yang disimpan; tabel tidak menggunakan tipe `citext`.

### 8.3 Tabel `tickets`

| Kolom | Tipe | Aturan setelah seluruh migrasi |
|---|---|---|
| `id` | UUID | Primary key. |
| `title` | VARCHAR(255) | Wajib. |
| `description` | TEXT | Wajib. |
| `category` | VARCHAR(100) | Wajib. |
| `priority` | VARCHAR(20) | Nullable dan tanpa default setelah migrasi `0003`. |
| `status` | VARCHAR(20) | Wajib; default `open`. |
| `requester_id` | UUID | Wajib; foreign key ke `users.id`. |
| `assignee_id` | UUID | Nullable; foreign key ke `users.id`, `ON DELETE SET NULL`. |
| `created_at` | TIMESTAMPTZ | Waktu pembuatan. |
| `updated_at` | TIMESTAMPTZ | Waktu update tiket; diperbarui aplikasi. |

Tidak ada trigger `updated_at`. Penambahan komentar/lampiran tidak otomatis menjalankan update pada row tiket melalui service saat ini.

### 8.4 Tabel `comments` dan `attachments`

`comments` menyimpan `id UUID`, `ticket_id UUID`, `author_id UUID`, `body TEXT`, `is_internal BOOLEAN DEFAULT FALSE`, `created_at TIMESTAMPTZ`, dan `updated_at TIMESTAMPTZ` (nullable, diisi saat komentar diedit).

`attachments` menyimpan `id UUID`, `ticket_id UUID`, `comment_id UUID` (nullable), `uploader_id UUID`, `filename VARCHAR(255)`, `file_path VARCHAR(500)`, `file_size BIGINT`, `content_type VARCHAR(100)`, dan `created_at TIMESTAMPTZ`. Semua kolom non-null kecuali `comment_id`.

Foreign key `comments.ticket_id` dan `attachments.ticket_id` menggunakan `ON DELETE CASCADE`. `attachments.comment_id` juga memakai `ON DELETE CASCADE`, sehingga metadata lampiran komentar ikut terhapus saat komentarnya dihapus. Foreign key penulis/pengunggah tidak menetapkan cascade. Penghapusan metadata attachment secara cascade pada SQL juga **tidak menghapus berkas fisik** di disk; service tetap mengumpulkan berkas lampiran komentar untuk dihapus dari disk. Saat ini aplikasi tidak menyediakan endpoint delete tiket.

### 8.5 Indeks dan constraint

Indeks eksplisit:

- `idx_tickets_requester` pada `tickets(requester_id)`.
- `idx_tickets_assignee` pada `tickets(assignee_id)`.
- `idx_tickets_status` pada `tickets(status)`.
- `idx_tickets_priority` pada `tickets(priority)`.
- `idx_comments_ticket` pada `comments(ticket_id)`.
- `idx_attachments_ticket` pada `attachments(ticket_id)`.

Primary key dan email unik juga menghasilkan indeks yang relevan di PostgreSQL. Tidak ada indeks pencarian judul khusus, SQL enum, atau `CHECK` constraint untuk membatasi daftar role/status/prioritas. Kategori juga berupa string, bukan tabel master kategori.

### 8.6 Connection pool dan retry

[`postgres.go`](../backend/internal/database/postgres.go) membuka `sql.DB`, lalu mengatur:

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)
```

Artinya maksimum 25 koneksi terbuka, maksimum 10 koneksi idle, dan umur koneksi dibatasi lima menit. `sql.DB` merupakan pool, bukan satu koneksi tunggal.

`Connect()` mencoba `db.Ping()` hingga 30 kali dengan jeda dua detik setiap kegagalan. Ini membantu saat PostgreSQL belum siap, tetapi bukan timeout total yang pasti 60 detik karena setiap pemanggilan `Ping()` sendiri memerlukan waktu.

### 8.7 Migrasi skema

Sumber: [`migrate.go`](../backend/internal/database/migrate.go).

```go
//go:embed migrations/*.sql
var migrationFiles embed.FS
```

Directive ini memasukkan SQL ke binary saat kompilasi. Container runtime tidak perlu menerima folder migrasi terpisah.

Urutan migrasi:

| Berkas | Efek |
|---|---|
| `0001_init.sql` | Membuat user, tiket, komentar, dan indeks awal. |
| `0002_attachments.sql` | Membuat tabel lampiran dan indeks tiketnya. |
| `0003_optional_priority.sql` | Menghapus `NOT NULL` dan default prioritas, mengubah string kosong lama menjadi `NULL`. |
| `0004_comment_attachments.sql` | Menambahkan `comment_id` pada lampiran, `updated_at` pada komentar, dan indeks lampiran per komentar. |

Runner membuat tabel `schema_migrations(version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ)`, mengurutkan nama berkas, lalu melewati migrasi yang sudah tercatat. Setiap migrasi baru dan pencatatan versinya dijalankan dalam transaksi yang sama. Jika eksekusi gagal, transaksi di-rollback.

Konsekuensi pengembangan:

- Tambahkan berkas migrasi baru untuk perubahan skema berikutnya.
- Mengubah isi migrasi yang sudah tercatat tidak membuatnya dijalankan ulang.
- Karena SQL di-embed, perubahan SQL memerlukan build/run ulang backend.
- Belum ada perintah down migration, checksum migrasi, atau lock migrasi lintas instance yang eksplisit.

### 8.8 Seeder dan akun demo

Sumber: [`seeder.go`](../backend/internal/database/seeder.go).

| Nama | Email | Role |
|---|---|---|
| Admin Helpdesk | `admin@helpdesk.local` | `admin` |
| Agent Helpdesk | `agent@helpdesk.local` | `agent` |
| Requester A (Budi) | `requester1@helpdesk.local` | `requester` |
| Requester B (Siti) | `requester2@helpdesk.local` | `requester` |

Password demo seluruh akun: `Password123!`. Nilai ini memang tertulis pada seeder dan fitur akun demo frontend; hanya untuk lingkungan demo.

Seeder:

1. Menghasilkan bcrypt hash password demo.
2. Mencari setiap akun berdasarkan email.
3. Menambahkan akun yang belum ada; akun yang sudah ada tidak di-reset password atau role-nya.
4. Menambahkan 13 tiket sampel, dengan pengecekan keberadaan berdasarkan judul.
5. Mengatur `created_at` mundur beberapa jam agar contoh berbagai umur/status tersedia.

Pada database benar-benar kosong, data sampel memiliki enam tiket `open`, dua `in_progress`, tiga `resolved`, dan dua `closed`. Satu tiket urgent-open berumur sekitar 30 jam sehingga overdue pada SLA default dua jam.

Pengecekan ini mencegah duplikasi pada startup berulang biasa, tetapi seed bukan satu transaksi global dan pengecekan tiket tidak didukung unique constraint judul. Jangan menganggapnya aman untuk eksekusi paralel banyak instance. Untuk produksi, nonaktifkan seed dan kelola akun administratif dengan prosedur yang sesuai.

## 9. Autentikasi dan Pengelolaan Sesi

### 9.1 Registrasi

Jalur kode:

```text
RegisterView.vue
  → auth store.register()
  → authService.register()
  → POST /api/auth/register
  → AuthHandler.Register()
  → AuthService.Register()
  → UserRepository
```

Langkah backend pada [`auth_service.go`](../backend/internal/service/auth_service.go):

1. Mengubah email menjadi lowercase dan memangkas spasi.
2. Memeriksa apakah email sudah ditemukan.
3. Membuat hash password dengan `bcrypt.DefaultCost`.
4. Membentuk `User` baru dengan UUID dan role **selalu `requester`**.
5. Menyimpan pengguna.
6. Menerbitkan JWT dan mengembalikan token serta profil pengguna.

Cuplikan inti:

```go
hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
```

Password mentah tidak disimpan ke tabel. Hash bcrypt dipakai karena pemeriksaan password tidak membutuhkan proses dekripsi.

Registrasi sukses langsung menghasilkan sesi, sehingga frontend tidak harus melakukan login terpisah. Jika email sudah ditemukan pada pemeriksaan awal, service mengembalikan `ErrConflict` yang dipetakan menjadi HTTP 409. Error unik database akibat request registrasi serentak belum dipetakan khusus menjadi conflict.

### 9.2 Login dan penerbitan JWT

Login menormalisasi email, mengambil user, lalu memanggil:

```go
bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
```

Email tidak ditemukan atau password tidak cocok sama-sama menghasilkan `ErrUnauthorized`, sehingga response tidak membedakan dua penyebab tersebut.

JWT dibuat oleh `generateToken()`:

```go
claims := jwt.MapClaims{
    "sub":   user.ID,
    "name":  user.Name,
    "email": user.Email,
    "role":  string(user.Role),
    "exp":   time.Now().Add(s.jwtExpiry).Unix(),
    "iat":   time.Now().Unix(),
}
t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
return t.SignedString([]byte(s.jwtSecret))
```

Arti claim:

- `sub`: ID pengguna.
- `name`, `email`, `role`: snapshot identitas ketika token diterbitkan.
- `exp`: waktu kedaluwarsa token.
- `iat`: waktu penerbitan.

JWT ditandatangani, bukan dienkripsi. Isi payload dapat dibaca oleh pemegang token; jangan memasukkan data rahasia ke claim.

### 9.3 Middleware autentikasi

Sumber: [`auth_middleware.go`](../backend/internal/middleware/auth_middleware.go).

Middleware `Auth()`:

1. Mengambil token dari `Authorization: Bearer ...`.
2. Jika header Bearer tidak tersedia, mencoba query `?token=...` untuk kebutuhan EventSource/unduhan.
3. Memverifikasi signature dan validitas token memakai library JWT.
4. Memastikan algoritma token termasuk keluarga HMAC.
5. Mengambil claim menjadi objek `domain.User`.
6. Menyimpan objek di Gin context dengan key `currentUser`.
7. Meneruskan request melalui `c.Next()`.

Token yang dibuat aplikasi adalah HS256. Middleware memeriksa tipe `SigningMethodHMAC`, bukan mengunci validasi hanya ke HS256.

Handler memanggil `middleware.CurrentUser(c)` untuk mendapatkan aktor. Identitas ini **dibentuk dari claim**, bukan diambil ulang dari database setiap request.

`RequireRoles()` juga didefinisikan sebagai middleware pembatas role, tetapi router saat ini tidak memasangnya pada route. Pemeriksaan role aktual dilakukan pada handler dan/atau service sesuai fitur.

### 9.4 Endpoint profil dan perubahan role

`GET /api/me` mengembalikan ID, nama, email, dan role dari context JWT. Endpoint ini tidak memuat user terbaru dari PostgreSQL.

Dampaknya:

- Perubahan role melalui admin memperbarui database, tetapi token lama masih membawa role lama sampai kedaluwarsa atau diganti saat login ulang.
- Memanggil `/me` menggunakan token lama tidak memperbarui snapshot claim menjadi role database terbaru.
- Tidak ada blacklist, refresh token, atau endpoint revoke/logout server.
- Koneksi SSE yang sudah terbuka tidak memverifikasi ulang JWT pada setiap event.

### 9.5 Sesi pada browser

Sumber: [`stores/auth.js`](../frontend/src/stores/auth.js) dan [`services/api.js`](../frontend/src/services/api.js).

Pinia menyimpan `token`, `user`, dan `loading`. Token dan user dipersist ke `localStorage` menggunakan `setAuth()`:

```js
setAuth(authData) {
  this.token = authData.token
  this.user = authData.user
  localStorage.setItem('token', authData.token)
  localStorage.setItem('user', JSON.stringify(authData.user))
}
```

`isAuthenticated` hanya memeriksa keberadaan token dan user lokal. Ini bukan pemeriksaan signature/kedaluwarsa. Router memakai nilai tersebut untuk pengalaman navigasi, sementara backend tetap memvalidasi request.

Request interceptor Axios menambahkan header:

```js
const token = localStorage.getItem('token')
if (token) {
  config.headers.Authorization = `Bearer ${token}`
}
```

Saat menerima HTTP 401, response interceptor menghapus sesi dan mengarahkan browser ke `/login`. Error lalu disederhanakan menjadi `Error(message)` untuk ditampilkan view. Penyederhanaan tersebut membuat pemanggil tidak lagi menerima objek Axios lengkap beserta status response-nya.

`fetchMe()` tersedia pada store tetapi belum dipanggil pada bootstrap. Bahkan jika dipanggil, batasan snapshot JWT pada `/me` tetap berlaku. Logout manual menghapus data lokal dan, melalui `MainLayout`, menutup koneksi SSE; token yang sudah diterbitkan tidak otomatis dicabut di server.

## 10. Hak Akses Berdasarkan Role

### 10.1 Matriks akses utama

| Aksi | Requester | Agent | Admin |
|---|---|---|---|
| Membuat tiket | Ya, sebagai diri sendiri | Tidak | Ya, termasuk atas nama user lain |
| Melihat daftar/detail tiket | Tiket milik sendiri | Seluruh tiket | Seluruh tiket |
| Menentukan prioritas saat create | Nilai kiriman diabaikan | Create ditolak | Bisa lewat API; UI create tidak menyediakan field |
| Mengubah prioritas setelah create | Tidak | Ya | Ya |
| Mengubah assignee manual | Tidak | Tidak | Ya |
| Mengubah requester/pelapor | Tidak | Tidak | Ya |
| Menulis komentar publik | Pada tiket sendiri | Ya | Ya |
| Menulis/membaca catatan internal | Tidak | Ya | Ya |
| Mengedit/menghapus komentar sendiri | Pada tiket sendiri, selama tiket belum Closed | Ya, selama tiket belum Closed | Ya, selama tiket belum Closed |
| Mengunggah/mengunduh/mempratinjau lampiran | Pada tiket sendiri | Ya | Ya |
| Mengelola lampiran komentar sendiri | Pada tiket sendiri, selama tiket belum Closed | Ya, selama tiket belum Closed | Ya, selama tiket belum Closed |
| Statistik umum | Hanya tiket sendiri melalui API | Seluruh tiket | Seluruh tiket |
| Statistik admin | Tidak | Tidak | Ya |
| Daftar pengguna | Tidak | Hanya daftar agent dan admin | Seluruh pengguna; filter role opsional |
| Mengubah role pengguna | Tidak | Tidak | Ya, dengan pemeriksaan admin terakhir |

Transisi status mengikuti tabel pada bagian 11, bukan sekadar prinsip bahwa admin boleh melakukan semuanya.

### 10.2 Isolasi requester

Saat daftar tiket diminta, service memaksa filter kepemilikan:

```go
if actor.Role == domain.RoleRequester {
    filter.RequesterID = actor.ID
}
```

Saat detail/update/komentar/lampiran diakses, service membandingkan `ticket.RequesterID` dengan `actor.ID`. Jadi keamanan objek tidak bergantung pada tombol frontend maupun sulit ditebaknya UUID.

**UUID tidak mencegah IDOR dengan sendirinya.** Perlindungan akses berasal dari verifikasi token, role, dan kepemilikan pada backend.

### 10.3 Ketentuan yang perlu diperhatikan

- Agent/admin tidak dibatasi hanya pada tiket yang ditugaskan kepadanya. Assignee adalah penugasan operasional, bukan pembatas akses staf.
- Agent yang mengambil tiket `open` akan menjadi assignee secara otomatis, termasuk jika sebelumnya tiket sudah memiliki assignee lain.
- Admin yang mengubah `open` ke `in_progress` tidak otomatis menjadi assignee.
- Admin dapat memilih user yang ada sebagai requester. Backend tidak mewajibkan target ber-role `requester`.
- Saat admin mengubah assignee, backend memeriksa keberadaan user, tetapi belum mewajibkan target ber-role agent/admin. UI memang menyaring pilihannya ke staf.
- Pada create oleh requester, `priority` dibuang dan `requester_id` yang bukan kewenangannya tidak dipakai, bukan selalu dibalas 403. Pada PATCH, perubahan field terlarang dengan nilai non-null ditolak.

### 10.4 Perubahan role dan admin terakhir

[`UserService.UpdateRole()`](../backend/internal/service/user_service.go) memeriksa aktor admin, mengambil user target, lalu menghitung jumlah admin bila aktor menurunkan role dirinya sendiri.

```go
if targetUser.ID == actor.ID && newRole != domain.RoleAdmin {
    adminCount, err := s.userRepo.CountByRole(ctx, domain.RoleAdmin)
    if err != nil {
        return err
    }
    if adminCount <= 1 {
        return domain.ErrForbidden
    }
}
```

Proteksi ini mencegah self-demotion admin terakhir pada alur biasa, tetapi belum memakai transaksi/locking untuk perubahan role serentak. Kode juga belum memvalidasi daftar nilai `newRole` dengan whitelist khusus, dan token role lama masih berlaku sebagaimana dijelaskan pada autentikasi.

## 11. Siklus Hidup dan Transisi Status Tiket

Sumber backend: [`state_machine.go`](../backend/internal/service/state_machine.go). Sumber UI: [`constants/ticket.js`](../frontend/src/constants/ticket.js).

| Status awal | Status tujuan | Role yang diizinkan | Makna |
|---|---|---|---|
| `open` | `in_progress` | `agent`, `admin` | Mulai menangani masalah. |
| `open` | `closed` | `requester`, `admin` | Membatalkan/menutup tiket yang belum diproses. |
| `in_progress` | `resolved` | `agent`, `admin` | Penanganan dinyatakan selesai. |
| `resolved` | `closed` | `requester`, `agent`, `admin` | Menutup tiket setelah penyelesaian. |
| `resolved` | `open` | `requester` | Masalah belum selesai atau terjadi kembali. |
| `closed` | Status lain | Tidak ada | Status akhir pada state machine. |

```mermaid
flowchart TD
    O[open] -->|agent atau admin mulai| P[in_progress]
    P -->|agent atau admin menyelesaikan| R[resolved]
    R -->|requester membuka kembali| O
    O -->|requester atau admin membatalkan| C[closed]
    R -->|requester, agent, atau admin menutup| C
```

`ValidateTransition(current, target, actor)` memeriksa status awal, status tujuan, dan role. Mengirim status yang sama tidak dianggap transisi ilegal. Perubahan status yang tidak sah menghasilkan HTTP 422 melalui mapping error domain.

Hal penting:

- Admin tidak dapat langsung mengubah `open` menjadi `resolved`.
- Admin tidak memiliki transisi `resolved` ke `open` pada aturan saat ini.
- Agent tidak dapat menutup tiket langsung dari `open`.
- Tidak ada transisi keluar dari `closed`.
- `closed` adalah status terminal. Diskusi dan lampiran komentar bersifat hanya-baca: penambahan, pengeditan, dan penghapusan komentar atau lampiran komentar ditolak dengan `ErrTicketClosed`. Komentar dan berkas lama tetap dapat dibaca, dipratinjau, dan diunduh.
- Lampiran tiket utama tetap dapat ditambah, diganti, atau dihapus oleh pemilik tiket meskipun tiket sudah `closed`; pembatas status hanya berlaku pada lampiran yang memiliki `comment_id`.
- Frontend `canTransition()` hanya menentukan tombol yang tampil. Validasi backend tetap menjadi keputusan akhir.

## 12. Cara Kerja Layanan Tiket

Sumber utama: [`ticket_service.go`](../backend/internal/service/ticket_service.go).

### 12.1 `Create()` — membuat tiket

Urutan:

1. Menolak role agent.
2. Mengosongkan prioritas bila aktor requester.
3. Menentukan requester default sebagai aktor.
4. Jika aktor admin mengirim `requester_id` nonkosong, memuat user tersebut dan menjadikannya pelapor.
5. Membentuk UUID, status `open`, `created_at`, dan `updated_at` dalam UTC.
6. Memanggil repository create.
7. Mengisi relasi requester pada objek response.
8. Mengirim event `ticket_created` hanya kepada agent/admin.
9. Mengembalikan tiket baru.

Create tidak menjalankan `Get()` lagi. Karena itu, response create tidak harus memiliki bentuk relasi yang identik dengan response detail; misalnya slice yang belum diisi dapat terserialisasi `null`.

### 12.2 `List()` — daftar tiket

Service menambahkan filter requester bila perlu, mengambil daftar dan total dari repository, lalu menghitung `IsOverdue` untuk setiap tiket yang memenuhi syarat SLA.

Metadata pagination:

```go
totalPages := int(math.Ceil(float64(total) / float64(filter.Limit)))
```

Dengan 23 tiket dan limit 10, `total_pages` menjadi 3. Nilai page/limit nonpositif dinormalisasi menjadi 1/10. Jika total nol, total halaman yang dihitung adalah nol.

Daftar memuat relasi requester/assignee, bukan seluruh thread komentar dan lampiran. Ini menghindari pemuatan semua diskusi untuk setiap baris daftar.

### 12.3 `Get()` — detail tiket

Urutan:

1. Memuat tiket beserta requester dan assignee.
2. Menolak requester yang bukan pemilik.
3. Menentukan apakah komentar internal boleh disertakan.
4. Memuat komentar terurut dari lama ke baru.
5. Memuat lampiran tiket.
6. Menghitung indikator overdue.
7. Mengembalikan objek detail.

Komentar dan lampiran didapat melalui query terpisah dari query tiket. Tidak ada transaksi snapshot khusus untuk keseluruhan pembacaan detail, sehingga perubahan bersamaan dapat terjadi di antara query-query tersebut.

### 12.4 `Update()` — status, prioritas, assignee, dan pelapor

Urutan pemeriksaan sengaja penting:

1. Memuat tiket saat ini.
2. Memeriksa kepemilikan requester.
3. Menolak field yang tidak boleh diubah role tersebut.
4. Memvalidasi transisi status jika status berubah.
5. Melakukan auto-assignment bila agent memulai pengerjaan.
6. Menerapkan prioritas staf.
7. Menerapkan assignee manual admin.
8. Menerapkan requester baru admin.
9. Memperbarui `UpdatedAt` dan menyimpan objek.
10. Memublikasikan `ticket_updated` kepada staf dan requester saat ini.
11. Memanggil `Get()` untuk mengembalikan detail terbaru sesuai akses aktor.

Cuplikan auto-assignment:

```go
if req.Status != nil && *req.Status != ticket.Status {
    if err := ValidateTransition(ticket.Status, *req.Status, *actor); err != nil {
        return nil, domain.ErrInvalidTransition
    }
    ticket.Status = *req.Status
    if *req.Status == domain.StatusInProgress && actor.Role == domain.RoleAgent {
        ticket.AssigneeID = &actor.ID
        ticket.Assignee = actor
    }
}
```

Satu request dapat mengirim beberapa field, tetapi proses read-validate-write belum memakai row lock, version column, atau kondisi status lama di SQL. Dua update yang bersamaan berpotensi saling menimpa. Ini berbeda dari data race Go: race detector tidak otomatis mendeteksi konflik logika pada database.

### 12.5 `Stats()`, `AdminStats()`, dan layanan pengguna

- `Stats()` meneruskan requester ID untuk membatasi ringkasan milik requester, atau string kosong untuk cakupan seluruh tiket staf.
- `AdminStats()` memeriksa role admin lalu meminta agregasi lengkap repository dengan parameter SLA.
- `UserService.List()` meneruskan daftar role ke repository. Pembatasan siapa boleh meminta daftar user dilakukan `UserHandler.List()`.
- `UserService.UpdateRole()` menangani pemeriksaan admin dan proteksi self-demotion yang dijelaskan sebelumnya.

## 13. Query SQL, Pencarian, Filter, dan Pagination

Sumber utama: [`ticket_repo.go`](../backend/internal/repository/ticket_repo.go).

### 13.1 SQL berparameter

Contoh insert:

```sql
INSERT INTO tickets
    (id, title, description, category, priority, status,
     requester_id, assignee_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9, $10)
```

Nilai dimasukkan melalui argumen `ExecContext()`, bukan disambung sebagai string SQL. Placeholder `$1`, `$2`, dan seterusnya membantu mencegah SQL injection untuk nilai input.

Untuk prioritas:

- `NULLIF($5, '')` mengubah string kosong Go menjadi SQL `NULL`.
- `COALESCE(t.priority, '')` pada SELECT mengubah SQL `NULL` menjadi string kosong agar dapat dipindai ke field string Go.

Untuk assignee yang nullable, pemuatan relasi memakai `LEFT JOIN` dan `sql.NullString`/`sql.NullTime`. Relasi requester memakai `JOIN` karena wajib ada.

### 13.2 Parameter daftar tiket

| Query parameter | Perilaku |
|---|---|
| `page` | Nomor halaman, default 1. |
| `limit` | Jumlah item per halaman, default 10. Belum ada batas maksimum eksplisit. |
| `status` | Filter kesamaan status. |
| `priority` | Filter kesamaan prioritas; `unset` berarti `NULL`/kosong. |
| `assignee_id` | ID user; `unassigned` untuk assignee null; `me` diterjemahkan handler menjadi ID aktor. |
| `search` | Pencarian judul tiket, case-insensitive dengan `LOWER(title) LIKE`. |
| `sort_by` | Whitelist `created_at`, `updated_at`, atau `priority`; fallback `created_at`. |
| `order` | `asc` menjadi ASC; selain itu DESC. |

`requester_id` pada `TicketFilter` merupakan filter internal yang dipaksakan service untuk requester; handler daftar tidak menyediakan query publik `requester_id`. Filter kategori juga belum tersedia pada handler daftar.

### 13.3 Penyusunan filter

Repository membangun `whereClauses`, daftar argumen, dan nomor placeholder secara bertahap. Contoh filter pemilik:

```go
whereClauses = append(whereClauses, fmt.Sprintf("t.requester_id = $%d", idx))
args = append(args, filter.RequesterID)
idx++
```

`fmt.Sprintf()` di sini membuat nomor placeholder, bukan memasukkan isi ID langsung ke SQL. Nama kolom sort dipilih lewat switch whitelist karena identifier SQL tidak bisa diperlakukan seperti nilai parameter biasa.

Query daftar dilakukan dua kali:

1. `SELECT COUNT(*)` dengan filter yang sama untuk jumlah total.
2. `SELECT` data dengan join, sort, limit, dan offset.

Rumus:

```text
offset = (page - 1) × limit
```

Pagination merupakan offset pagination, bukan cursor pagination. Tidak ada jaminan snapshot bersama antara count dan daftar jika data berubah bersamaan.

### 13.4 Semantik pencarian dan sort

- Search hanya mengenai **judul**, bukan deskripsi atau komentar.
- Search menggunakan `%kata%`. Karakter `%` dan `_` pada input tetap memiliki arti wildcard LIKE; berparameter berarti aman dari penyisipan sintaks SQL, bukan otomatis menjadi pencarian literal karakter wildcard.
- Sort `priority` mengikuti urutan string database, bukan bobot bisnis `urgent > high > medium > low`.
- Daftar kosong dapat menjadi `"data": null` karena beberapa slice repository tidak diinisialisasi sebagai array kosong. Konsumen sebaiknya tidak mengasumsikan setiap list selalu `[]`.

### 13.5 Repository lain

| Berkas | Method dan peran |
|---|---|
| `user_repo.go` | `Create`, `FindByID`, `FindByEmail`, `List`, `CountByRole`, `UpdateRole`. Daftar user diurutkan menurut nama. |
| `comment_repo.go` | Menyimpan komentar, join penulis, dan menambahkan filter internal sesuai flag service. |
| `attachment_repo.go` | Menyimpan metadata, memuat attachment beserta uploader per tiket, dan mencari attachment untuk download. |

Pencarian satu row menerjemahkan `sql.ErrNoRows` menjadi `domain.ErrNotFound`. Update user/tiket juga memeriksa jumlah row terdampak agar target yang tidak ada tidak dianggap berhasil.

## 14. Komentar dan Catatan Internal

Endpoint untuk diskusi:

| Method | Endpoint | Keterangan |
|---|---|---|
| POST | `/api/tickets/:id/comments` | Menambahkan komentar atau catatan internal. |
| PATCH | `/api/comments/:id` | Mengedit isi komentar milik sendiri. |
| DELETE | `/api/comments/:id` | Menghapus komentar milik sendiri beserta berkasnya. |
| POST | `/api/comments/:id/attachments` | Menambahkan berkas pada komentar milik sendiri. |

Contoh komentar publik:

```json
{
  "body": "Masalah masih terjadi setelah perangkat di-restart.",
  "is_internal": false
}
```

Contoh catatan staf:

```json
{
  "body": "Sedang memeriksa konfigurasi switch lantai tiga.",
  "is_internal": true
}
```

`TicketService.AddComment()` memuat tiket untuk memastikan objek ada dan aktor boleh mengaksesnya. Requester yang mencoba menulis internal note ditolak. Service membentuk ID, penulis, dan waktu; repository menyimpan komentar.

Saat detail dibaca:

```go
includeInternal := actor.Role == domain.RoleAgent || actor.Role == domain.RoleAdmin
comments, err := s.commentRepo.ListByTicket(ctx, id, includeInternal)
```

Repository kemudian menambahkan:

```sql
AND c.is_internal = FALSE
```

untuk requester. Dengan demikian isi catatan internal tidak sekadar disembunyikan oleh CSS atau komponen, tetapi tidak dimasukkan ke hasil query detail requester.

Batasan penting:

- Komentar bersifat daftar datar menurut waktu, bukan reply bertingkat.
- Komentar dapat diedit atau dihapus hanya oleh penulisnya. `TicketService.ownComment()` memeriksa `AuthorID` dan menolak aksi bila tiket berstatus `closed`.
- Komentar baru ditolak bila tiket berstatus `closed` (`ErrTicketClosed`).
- Saat komentar dihapus, berkas fisik lampiran komentar ikut dihapus; service mengembalikan daftar berkas untuk dibersihkan handler.
- Event `comment_added` untuk internal note tetap dikirim juga kepada requester pemilik tiket, tetapi payload hanya berisi `ticket_id`, bukan isi komentar. Requester dapat menerima toast perubahan meskipun catatan internalnya tidak terlihat.
- Berkas pada form komentar kini menjadi **lampiran komentar** (`comment_id` terisi), bukan lampiran tiket utama. Lampiran komentar mengikuti visibilitas komentarnya: catatan internal dan berkasnya tidak dikirim kepada requester.
- Berkas yang dipilih baru diunggah setelah komentar tersimpan dan memiliki ID. Jika unggah gagal, frontend menyimpan `pendingCommentId` agar tombol **Selesaikan Komentar** dapat mencoba unggah lagi tanpa menggandakan komentar.

## 15. Unggah, Pratinjau, dan Unduh Lampiran

Sumber handler: [`ticket_handler.go`](../backend/internal/handler/ticket_handler.go), [`ticket_content_handler.go`](../backend/internal/handler/ticket_content_handler.go), dan service `ticket_content.go`.

### 15.1 Penyimpanan

Sistem memisahkan:

- **Byte berkas:** direktori lokal `UPLOAD_DIR`.
- **Metadata:** tabel `attachments`, dengan `comment_id` kosong untuk lampiran tiket utama dan terisi untuk lampiran komentar.

Kedua jenis lampiran memakai direktori dan aturan ukuran yang sama, tetapi izin pengelolaannya berbeda: lampiran komentar hanya dapat diubah atau dihapus oleh penulis komentar, dan tidak lagi dapat diubah setelah tiket `closed`.

Batas ukuran per file pada handler adalah:

```go
const maxSize = 5 << 20
```

Nilainya 5.242.880 byte atau 5 MiB, walaupun pesan UI menyebut 5 MB.

Ekstensi yang diizinkan: `.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`, `.pdf`, `.txt`, `.csv`, `.doc`, `.docx`, `.xls`, `.xlsx`, `.zip`.

### 15.2 Alur upload

1. Middleware memvalidasi token.
2. Handler membaca field multipart bernama `file`.
3. Memeriksa ukuran `fileHeader.Size`.
4. Mengambil ekstensi nama file dan mencocokkannya dengan whitelist.
5. Membuat direktori upload jika belum ada.
6. Memakai `filepath.Base()` untuk mengambil nama dasar.
7. Menyusun nama storage dengan prefix UUID.
8. Menyimpan berkas ke disk.
9. Memanggil service untuk memeriksa akses dan menyimpan metadata: `AddAttachment()` untuk lampiran tiket utama atau `AddCommentAttachment()` untuk lampiran komentar. Keduanya memeriksa tiket induk dan, untuk lampiran komentar, kepemilikan komentar serta status tiket.
10. Menghapus berkas yang baru ditulis jika service gagal.
11. Mengirim event `attachment_added` dan response metadata jika berhasil.

Cuplikan penamaan:

```go
fileID := uuid.NewString()
sanitizedName := filepath.Base(fileHeader.Filename)
storedFilename := fmt.Sprintf("%s_%s", fileID, sanitizedName)
storedPath := filepath.Join(uploadDir, storedFilename)
```

UUID prefix membantu mencegah benturan nama, sedangkan `filepath.Base()` membuang komponen direktori pada nama berkas. ID prefix storage dan ID metadata attachment dibuat melalui pemanggilan UUID yang berbeda.

### 15.3 Validasi file yang benar-benar dilakukan

Handler memiliki map `allowedExt` dari ekstensi ke MIME type. `ContentType` diset dari nilai map itu, **bukan dari pemeriksaan byte file**. Kode saat ini tidak memanggil MIME sniffing, pemeriksaan signature dokumen, antivirus, atau pemeriksaan isi ZIP.

Karena itu, nama file berekstensi aman tidak membuktikan isi berkas aman. Pernyataan dokumentasi lama tentang deteksi MIME tidak menggambarkan implementasi handler saat ini.

Batas ukuran diperiksa setelah multipart diproses oleh `FormFile()`. Belum ada pembatas total body seperti `http.MaxBytesReader()` pada handler. Reverse proxy juga memiliki batasnya sendiri; lihat deployment dan troubleshooting.

### 15.4 Alur download dan pratinjau

Endpoint unduh: `GET /api/attachments/:id/download`. Endpoint pratinjau: `GET /api/attachments/:id/preview`.

1. Klien mengirim ID attachment beserta token.
2. Service memuat metadata, tiket terkait, dan memeriksa kepemilikan requester.
3. Untuk lampiran komentar, service juga memeriksa akses ke komentar induk. Berkas pada catatan internal dengan demikian tidak dapat diunduh atau dipratinjau oleh requester.
4. Download memasang `Content-Disposition: attachment`; pratinjau memasang `Content-Disposition: inline` dan MIME yang aman untuk ditampilkan browser.
5. `c.File()` mengirim isi disk.

Frontend memakai `window.open()` dengan query token karena navigasi tab baru tidak melewati interceptor Axios. Pratinjau di dalam aplikasi memakai `fetch()` dengan header `Authorization`, lalu menampilkan teks, gambar, PDF, atau pesan fallback sesuai `Content-Type`. Jika file fisik hilang, keberadaan metadata database saja tidak membuat unduhan maupun pratinjau berhasil.

Tidak ada layanan konversi atau penampil pihak ketiga; berkas dibaca langsung dari penyimpanan lokal backend dan diproses di browser pengguna.

### 15.5 Operasi gabungan di UI tidak atomik

Pada create, request membuat tiket dan request upload file dilakukan terpisah. Jika create sukses tetapi upload gagal, tiket tetap ada dan frontend tetap beralih ke detail; kegagalan lampiran hanya dicatat ke console pada alur tersebut.

Pada form komentar:

```js
if (!pendingCommentId.value) {
  const res = await ticketService.addComment(ticket.value.id, {
    body: newComment.body.trim(), is_internal: newComment.is_internal
  })
  pendingCommentId.value = res.data.id
}
if (commentFile.value) {
  await ticketService.uploadCommentAttachment(pendingCommentId.value, commentFile.value)
}
```

Komentar dibuat lebih dulu agar lampiran memiliki `comment_id`. Jika unggah gagal, komentar tidak dibatalkan; `pendingCommentId` dipertahankan dan tombol berubah menjadi **Selesaikan Komentar** sehingga pengguna dapat mencoba lagi tanpa menggandakan komentar. Ini konsekuensi dua operasi terpisah, bukan transaksi tunggal.

## 16. SLA dan Statistik Operasional

### 16.1 Definisi SLA aktual

Pada implementasi sekarang, tiket dianggap overdue jika seluruh kondisi berikut terpenuhi:

```text
status == open
DAN priority == urgent
DAN waktu sekarang - created_at > SLA_HOURS
```

Cuplikan service:

```go
if ticket.Status == domain.StatusOpen && ticket.Priority == domain.PriorityUrgent {
    if time.Since(ticket.CreatedAt) > s.slaDuration {
        ticket.IsOverdue = true
    }
}
```

Default `SLA_HOURS` adalah dua jam. **Tidak ada ambang terpisah 4/8/24/48 jam untuk semua prioritas pada kode saat ini.**

Konsekuensi:

- Tiket high/medium/low tidak otomatis overdue menurut aturan ini.
- Tiket urgent yang sudah `in_progress` tidak lagi memenuhi aturan overdue.
- Tiket yang dibuka kembali tetap memakai `created_at` awal; belum ada penghitung siklus SLA baru.
- Tidak ada scheduler, eskalasi, email pengingat, atau job background SLA.
- Badge berubah ketika detail/daftar dimuat ulang; frontend tidak menjalankan timer lokal khusus untuk mengganti badge pada detik melewati SLA.
- Sistem belum menyimpan waktu resolved/closed maupun histori pelanggaran SLA. Statistik overdue adalah kondisi saat ini, bukan riwayat breach.

### 16.2 Statistik umum

Endpoint `GET /api/stats` dan alias `GET /api/tickets/stats` mengembalikan:

- `total_tickets`.
- `open_tickets`.
- `in_progress_tickets`.
- `resolved_tickets`.
- `closed_tickets`.
- `urgent_tickets`.
- `overdue_tickets` sebagai field struct, tetapi query statistik umum **belum mengisinya**, sehingga nilainya nol pada jalur ini.

`urgent_tickets` menghitung prioritas urgent dengan status bukan `closed` dan bukan `resolved`; ini berbeda dari overdue yang hanya urgent-open lewat batas umur.

Requester mendapatkan statistik tiket miliknya. Staf mendapatkan statistik seluruh tiket. Kartu ringkasan pada halaman daftar hanya ditampilkan bagi non-requester dan tidak mengikuti filter tabel karena request statistik tidak menerima filter daftar.

### 16.3 Statistik admin

Endpoint `GET /api/admin/stats` memanggil `TicketRepository.AdminStats()`:

1. Mengambil ringkasan umum semua tiket.
2. Menghitung urgent-open overdue lewat query waktu PostgreSQL.
3. Mengelompokkan seluruh tiket menurut kategori.
4. Mengelompokkan prioritas yang terisi untuk tiket berstatus **bukan `closed`**; tiket resolved masih termasuk pada distribusi prioritas ini.
5. Menghitung beban setiap user ber-role **`agent`**, dengan tiket yang bukan resolved/closed. Agent dengan nol tiket tetap muncul melalui `LEFT JOIN`.

Contoh query overdue:

```sql
SELECT COUNT(*)
FROM tickets
WHERE priority = 'urgent'
  AND status = 'open'
  AND created_at < NOW() - make_interval(hours => $1)
```

Parameter berasal dari durasi SLA konfigurasi. Clock aplikasi digunakan pada detail/daftar, sedangkan clock PostgreSQL digunakan pada agregasi admin; sinkronisasi waktu server penting.

Statistik tidak menyediakan field `resolution_rate` atau `sla_breach_rate` yang dihitung khusus. Halaman admin saat ini menampilkan angka KPI, distribusi kategori/prioritas, dan tabel beban agent, bukan seluruh metrik yang pernah disebut pada README lama.

## 17. Notifikasi Real-Time dengan SSE

### 17.1 Mengapa SSE

**Server-Sent Events** adalah koneksi HTTP satu arah: server mengirim perubahan kepada browser. Pembuatan/update data tetap menggunakan HTTP biasa; SSE hanya memberi sinyal.

Dibanding WebSocket, kode ini tidak memerlukan kanal dua arah atau library tambahan. Browser memakai `EventSource`, sedangkan backend menulis format teks SSE ke response.

### 17.2 Broker backend

Sumber: [`realtime/broker.go`](../backend/internal/realtime/broker.go).

`Broker` menyimpan:

- `clients`: kumpulan subscriber aktif.
- `sync.RWMutex`: melindungi akses bersama ke kumpulan subscriber.
- Setiap client memiliki channel `[]byte`, role, dan ID pengguna.

`Subscribe()` membuat channel dengan buffer 32 pesan dan mengembalikan fungsi unsubscribe. `BroadcastTo()` mengubah payload menjadi JSON lalu membentuk:

```text
event: ticket_updated
data: {"id":"..."}

```

Baris kosong mengakhiri satu event. Contoh di atas adalah format ilustratif; ID aktual berbentuk UUID.

Pengiriman ke channel bersifat nonblocking:

```go
select {
case c.ch <- msg:
default:
    // Client slow or buffer full; drop to avoid blocking other clients
}
```

Jika browser terlalu lambat dan buffer penuh, event dibuang agar subscriber lain tidak terhambat. Karena itu, SSE bukan antrean pesan yang menjamin semua perubahan diterima.

### 17.3 Audience dan privasi event

`Audience` memuat daftar role dan user ID. Pencocokan menggunakan **OR**: role cocok atau user ID cocok cukup untuk menerima event. Audience kosong berarti semua subscriber.

| Event | Penerima | Payload |
|---|---|---|
| `ticket_created` | Semua agent/admin | `id`, `title`. |
| `ticket_updated` | Agent/admin dan requester tiket saat ini | `id`. |
| `comment_added` | Agent/admin dan requester tiket | `ticket_id`. |
| `attachment_added` | Agent/admin dan requester tiket | `ticket_id`. |

Requester lain tidak menerima event tiket tersebut. Ketika admin memindahkan pelapor, update dikirim kepada pemilik baru, bukan otomatis kepada pemilik lama. Event tidak menyertakan aktor, sehingga UI tidak dapat membedakan semua notifikasi perubahan sendiri dari perubahan orang lain hanya berdasarkan payload.

### 17.4 Handler streaming

`TicketHandler.StreamEvents()`:

1. Mengatur `Content-Type: text/event-stream`, `Cache-Control: no-cache`, dan `X-Accel-Buffering: no`.
2. Menulis komentar SSE awal `: connected` dan flush.
3. Subscribe memakai role serta user ID JWT.
4. Menunggu pesan, koneksi ditutup, atau ticker heartbeat.
5. Mengirim `: ping` setiap 25 detik.
6. Memanggil unsubscribe dan menghentikan ticker ketika handler berakhir.

Heartbeat berupa komentar SSE, bukan notifikasi tiket. Flush diperlukan agar pesan tidak tertahan di buffer writer.

Handler menggunakan loop `select` dan `c.Writer.Write()`/`Flush()`, bukan `c.Stream()` pada implementasi ini.

### 17.5 Klien frontend dan refetch

Sumber: [`realtime.service.js`](../frontend/src/services/realtime.service.js).

`useRealtime()` membagikan satu `EventSource` melalui variabel tingkat modul `_es`. `onEvent(type, fn)` mendaftarkan callback dan mengembalikan fungsi untuk melepasnya.

```js
_es = new EventSource(`${BASE_URL}/events?token=${encodeURIComponent(token)}`)
```

- `MainLayout` menampilkan toast global.
- `TicketsView` mengambil ulang daftar/statistik pada event tiket dibuat/diubah.
- `TicketDetailView` mengambil ulang detail bila ID pada event cocok dengan tiket yang dibuka.
- `AdminStatsView` tidak berlangganan refetch statistik otomatis; tombol refresh tetap diperlukan.
- Unsubscribe view melepas callback, bukan langsung menutup koneksi bersama.
- Logout manual layout memanggil `disconnect()`.

Komentar kode menyebut reconnect lima detik, tetapi tidak ada timer lima detik yang diimplementasikan. Pemulihan koneksi mengandalkan perilaku bawaan `EventSource`.

### 17.6 Batasan transport

Broker hanya ada di memori proses backend:

- Restart menghilangkan subscriber dan event yang belum diterima.
- Tidak ada penyimpanan event, event ID, atau replay `Last-Event-ID`.
- Dua instance backend memiliki broker berbeda; tanpa pub/sub bersama, event dari satu instance tidak otomatis sampai ke subscriber instance lainnya.
- Tidak ada validasi ulang token sepanjang stream yang sudah aktif.
- Setelah reconnect, klien tidak menjalankan mekanisme catch-up data khusus; refresh API tetap menjadi sumber keadaan terbaru.

## 18. Cara Kerja Frontend Vue

### 18.1 Bootstrap

Sumber: [`main.js`](../frontend/src/main.js), [`App.vue`](../frontend/src/App.vue), dan [`index.html`](../frontend/index.html).

`index.html` menyediakan `#app` dan memuat `src/main.js`. Sebelum Vue berjalan, script inline membaca tema tersimpan agar class `dark` dapat dipasang sebelum tampilan pertama.

Urutan `main.js`:

1. Membuat instance aplikasi melalui `createApp(App)`.
2. Memasang Pinia.
3. Membaca preferensi `lang` dan `theme` dari localStorage.
4. Menerapkan tema melalui settings store.
5. Memasang router.
6. Mount ke `#app`.

`style.css` dimuat terlebih dahulu, lalu `dark.css` agar override tema dapat berlaku sesudah style dasar.

### 18.2 Pola Composition API

Komponen menggunakan `<script setup>`, sehingga variabel/fungsi script dapat langsung digunakan template tanpa object `methods`/`data` terpisah.

Contoh aktual pada detail tiket:

```js
const ticket = ref(null)
const loading = ref(false)
const newComment = reactive({ body: '', is_internal: false })
const isAgentOrAdmin = computed(() => auth.isAgent || auth.isAdmin)
```

Keterangan:

- `ref()` menyimpan nilai reaktif; di JavaScript diakses dengan `.value`.
- `reactive()` dipakai untuk object form yang diubah field-nya.
- `computed()` menghitung nilai turunan berdasarkan dependensi reaktif.
- Template Vue membuka ref secara otomatis, sehingga cukup menulis `ticket.title` di template.
- `onMounted()` memulai pemuatan data/subscription.
- `onUnmounted()` melepaskan callback realtime agar tidak tetap memproses event setelah view dibuang.

Data tiket, komentar, daftar user, dan statistik dikelola lokal di masing-masing view. Tidak ada global ticket store atau library cache server-state khusus.

### 18.3 Routing dan guard

Sumber: [`router/index.js`](../frontend/src/router/index.js).

| Route | Halaman | Pembatas UI |
|---|---|---|
| `/login` | `LoginView.vue` | Guest-only. |
| `/register` | `RegisterView.vue` | Guest-only. |
| `/` | `MainLayout` lalu redirect | Memerlukan sesi lokal. |
| `/tickets` | `TicketsView.vue` | Semua role terautentikasi. |
| `/tickets/create` | `CreateTicketView.vue` | Requester/admin. |
| `/tickets/:id` | `TicketDetailView.vue` | Semua role terautentikasi; akses objek oleh API. |
| `/users` | `UsersView.vue` | Admin. |
| `/admin/stats` | `AdminStatsView.vue` | Admin. |
| URL lainnya | Redirect | Login atau landing sesuai role; tidak ada view 404 khusus. |

Route privat menjadi children `MainLayout` dan mewarisi metadata `requiresAuth`.

Guard utama:

```js
if (to.meta.requiresAuth && !auth.isAuthenticated) {
  return next({ name: 'login', query: { redirect: to.fullPath } })
}
```

Guard berikutnya mengalihkan pengguna yang sudah login dari halaman guest, lalu memeriksa `meta.roles`. Landing admin adalah `/admin/stats`; agent/requester `/tickets`. Login memakai query redirect sebelumnya jika tersedia.

Komponen route dimuat lewat dynamic import, misalnya:

```js
component: () => import('../views/TicketsView.vue')
```

Ini memungkinkan Vite memisahkan bundle halaman dan memuatnya saat diperlukan.

### 18.4 Wrapper service HTTP

| Berkas | Tanggung jawab |
|---|---|
| `services/api.js` | Membuat instance Axios, menyisipkan token, menangani 401/pesan error. |
| `services/auth.service.js` | Wrapper register, login, dan profil `/me`. |
| `services/ticket.service.js` | Wrapper tiket, komentar, lampiran, serta statistik. |
| `services/user.service.js` | Wrapper daftar pengguna dan update role. |
| `services/realtime.service.js` | Subscription SSE singleton dan callback event. |

Contoh wrapper:

```js
get(id) {
  return api.get(`/tickets/${id}`).then((r) => r.data)
}
```

Ada dua lapisan `data` yang perlu dibedakan:

1. `r.data` milik Axios adalah body HTTP.
2. Body API umumnya berbentuk `{ "data": ... }`.
3. Service membuka lapisan Axios, tetapi tetap mengembalikan envelope API.
4. View kemudian mengakses `res.data` untuk entity dan `res.meta` untuk pagination.

Ini bukan double-fetch atau response bertingkat yang tidak sengaja; keduanya mewakili objek berbeda.

### 18.5 Progress navigasi

`App.vue` memasang hook router untuk loading bar atas. `startBar()` memulai progres 8%, menaikkannya hingga mendekati 90%, lalu `finishBar()` menyelesaikannya ke 100%.

Loading bar tersebut adalah indikator perpindahan route, **bukan persentase penyelesaian request API**. View tetap menggunakan `loading` dan skeleton sendiri untuk proses pengambilan data.

## 19. Penjelasan Setiap Halaman Frontend

### 19.1 `LoginView.vue`

Fungsi penting:

- `fillDemo()`: mengisi form dari akun demo, bukan langsung login.
- `handleLogin()`: validasi field, memanggil store, lalu redirect sesuai tujuan/role.

Tombol submit menggunakan state loading agar tidak dikirim berulang selama proses tersebut. Error API tampil sebagai pesan. Kredensial demo hardcoded dimasukkan ke bundle frontend; fitur ini ditujukan untuk demo, bukan mekanisme penyembunyian akun.

### 19.2 `RegisterView.vue`

`handleRegister()` memeriksa nama/email/password, pola email sederhana, dan password minimal delapan karakter. Setelah `auth.register()` berhasil, pengguna diarahkan ke `/`, lalu router memilih landing.

Frontend tidak mengirim role untuk registrasi. Backend menetapkan requester secara eksplisit.

### 19.3 `TicketsView.vue`

State utama: `tickets`, `meta`, `stats`, `filters`, `assignees`, `expandedTickets`, loading, dan error.

| Fungsi | Kegunaan |
|---|---|
| `fetchTickets()` | Mengirim page, limit, search, status, priority, dan assignee ke API. |
| `fetchStats()` | Memuat ringkasan untuk non-requester. |
| `fetchAssignees()` | Memuat pilihan user staf untuk admin. |
| `handleSearchInput()` | Debounce pencarian 300 ms. |
| `applyFilters()` | Mengembalikan halaman ke 1 lalu mengambil daftar. |
| `resetFilters()` | Mengosongkan filter dan memuat ulang. |
| `changePage()` | Mengubah halaman aktif lalu meminta data. |
| `toggleExpand()` / `isExpanded()` | Membuka/mengecek ringkasan tiket. |

Desktop menggunakan tabel, sedangkan layar kecil menggunakan kartu. Kolom pelapor dan kartu statistik disembunyikan bagi requester. Tombol create disembunyikan bagi agent. Badge SLA menggunakan nilai backend `is_overdue`.

Setiap tiket yang deskripsinya dibuka diberi kelas `ticket-expanded-row` pada baris utama maupun baris ringkasan. Karena border pada `<tr>` tidak konsisten dengan `border-collapse`, penanda berupa garis biru digambar dengan `box-shadow: inset 4px 0 0 #0ea5e9` pada sel pertama. Dengan demikian semua deskripsi yang terbuka tetap bertanda, termasuk ketika beberapa baris dibuka sekaligus.

Pilihan assignee bernama hanya diambil untuk admin meskipun filter assignee juga terlihat bagi agent. Agent tetap memiliki opsi umum seperti `me` dan `unassigned`.

Event `ticket_created`/`ticket_updated` memicu request ulang. Filter dan nomor halaman tidak disimpan pada query URL; reload halaman memulihkan state awal view, bukan seluruh filter sebelumnya.

### 19.4 `CreateTicketView.vue`

Form mengirim judul, deskripsi, kategori, dan requester untuk admin. Kategori awal `IT Support`; pilihan kategori lain mengikuti daftar yang digunakan UI. Prioritas tidak disediakan di form create.

Fungsi penting:

- `handleFileSelected()`: validasi ukuran dan ekstensi satu file opsional.
- `removeFile()`: membersihkan pilihan file.
- `handleSubmit()`: membuat tiket, mengunggah file jika ada, lalu membuka detail.
- `formatFileSize()`: menampilkan ukuran file yang mudah dibaca.

Pilihan requester admin diisi dari seluruh user, bukan hanya role requester. Create dan upload tetap dua request, dengan konsekuensi partial success yang telah dijelaskan.

### 19.5 `TicketDetailView.vue`

Menampilkan judul/deskripsi, identitas pelapor, status/prioritas, badge SLA, lampiran, diskusi, dan kontrol staf/admin.

| Identifier | Tanggung jawab |
|---|---|
| `fetchTicket()` | Mengambil detail berdasarkan `route.params.id`. |
| `fetchUsers()` | Memuat assignee staf dan daftar pelapor untuk admin. |
| `availableActions` | Membentuk daftar tombol transisi dari status dan role. |
| `changeStatus()` | Menyiapkan isi dialog konfirmasi. |
| `confirmAction()` / `updateStatus()` | Melakukan PATCH status. |
| `updatePriority()` | PATCH prioritas untuk agent/admin. |
| `updateAssignee()` | PATCH assignee untuk admin. |
| `updateRequester()` | PATCH pelapor untuk admin. |
| `submitComment()` | Mengirim komentar terlebih dahulu, lalu lampiran komentar memakai `pendingCommentId`. |
| `handleCommentFileSelected()` | Memeriksa ukuran/ekstensi lampiran dari form komentar dan menyiapkan pratinjau lokal. |
| `handleFileUpload()` | Upload langsung dari bagian lampiran; pemeriksaan klien hanya ukuran. |
| `downloadAttachment()` | Membuka URL unduhan melalui service. |
| `previewAttachment()` | Mengambil URL pratinjau dan menampilkannya di komponen `FilePreview` (lihat `openTicketAttachmentPreview()` dan `openCommentAttachmentPreview()`). |
| `startEditComment()` / `cancelEditComment()` / `saveComment()` | Mengelola edit komentar milik sendiri. |
| `confirmDeleteComment()` | Menghapus komentar milik sendiri beserta berkasnya. |
| `showToast()` | Menampilkan notifikasi lokal sekitar tiga detik. |

Kontrol prioritas diturunkan dari:

```js
const canEditPriority = computed(() => {
  return auth.isAdmin || auth.isAgent
})
```

Komponen merender daftar komentar yang diterima API, tanpa menyaring internal note lagi untuk requester. Penyaringan keamanan memang dilakukan backend.

Lampiran tiket utama dan lampiran komentar ditampilkan pada bagian terpisah. Keduanya memiliki tombol **Pratinjau** dan **Unduh**; draft lampiran pada form komentar juga dapat dipratinjau secara lokal sebelum dikirim. Saat tiket berstatus `closed`, `isClosed` menyembunyikan formulir balasan, tombol edit/hapus komentar, dan tombol kelola lampiran komentar, serta menampilkan pemberitahuan diskusi hanya-baca. Lampiran tiket utama tetap dapat dikelola oleh pemiliknya.

**Perbedaan penting saat unassign:** frontend mengirim `assignee_id: assigneeId || null`. Backend memperlakukan `null` sebagai field tidak diubah. Akibatnya memilih opsi kosong pada UI tidak benar-benar melepas assignee pada implementasi saat ini, walaupun API backend menerima string kosong/`unassigned` untuk tujuan tersebut.

View belum memiliki watcher `route.params.id`. Bila router memakai ulang instance komponen detail saat hanya parameter ID berubah, pemuatan awal `onMounted()` tidak otomatis dijalankan lagi. Event SSE hanya memicu refetch untuk ID yang cocok dengan route saat itu.

### 19.6 `UsersView.vue`

- `fetchUsers()` mengambil seluruh pengguna.
- `changeRole(user, newRole)` mengirim PATCH, lalu memperbarui role pada row lokal.
- `savingId` menonaktifkan dropdown row yang sedang disimpan.
- Jika gagal, toast error ditampilkan dan daftar dimuat ulang.

Mengubah role akun sendiri tidak otomatis mengubah `auth.user`/localStorage/token. Menu browser dan otorisasi JWT dapat tetap merefleksikan role lama sampai sesi diterbitkan ulang.

### 19.7 `AdminStatsView.vue`

`fetchStats()` berjalan saat mount dan ketika tombol refresh ditekan. Data ditampilkan sebagai:

- Enam kartu KPI: total, open, in progress, resolved, closed, overdue.
- Bar distribusi kategori terhadap total tiket.
- Bar distribusi prioritas terhadap total kelompok prioritas yang dikembalikan.
- Tabel jumlah tiket per agent.

`calcPercent()` membulatkan persentase. Bar menggunakan CSS, bukan library grafik. Label beban agent adalah tinggi pada jumlah `>=5`, normal pada `>=2`, dan ringan di bawahnya; ini ambang presentasi frontend, bukan kapasitas yang disimpan database.

Tidak ada subscription refresh statistik otomatis pada halaman ini.

## 20. Komponen Bersama, Bahasa, dan Tema

### 20.1 `MainLayout.vue`

Layout halaman terautentikasi menyediakan:

- Sidebar desktop dan versi collapsed yang dipersist sebagai `sidebar_collapsed`.
- Drawer mobile.
- Navbar, dropdown pengguna, dan navigasi sesuai role.
- Toggle bahasa, tema, dan fullscreen browser.
- Outlet child `<router-view />`.
- Toast global dari event SSE.

`handleLogout()` menutup realtime, menghapus sesi, dan mengarahkan ke login. `toggleFullscreen()` pada layout memakai Fullscreen API browser. Action bernama sama pada settings store hanya mengubah boolean/persist state, bukan memanggil API browser.

### 20.2 Komponen UI reusable

| Komponen di `frontend/src/components/shared/` | Peran |
|---|---|
| `StatusBadge.vue` | Label dan warna status; indikator in-progress beranimasi. |
| `PriorityBadge.vue` | Label/warna prioritas dan fallback belum ditentukan. |
| `LoadingSkeleton.vue` | Placeholder loading dengan jumlah baris melalui prop `rows`. |
| `ErrorAlert.vue` | Pesan error dengan emit `retry` opsional. |
| `EmptyState.vue` | Penjelasan daftar kosong dan slot untuk aksi. |
| `ConfirmDialog.vue` | Konfirmasi aksi dengan emit `confirm`/`close`; dipasang ke body melalui Teleport. |
| `Toast.vue` | Notifikasi sukses/error; waktu tampil dikendalikan pemanggil. |

Pemisahan ini mengurangi duplikasi presentasi, tetapi sebagian label status/kategori/prioritas masih didefinisikan ulang pada view. `constants/ticket.js` belum menjadi satu-satunya sumber seluruh opsi UI.

### 20.3 i18n Indonesia/Inggris

Sumber: [`i18n/index.js`](../frontend/src/i18n/index.js).

Implementasi menggunakan kamus lokal, bukan package `vue-i18n`. `useI18n()` membaca settings store dan mengembalikan fungsi `t()`.

```js
const lang = settings.language || 'id'
let text = translations[lang]?.[key] || translations.id[key] || key
```

Urutan fallback: bahasa aktif → Indonesia → nama key. Parameter seperti `{label}` diganti dari object parameter yang diterima `t(key, params)`.

Bahasa dipersist pada key `lang`. Tidak semua teks sepenuhnya dinamis: beberapa label kategori/prioritas, teks fallback, dan format waktu masih tetap. View memakai locale tanggal `id-ID` meskipun bahasa diubah ke Inggris; zona waktu mengikuti browser. Atribut `lang` dokumen HTML juga belum ikut diperbarui.

### 20.4 Tema gelap/terang

Sumber: `stores/settings.js`, `style.css`, `dark.css`, dan `tailwind.config.js`.

```js
setTheme(theme) {
  this.theme = theme
  localStorage.setItem('theme', theme)
  document.documentElement.classList.toggle('dark', theme === 'dark')
}
```

Tailwind memakai `darkMode: 'class'`. Class `dark` pada root membuat utility `dark:*` dan override `.dark ...` aktif. `dark.css` juga memetakan ulang beberapa utility warna agar elemen lama mengikuti tema.

Default adalah light, bukan otomatis tema sistem operasi. Preferensi tema/bahasa dipulihkan pada bootstrap. Key `fullscreen` ditulis oleh store tetapi tidak dipulihkan pada bootstrap; status fullscreen browser sendiri ditangani layout.

## 21. Kontrak API dan Penanganan Error

### 21.1 Envelope response

Response sukses satu objek:

```json
{
  "data": {
    "status": "open",
    "priority": ""
  }
}
```

Contoh tersebut hanya menunjukkan sebagian field untuk menjelaskan envelope, bukan response tiket lengkap.

Response daftar tiket:

```json
{
  "data": [],
  "meta": {
    "total_items": 0,
    "total_pages": 0,
    "current_page": 1,
    "limit": 10
  }
}
```

`[]` di atas adalah contoh bentuk koleksi. Implementasi repository saat ini juga dapat mengembalikan `null` untuk koleksi kosong tertentu.

Response gagal:

```json
{
  "error": "Anda tidak memiliki akses untuk aksi ini"
}
```

### 21.2 Daftar endpoint

Semua path berikut berawalan `/api`. Endpoint selain login/register memerlukan autentikasi.

| Method | Endpoint | Handler | Hasil sukses utama |
|---|---|---|---|
| POST | `/auth/register` | `AuthHandler.Register` | 201, token dan user requester baru. |
| POST | `/auth/login` | `AuthHandler.Login` | 200, token dan user. |
| GET | `/me` | `AuthHandler.Me` | 200, profil dari claim token. |
| GET | `/tickets` | `TicketHandler.List` | 200, daftar sesuai akses/filter dan meta. |
| POST | `/tickets` | `TicketHandler.Create` | 201, tiket baru; agent ditolak. |
| GET | `/tickets/:id` | `TicketHandler.Get` | 200, detail dan relasi sesuai akses. |
| PATCH | `/tickets/:id` | `TicketHandler.Update` | 200, detail tiket setelah update. |
| POST | `/tickets/:id/comments` | `TicketHandler.AddComment` | 201, komentar baru; ditolak saat tiket Closed. |
| PATCH | `/comments/:id` | `TicketHandler.UpdateComment` | 200, komentar milik sendiri setelah diedit. |
| DELETE | `/comments/:id` | `TicketHandler.DeleteComment` | 204, komentar dan berkasnya dihapus. |
| POST | `/tickets/:id/attachments` | `TicketHandler.UploadAttachment` | 201, metadata lampiran tiket; multipart field `file`. |
| POST | `/comments/:id/attachments` | `TicketHandler.UploadCommentAttachment` | 201, metadata lampiran komentar; multipart field `file`. |
| GET | `/attachments/:id/download` | `TicketHandler.DownloadAttachment` | 200, isi file bila tersedia dan diizinkan. |
| GET | `/attachments/:id/preview` | `TicketHandler.PreviewAttachment` | 200, isi file untuk ditampilkan inline di aplikasi. |
| GET | `/events` | `TicketHandler.StreamEvents` | Stream `text/event-stream`. |
| GET | `/stats` | `TicketHandler.Stats` | 200, ringkasan sesuai cakupan user. |
| GET | `/tickets/stats` | `TicketHandler.Stats` | Alias `/stats`. |
| GET | `/admin/stats` | `TicketHandler.AdminStats` | 200, statistik lengkap; admin saja. |
| GET | `/users` | `UserHandler.List` | 200, daftar sesuai role aktor; requester ditolak. |
| PATCH | `/users/:id/role` | `UserHandler.UpdateRole` | 200, pesan sukses; admin saja. |

Parameter `:id` adalah parameter path, umumnya UUID pada database nyata. Menuliskan ID dengan format tidak valid belum selalu diterjemahkan menjadi 400/404 karena beberapa kegagalan cast PostgreSQL diteruskan sebagai error umum.

### 21.3 Contoh payload

Registrasi:

```json
{
  "name": "Dewi",
  "email": "dewi@example.com",
  "password": "contoh-password-kuat-untuk-demo"
}
```

Create tiket:

```json
{
  "title": "Tidak dapat mengakses VPN",
  "description": "Koneksi gagal sejak pukul 09.00 setelah memasukkan OTP.",
  "category": "Network"
}
```

Mulai pengerjaan oleh agent:

```json
{
  "status": "in_progress"
}
```

Menentukan prioritas oleh staf:

```json
{
  "priority": "urgent"
}
```

Melepas assignee melalui API sebagai admin:

```json
{
  "assignee_id": "unassigned"
}
```

Mengubah role sebagai admin:

```json
{
  "role": "agent"
}
```

### 21.4 Error domain dan status HTTP

Sumber: [`domain/errors.go`](../backend/internal/domain/errors.go) dan [`handler/response.go`](../backend/internal/handler/response.go).

| Kondisi | HTTP | Penjelasan |
|---|---|---|
| Binding/validasi request gagal | 400 | Field wajib/format JSON tidak sesuai. |
| Token hilang/tidak valid/expired; login salah | 401 | Pengguna belum terautentikasi secara sah. |
| Role atau kepemilikan tidak mengizinkan | 403 | Identitas sah tetapi aksi ditolak. |
| Resource tidak ditemukan | 404 | Target tidak ada menurut repository/service. |
| Email sudah terdaftar pada precheck | 409 | Konflik registrasi. |
| Transisi status ilegal | 422 | Request terbaca, tetapi perpindahan status tidak diperbolehkan. |
| Error lain | 500 | Pesan umum agar detail internal tidak langsung terungkap. |

`respondError()` memakai `errors.Is()` sehingga error domain yang dibungkus dengan `%w` tetap dapat dikenali. Binding error dan error middleware tertentu ditulis langsung oleh handler/middleware, tidak seluruhnya melalui helper yang sama.

Pesan kegagalan binding create masih menyebut prioritas wajib, padahal tag DTO tidak mewajibkan prioritas. Gunakan DTO dan perilaku service sebagai acuan kontrak aktual.

## 22. Contoh Alur Penggunaan Menyeluruh

### 22.1 Requester membuat laporan

1. Budi login menggunakan akun requester.
2. Token dan profil disimpan browser.
3. Budi membuka `/tickets/create` dan mengisi laporan.
4. Frontend mengirim `POST /api/tickets` dengan Bearer token.
5. Backend memvalidasi input, menetapkan pelapor dari token, status open, dan prioritas kosong.
6. Repository menyimpan tiket.
7. Agent/admin menerima event tiket baru; requester lain tidak menerima judul laporan Budi.
8. Jika Budi memilih file, frontend mengirim request upload terpisah.
9. Browser berpindah ke detail tiket dan mengambil detail lengkap.

### 22.2 Agent mengambil dan menyelesaikan tiket

1. Agent melihat tiket baru pada daftar atau toast.
2. Agent membuka detail dan menetapkan prioritas.
3. Agent memilih mulai pengerjaan dan mengonfirmasi dialog.
4. Backend memvalidasi `open → in_progress` dan otomatis menjadikan agent sebagai assignee.
5. Agent menambahkan komentar publik atau internal note untuk koordinasi.
6. Requester menerima perubahan relevan tetapi detailnya tidak berisi internal note.
7. Setelah selesai, agent mengubah `in_progress → resolved`.
8. Requester mengonfirmasi `resolved → closed`, atau membuka kembali `resolved → open` jika masalah belum selesai.

### 22.3 Admin mengatur operasional

1. Admin login dan langsung menuju `/admin/stats`.
2. Statistik menunjukkan kondisi saat diminta, bukan hasil langganan SSE otomatis halaman tersebut.
3. Admin dapat membuka daftar tiket, mengubah assignee/requester, dan mengelola role pengguna.
4. Memindahkan requester membuat kepemilikan akses mengikuti requester baru.
5. Role user yang diubah di database baru masuk ke JWT baru saat user login ulang; token lama tidak diubah secara retroaktif.

## 23. Menjalankan Aplikasi dan Deployment

### 23.1 Menggunakan Docker Compose

Jalankan dari root repositori pada terminal pengguna:

```sh
docker compose up --build
```

| Service | Alamat dari host | Alamat pada jaringan Compose |
|---|---|---|
| Frontend | `http://localhost:3000` | `frontend:80` |
| Backend | `http://localhost:8080/api` | `backend:8080` |
| PostgreSQL | `localhost:5433` | `db:5432` |

Docker Compose:

1. Menjalankan PostgreSQL dan healthcheck `pg_isready`.
2. Menjalankan backend setelah database sehat.
3. Backend tetap melakukan koneksi/retry, migrasi, dan seed sendiri.
4. Frontend bergantung pada startup backend, tetapi `depends_on` frontend bukan healthcheck kesiapan API.

Volume:

- `db_data`: data PostgreSQL.
- `uploads_data`: berkas lampiran backend.

Menghentikan container dengan `docker compose down` tidak sama dengan menghapus volume. Hindari `docker compose down -v` jika masih membutuhkan database dan lampiran karena opsi itu menghapus volume bernama yang dikelola Compose.

### 23.2 Menjalankan lokal tanpa container aplikasi

Prasyarat: PostgreSQL yang dapat diakses, Go yang kompatibel dengan modul, Node/npm yang kompatibel dengan frontend. Node 20 digunakan pada Docker/CI.

1. Jalankan database, misalnya dari root:

   ```sh
   docker compose up -d db
   ```

2. Masuk ke backend, salin template environment, lalu sesuaikan nilainya:

   ```sh
   cd backend
   cp .env.example .env
   ```

   Pastikan konfigurasi lokal setidaknya mencakup:

   ```dotenv
   PORT=8080
   DATABASE_URL=postgres://postgres:postgres@localhost:5433/helpdesk?sslmode=disable
   JWT_SECRET=ganti_dengan_secret_acak_yang_kuat
   JWT_EXPIRY_HOURS=24
   SLA_HOURS=2
   SEED_ON_START=true
   UPLOAD_DIR=./uploads
   ```

   Nilai secret di atas hanya penanda yang harus diganti, bukan secret yang disarankan. Contoh DSN memakai akun database demo Compose.

3. Jalankan backend pada terminal tersebut:

   ```sh
   go run ./cmd/api
   ```

4. Pada terminal lain dari root proyek, jalankan frontend:

   ```sh
   cd frontend
   npm ci
   VITE_API_BASE_URL=/api npm run dev
   ```

   Frontend tersedia pada `http://localhost:5173`. Setting `/api` membuat request menggunakan proxy development menuju backend `8080`.

Perintah server di atas memang berjalan terus sampai dihentikan oleh pengguna. Folder kerja backend penting karena `.env` dan `./uploads` menggunakan path relatif.

### 23.3 Build image backend

[`backend/Dockerfile`](../backend/Dockerfile):

1. Stage build mengunduh modul dan mengompilasi `./cmd/api` dengan `CGO_ENABLED=0` serta `GOOS=linux`.
2. Stage runtime menggunakan Alpine 3.20, menambahkan CA certificates dan user `appuser`.
3. Binary disalin ke `/app/server`.
4. Folder uploads dibuat dan dimiliki appuser.
5. Proses dijalankan sebagai user non-root.

Ini mengurangi ukuran/permukaan image dibanding membawa toolchain Go ke runtime, tetapi image runtime tetap Alpine, bukan image distroless.

### 23.4 Build image frontend dan Nginx

[`frontend/Dockerfile`](../frontend/Dockerfile):

1. Stage Node memasang dependency dengan `npm install`.
2. Build argument `VITE_API_BASE_URL` dipakai ketika Vite menghasilkan `dist`.
3. Stage Nginx menyalin aset build dan konfigurasi proxy.

[`frontend/nginx.conf`](../frontend/nginx.conf) menangani:

- `/api/` → proxy ke `http://backend:8080/api/`.
- `/api/events` → location khusus tanpa buffering/cache dan read timeout 3600 detik.
- Path SPA → `try_files ... /index.html`, sehingga membuka langsung `/tickets/:id` tidak menjadi 404 file biasa.
- Aset statis tertentu → cache tujuh hari dengan header immutable.

Konfigurasi belum menetapkan `client_max_body_size`. Default Nginx adalah `1m` apabila tidak dioverride konfigurasi lain, sehingga upload melalui frontend dapat tertolak 413 sebelum mencapai batas 5 MiB backend. Saat menyiapkan deployment, periksa konfigurasi efektif Nginx dan tetapkan batas yang mengakomodasi file serta overhead multipart.

### 23.5 Kebutuhan operasional produksi

Sebelum penggunaan nyata, setidaknya tinjau:

- Secret JWT dan kredensial database yang kuat, bukan nilai Compose demo.
- Nonaktifkan seed dan kredensial demo frontend.
- HTTPS di terminasi proxy.
- CORS yang benar-benar memakai whitelist origin.
- Backup database **dan** direktori/volume uploads sebagai pasangan.
- Kebijakan token ketika role berubah atau pengguna dinonaktifkan.
- Kapasitas disk, limit body, validasi isi file, dan logging.
- Strategi migrasi dan realtime jika backend lebih dari satu instance.

Daftar ini merupakan kebutuhan peninjauan, bukan fitur yang sudah semuanya diterapkan.

## 24. Pengujian dan Pipeline CI

### 24.1 Struktur test

Sumber: direktori [`backend/tests`](../backend/tests).

| Berkas | Cakupan |
|---|---|
| `ticket_test.go` | Auth service, isolasi tiket, transisi, internal note, hak agent/admin, SLA, akses lampiran. |
| `api_test.go` | Request melalui router Gin, binding, JWT, handler, service, upload/download, dan streaming SSE dasar. |
| `realtime_test.go` | Pemilihan audience event broker. |
| `mocks.go` | Repository berbasis map/slice untuk menggantikan PostgreSQL. |

Terdapat 18 fungsi test top-level pada sumber saat ini:

| Kelompok | Test |
|---|---|
| HTTP | `TestAPI_AuthEndpoints`, `TestAPI_TicketsEndpoints`, `TestAPI_CommentsEndpoint`, `TestAPI_AttachmentsUploadAndDownload`, `TestAPI_StatsEndpoints`, `TestAPI_UsersEndpoints`, `TestAPI_EventsEndpoint`. |
| Service | `TestAuthRegisterAndLogin`, `TestTicketIsolation`, `TestValidStatusTransitions`, `TestInvalidStatusTransitions`, `TestInternalNoteSecurity`, `TestAgentCannotCreateTicket`, `TestAgentCanManagePriority`, `TestAdminCanChangeRequester`, `TestSLAOverdueCalculation`, `TestAttachmentUploadAndAccess`. |
| Broker | `TestBrokerAudienceScoping`. |

`setupTestEnv()` menyusun router/handler/service nyata dengan repository mock, dan direktori sementara untuk upload. Test HTTP memakai `httptest`, bukan membutuhkan database produksi.

### 24.2 Apa yang belum dibuktikan test tersebut

Test yang ada bukan integration test PostgreSQL nyata:

- Migrasi dan query repository SQL tidak dieksekusi terhadap database sungguhan oleh harness ini.
- Mock daftar tiket hanya menerapkan sebagian filter dan tidak meniru seluruh pagination/sort SQL.
- Mock statistik tidak identik dengan seluruh agregasi SQL; misalnya cakupan urgent pada mock berbeda untuk tiket resolved.
- Test upload HTTP memeriksa jalur sukses dan penolakan ekstensi `.exe`; bukan bukti adanya validasi isi MIME/antivirus atau cakupan seluruh batas ukuran.
- Test SLA memeriksa urgent-open tiga jam dengan SLA dua jam, bukan semua prioritas.
- Test pengguna yang ada memeriksa penolakan requester dan update role, tetapi tidak mencakup seluruh skenario admin terakhir/concurrency.
- Test SSE API memeriksa stream dapat dibuka/ditutup dan content type; broker memiliki test audience tersendiri.
- Frontend belum memiliki script unit test, E2E test, atau lint pada `package.json`.

### 24.3 Perintah pengujian

Dari direktori `backend`:

```sh
go test -v -race ./...
go vet ./...
```

Menjalankan subset untuk isolasi masalah:

```sh
go test ./tests -run TestTicketIsolation -v
go test ./tests -run TestSLAOverdueCalculation -v
go test ./tests -run TestBrokerAudienceScoping -v
```

Dari direktori `frontend`:

```sh
npm ci
npm run build
```

Build frontend memeriksa kemampuan Vite mengompilasi/bundle proyek, bukan otomatis membuktikan akses role, upload proxy, atau perilaku browser berjalan benar.

### 24.4 Pipeline GitHub Actions

Sumber: [`.github/workflows/ci.yml`](../.github/workflows/ci.yml).

Workflow berjalan pada push/pull request ke branch `master` dan `main` dengan dua job:

1. **Backend:** setup Go 1.22, `go vet ./...`, lalu `go test -v -race ./...`.
2. **Frontend:** setup Node 20, `npm ci`, lalu `npm run build`.

Nama job frontend menyebut typecheck, tetapi langkahnya tidak menjalankan TypeScript checker terpisah. Workflow juga belum membangun/publish image, mengirim aplikasi ke server, atau melakukan deployment. Jadi yang tersedia adalah **CI**, bukan pipeline CD lengkap.

### 24.5 Validasi saat penyusunan dokumen

- `npm run build` pada frontend selesai dengan sukses.
- Percobaan backend `go test -race ./...` menggunakan cache build sementara mencapai batas 120 detik sebelum ringkasan hasil paket test tersedia. Keluaran yang sempat terlihat hanya paket yang tidak memiliki test. Hasil keseluruhan backend **belum dapat dinyatakan lulus atau gagal** dari percobaan itu.
- Tidak dilakukan pengujian browser, deployment Docker penuh, atau integration test PostgreSQL nyata dalam penyusunan dokumentasi ini.

Catatan ini merupakan snapshot validasi saat dokumen dibuat, bukan pengganti menjalankan CI pada perubahan berikutnya.

## 25. Batasan Implementasi dan Catatan Keamanan

Bagian ini merangkum perilaku yang penting diketahui agar dokumen tidak memberi kesan kemampuan yang belum ada. Ini bukan audit keamanan menyeluruh.

| Area | Perilaku saat ini | Dampak/catatan |
|---|---|---|
| CORS | Router selalu mengirim origin `*`; `AllowedOrigins` tidak diterapkan. | Environment whitelist tidak membatasi origin aktual. Kombinasi wildcard dan credentials juga bukan konfigurasi credentialed CORS yang benar. |
| Perubahan role | Middleware dan `/me` memakai claim JWT tanpa lookup database. | Token lama mempertahankan hak lama; login ulang diperlukan untuk token baru. |
| Validasi enum | Tidak ada whitelist menyeluruh untuk role/prioritas dan tidak ada CHECK SQL. | API dapat menerima nilai di luar opsi UI pada field tertentu; konstanta Go saja tidak mencegahnya. |
| Ownership | Verifikasi requester dilakukan service. | Ini perlindungan utama terhadap akses objek lintas requester; UUID bukan penggantinya. |
| Upload | Whitelist ekstensi dan ukuran; MIME ditentukan dari ekstensi. | Isi berkas tidak diverifikasi, belum ada antivirus/scan arsip. |
| Privasi lampiran | Lampiran milik tiket, tanpa flag internal. | File pada form internal note tetap dapat dilihat pemilik tiket. |
| Token URL | SSE/download memakai `?token=`. | Hindari pencatatan query token pada log proxy/monitoring dan gunakan HTTPS. |
| Token browser | JWT disimpan di localStorage. | Script berbahaya akibat XSS dapat membaca token; tidak ada cookie HttpOnly dalam desain ini. |
| Metadata file | `file_path` dapat ikut response JSON. | Detail path storage belum disembunyikan dari klien. |
| Unassign UI | UI mengirim `null`, service menganggap tidak ada perubahan. | Gunakan API string kosong/`unassigned` sampai ketidaksesuaian kontrak diperbaiki. |
| Multi-request | Create+upload dan komentar+upload terpisah. | Partial success mungkin terjadi; retry perlu menghindari duplikasi. |
| Konflik update | Read-modify-write tanpa version lock. | Request bersamaan berpotensi lost update/validasi berdasarkan status lama. |
| Admin terakhir | Pengecekan count sebelum self-demotion tanpa lock. | Belum aman terhadap seluruh kombinasi konkurensi/token role lama. |
| SLA | Urgent-open saja, memakai umur dari create. | Tidak ada SLA semua prioritas atau histori breach/resolution time. |
| Realtime | Broker memori, event best-effort dan tanpa replay. | Tidak cocok dianggap durable message bus atau sinkron lintas replica. |
| Batas proxy | `client_max_body_size` tidak diset di Nginx proyek. | Batas efektif proxy dapat lebih kecil dari 5 MiB backend. |
| Validasi ukuran input | Tag required/email/min terbatas; banyak panjang field tidak divalidasi di DTO. | Input melewati kapasitas VARCHAR dapat menjadi error database umum. |
| Rate limiting | Tidak ada limiter login/register/upload eksplisit. | Perlindungan brute force dan penyalahgunaan sumber daya perlu ditambahkan untuk produksi. |
| Logging/observability | Belum ada observability operasional terstruktur lengkap. | Investigasi kegagalan/kapasitas memerlukan peningkatan logging dan metrik. |
| Frontend async | Refetch SSE dan request pengguna dapat berjalan bersamaan. | Belum ada cancellation/deduplikasi untuk mencegah response lama menimpa yang baru. |

Selain itu, beberapa timer/listener frontend belum memiliki cleanup lengkap, dialog belum memiliki pengelolaan fokus penuh, dan state autentikasi dari JSON localStorage belum dilindungi dari isi storage yang rusak. Hal-hal tersebut tidak diubah oleh dokumen ini.

**Perbedaan utama dari dokumentasi lama yang harus diingat:** SLA bukan per-prioritas 4/8/24/48 jam, UUID bukan pencegah IDOR otomatis, upload belum melakukan sniffing MIME, tidak ada folder composables yang dipakai, KPI rate belum diimplementasikan sebagai metrik khusus, dan workflow belum melakukan CD.

## 26. Panduan Membaca dan Mengembangkan Kode

### 26.1 Urutan membaca yang disarankan

Untuk backend:

1. `domain/entity.go`: pahami objek dan konstanta.
2. `domain/dto.go`: pahami input/output yang tersedia.
3. `domain/interfaces.go`: lihat kontrak antarlapisan.
4. `cmd/api/main.go`: lihat hubungan dependensi.
5. `handler/router.go`: temukan endpoint fitur.
6. Handler terkait: pahami input HTTP.
7. Service terkait: pahami aturan bisnis.
8. Repository terkait: pahami query dan penyimpanan.
9. Migrasi: pastikan struktur database sesuai.
10. Test: lihat skenario yang sudah memiliki verifikasi otomatis.

Untuk frontend:

1. `main.js` → `App.vue` → `router/index.js`.
2. `stores/auth.js` dan `services/api.js`.
3. `MainLayout.vue` untuk kerangka aplikasi.
4. View yang relevan dan wrapper service-nya.
5. `constants/ticket.js`, `i18n/index.js`, dan komponen shared.

### 26.2 Contoh menambah field tiket

Jika ingin menambah field bisnis baru, perubahan biasanya melibatkan:

1. Migrasi SQL baru.
2. Entity dan DTO jika field masuk kontrak API.
3. Query INSERT/SELECT/UPDATE dan mapping `Scan()` repository.
4. Validasi/kewenangan perubahan pada service.
5. Form/tampilan frontend.
6. Terjemahan label bila diperlukan.
7. Mock repository dan test.
8. Dokumentasi API/Postman jika kontrak berubah.

Jangan hanya menambah field pada entity: query SQL memakai daftar kolom eksplisit, sehingga penyimpanan dan pembacaan tidak otomatis mengikuti perubahan struct.

### 26.3 Contoh menambah status atau aturan transisi

Periksa bersama-sama:

- Konstanta backend.
- `ValidateTransition()`.
- Peta `TRANSITIONS` frontend.
- Pilihan filter dan `StatusBadge`.
- Query statistik yang menghitung status tertentu.
- Semantik SLA/workload.
- Seeder, mock, dan test.

Frontend dan backend memiliki peta transisi terpisah. Mengubah salah satunya saja dapat menampilkan tombol yang selalu ditolak API, atau menyembunyikan aksi yang sebenarnya diizinkan backend.

### 26.4 Menambah notifikasi real-time

1. Tentukan kapan perubahan dianggap berhasil tersimpan.
2. Pilih audience secara eksplisit, bukan broadcast ke semua pengguna.
3. Hindari memasukkan isi sensitif ke payload.
4. Daftarkan event baru pada listener `realtime.service.js` bila jenisnya baru.
5. Tambahkan handler refetch/toast pada view yang relevan.
6. Tambahkan test pemilihan audience dan jangan menganggap event pasti diterima.

## 27. Troubleshooting dan Glosarium

### 27.1 Masalah yang sering dijumpai

| Gejala | Pemeriksaan awal |
|---|---|
| Backend gagal sebelum server aktif | Pastikan `DATABASE_URL` dan `JWT_SECRET` tersedia; periksa log koneksi/migrasi/seed. |
| Database lokal tidak bisa diakses | Dari host Compose memakai port `5433`, bukan `5432`; dari container backend gunakan host `db:5432`. |
| Frontend memanggil backend yang salah | Periksa `VITE_API_BASE_URL`, apakah URL absolut atau `/api`, dan lakukan rebuild bila bundle sudah dibuat. |
| Mengubah `ALLOWED_ORIGINS` tidak mengubah CORS | Nilai config belum dipakai router; perilaku aktual berasal dari middleware CORS hardcoded. |
| Agent tidak bisa membuat tiket | Ini aturan bisnis yang disengaja, bukan kegagalan login. |
| Requester tidak melihat tiket rekan | Ini isolasi kepemilikan yang disengaja. |
| Admin/agent mendapat 422 saat status diubah | Cocokkan status awal, target, dan role dengan matriks transisi. |
| Prioritas kosong pada tiket baru | Memang diperbolehkan; requester tidak menentukan prioritas. |
| Tombol unassigned tidak melepas penugasan | Perbedaan `null` frontend versus string kosong/`unassigned` backend. |
| Upload kecil berhasil tetapi file lebih besar mendapat 413 | Periksa batas body Nginx/proxy sebelum memeriksa batas 5 MiB handler. |
| Download gagal setelah pindah server | Pastikan volume upload dipindah bersama database dan `file_path` tetap valid. |
| Internal note tidak muncul pada requester | Isi internal note memang disaring backend; lampiran tidak mengikuti privasi komentar. |
| High priority lama tidak ditandai overdue | SLA aktual hanya urgent-open, bukan semua prioritas. |
| Statistik admin tidak berubah meskipun ada toast | Halaman statistik tidak memiliki subscription auto-refetch; gunakan refresh. |
| Hak user belum berubah setelah admin mengedit role | Token masih menyimpan role lama; login ulang menerbitkan claim baru. |
| SSE tidak sampai | Periksa token, koneksi `/api/events`, konfigurasi tanpa buffering, audience, dan instance backend yang dipakai. |
| Halaman detail lama tetap tampil saat ID route berubah | View belum mengawasi perubahan parameter ID pada instance komponen yang dipakai ulang. |

### 27.2 Istilah

| Istilah | Arti dalam sistem ini |
|---|---|
| SPA | Single Page Application; navigasi utama dilakukan Vue di browser. |
| DTO | Objek kontrak input/output antarlapisan/API. |
| RBAC | Role-Based Access Control; kewenangan berdasarkan role. |
| Ownership | Pemeriksaan bahwa requester adalah pemilik tiket. |
| JWT | Token bertanda tangan yang membawa claim identitas. |
| bcrypt | Algoritma hash password untuk penyimpanan dan verifikasi. |
| Repository | Lapisan akses data yang menjalankan SQL. |
| Service | Lapisan aturan bisnis dan koordinasi repository. |
| State machine | Daftar perpindahan status yang sah beserta syarat role. |
| SSE | Stream HTTP satu arah dari server ke browser. |
| Broker | Penghubung publisher event dan subscriber aktif. |
| Audience | Cakupan role/user yang boleh menerima suatu event. |
| SLA | Ambang layanan; pada kode ini urgent-open melewati umur tertentu. |
| Migration | Perubahan struktur database yang dicatat versinya. |
| Seeder | Pengisi data awal/demo. |
| Reverse proxy | Perantara HTTP yang meneruskan request browser ke backend. |
| Idempotensi | Eksekusi ulang tidak menggandakan efek pada kondisi yang ditangani. |
| Partial success | Sebagian operasi sudah berhasil ketika operasi lanjutannya gagal. |
| Lost update | Perubahan tertimpa oleh request lain yang berangkat dari data lama. |
| IDOR | Akses objek berdasarkan ID tanpa verifikasi kewenangan yang semestinya. |

### Referensi repositori

- [README dan panduan ringkas](../README.md)
- [Catatan arsitektur awal](ARCHITECTURE.md)
- [Dokumen kebutuhan produk](../PRD_INTERNAL_HELPDESK_SYSTEM.md)
- [Dokumen desain](../DESIGN.md)
- [Koleksi Postman](postman_collection.json)
- [Docker Compose](../docker-compose.yml)

Apabila uraian rencana pada dokumen kebutuhan atau catatan lama berbeda dari kode, gunakan implementasi saat ini sebagai acuan perilaku yang tersedia, lalu verifikasi dengan test yang sesuai sebelum menjadikannya kontrak baru.

---

# Pendalaman Kode API dan Relasinya dengan Frontend

Bagian 28–39 melengkapi penjelasan sistem dengan penelusuran **aksi pengguna → fungsi Vue → service HTTP → endpoint → handler → service bisnis → repository → tabel → response → state → komponen**. Tujuannya bukan hanya mengetahui fungsi setiap berkas, tetapi juga memahami data yang mengalir dan dampak perubahan antarbagiannya.

Rujukan baris kode menunjuk posisi saat lampiran ini disusun dan dapat bergeser pada perubahan selanjutnya. Contoh JSON merupakan data ilustratif, bukan hasil pengambilan data pengguna atau token sungguhan. Diagram menunjukkan alur logis; penjadwalan response dan event SSE pada runtime tidak dijamin berurutan sama persis.

## 28. Peta Lengkap Relasi Endpoint dan Pemakainya

### 28.1 Inventaris seluruh endpoint

Definisi route ada pada [`handler/router.go`](../backend/internal/handler/router.go#L35-L66). Terdapat **16 pasangan method-path**. Dua path statistik umum menggunakan handler yang sama, sedangkan statistik admin merupakan operasi berbeda.

| No. | Endpoint | Wrapper frontend | Pemakai aktif/jalur frontend | Handler backend | Service bisnis |
|---|---|---|---|---|---|
| 1 | `POST /api/auth/register` | `authService.register()` | `RegisterView.handleRegister()` → `auth.register()` | `AuthHandler.Register()` | `AuthService.Register()` |
| 2 | `POST /api/auth/login` | `authService.login()` | `LoginView.handleLogin()` → `auth.login()` | `AuthHandler.Login()` | `AuthService.Login()` |
| 3 | `GET /api/me` | `authService.me()` | Hanya direferensikan `auth.fetchMe()`; action ini belum dipanggil UI/bootstrap | `AuthHandler.Me()` | Tidak ada; membaca context JWT |
| 4 | `GET /api/tickets` | `ticketService.list()` | `TicketsView.fetchTickets()` | `TicketHandler.List()` | `TicketService.List()` |
| 5 | `POST /api/tickets` | `ticketService.create()` | `CreateTicketView.handleSubmit()` | `TicketHandler.Create()` | `TicketService.Create()` |
| 6 | `GET /api/tickets/:id` | `ticketService.get()` | `TicketDetailView.fetchTicket()` | `TicketHandler.Get()` | `TicketService.Get()` |
| 7 | `PATCH /api/tickets/:id` | `ticketService.update()` | Empat fungsi update pada `TicketDetailView` | `TicketHandler.Update()` | `TicketService.Update()` |
| 8 | `POST /api/tickets/:id/comments` | `ticketService.addComment()` | `TicketDetailView.submitComment()` | `TicketHandler.AddComment()` | `TicketService.AddComment()` |
| 9 | `POST /api/tickets/:id/attachments` | `ticketService.uploadAttachment()` | Create, balasan dengan file, dan upload langsung di detail | `TicketHandler.UploadAttachment()` | `TicketService.AddAttachment()` |
| 10 | `GET /api/attachments/:id/download` | `ticketService.downloadAttachment()` | Tombol download pada detail | `TicketHandler.DownloadAttachment()` | `TicketService.GetAttachment()` |
| 11 | `GET /api/stats` | `ticketService.stats()` | `TicketsView.fetchStats()` untuk non-requester | `TicketHandler.Stats()` | `TicketService.Stats()` |
| 12 | `GET /api/tickets/stats` | Tidak ada wrapper khusus | Tidak diminta frontend saat ini; alias server | `TicketHandler.Stats()` | `TicketService.Stats()` |
| 13 | `GET /api/admin/stats` | `ticketService.adminStats()` | `AdminStatsView.fetchStats()` | `TicketHandler.AdminStats()` | `TicketService.AdminStats()` |
| 14 | `GET /api/users` | `userService.list()` | Create admin, filter assignee admin, detail admin, dan manajemen user | `UserHandler.List()` | `UserService.List()` |
| 15 | `PATCH /api/users/:id/role` | `userService.updateRole()` | `UsersView.changeRole()` | `UserHandler.UpdateRole()` | `UserService.UpdateRole()` |
| 16 | `GET /api/events` | `useRealtime()` melalui `EventSource` | Layout, daftar tiket, dan detail tiket | `TicketHandler.StreamEvents()` | Tidak ada; subscribe broker langsung |

`RegisterView`, `LoginView`, dan nama view lain dalam tabel merujuk berkas `.vue` pada `frontend/src/views/`, bukan class JavaScript yang diekspor. Identifier setelah titik adalah fungsi pada `<script setup>` atau action store yang terkait.

### 28.2 Relasi modul frontend

```mermaid
flowchart TD
    L[LoginView dan RegisterView] --> A[Pinia auth store]
    A --> AS[auth.service.js]
    T[TicketsView] --> TS[ticket.service.js]
    C[CreateTicketView] --> TS
    D[TicketDetailView] --> TS
    ST[AdminStatsView] --> TS
    C --> US[user.service.js]
    D --> US
    T --> US
    U[UsersView] --> US
    AS --> AX[api.js dan interceptor Axios]
    TS -->|Operasi JSON dan upload| AX
    US --> AX
    AX --> API[Gin API]
    TS -->|Download melalui window.open| API
    M[MainLayout] --> RT[realtime.service.js]
    T --> RT
    D --> RT
    RT -->|EventSource dengan query token| API
```

Dua jalur tidak melewati `api.js`: download via tab baru dan stream `EventSource`. Keduanya memiliki konsekuensi autentikasi/error yang berbeda dari request Axios.

### 28.3 Relasi modul backend

- `AuthHandler` menerima `domain.AuthService`.
- `TicketHandler` menerima `domain.TicketService` dan lokasi upload.
- `UserHandler` menerima `domain.UserService`.
- `AuthService` memakai `domain.UserRepository`.
- `TicketService` memakai repository tiket, user, komentar, dan lampiran.
- `UserService` memakai repository user.
- Service tiket memublikasikan event melalui `realtime.DefaultBroker`.
- Handler SSE berlangganan broker yang sama, bukan menjalankan query SQL.

Kontrak method dapat dilihat pada [`domain/interfaces.go`](../backend/internal/domain/interfaces.go#L5-L54). Instance konkret disusun pada [`cmd/api/main.go`](../backend/cmd/api/main.go#L42-L60).

## 29. Mekanisme Request dan Response Lintas Lapisan

### 29.1 URL halaman berbeda dari URL API

Contoh saat membuka halaman detail:

```text
URL halaman: /tickets/10000000-0000-4000-8000-000000000001
  → Vue Router memilih TicketDetailView
  → fetchTicket() berjalan
  → request API: GET /api/tickets/10000000-0000-4000-8000-000000000001
```

Vue Router menentukan komponen yang ditampilkan. Gin Router menentukan operasi data di server. Mengunjungi halaman tidak berarti browser langsung menampilkan JSON API.

Dalam Docker, Nginx menyajikan halaman dan meneruskan `/api`. Dalam development dengan base URL `/api`, Vite mem-proxy request ke backend. Jika memakai fallback absolut `http://localhost:8080/api`, browser menghubungi backend langsung.

### 29.2 Axios, header, dan tiga bentuk input

[`services/api.js`](../frontend/src/services/api.js#L4-L15) membuat satu instance:

```js
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api',
  headers: { 'Content-Type': 'application/json' }
})
```

Bentuk input API:

| Jenis input | Contoh frontend | Cara handler membacanya | Contoh data |
|---|---|---|---|
| Path parameter | ``api.get(`/tickets/${id}`)`` | `c.Param("id")` | ID tiket/attachment/user. |
| Query string | `api.get('/tickets', { params })` | `c.Query()` / `c.DefaultQuery()` | Page, limit, filter, search. |
| JSON body | `api.patch(..., { status })` | `c.ShouldBindJSON(&req)` | Status, prioritas, isi komentar. |
| Multipart body | `FormData` dengan field `file` | `c.FormFile("file")` | Byte file dan nama aslinya. |
| Header | Interceptor menambahkan Authorization | `c.GetHeader("Authorization")` | Token Bearer. |
| Query token | EventSource/window.open | `c.Query("token")` pada middleware | Fallback token untuk transport tanpa header khusus. |

Request interceptor mengambil token dari localStorage setiap request, bukan dari argumen fungsi view. Token juga dapat ikut pada login/register bila storage masih berisi token; endpoint autentikasi sendiri tidak membutuhkan token tersebut.

### 29.3 Relasi body Axios dengan envelope backend

Cuplikan [`ticket.service.js`](../frontend/src/services/ticket.service.js#L4-L8):

```js
list(params = {}) {
  return api.get('/tickets', { params }).then((r) => r.data)
},
get(id) {
  return api.get(`/tickets/${id}`).then((r) => r.data)
}
```

Urutan pembukaan response:

```text
Response Axios r
├── status: 200
├── headers: ...
└── data: body JSON API
    ├── data: entity atau array
    └── meta: pagination, khusus daftar tiket

Service mengembalikan r.data → body JSON API
View menerima res → body JSON API
View membaca res.data → entity/array
View membaca res.meta → pagination
```

Backend memakai [`respondData()`](../backend/internal/handler/response.go#L35-L37):

```go
func respondData(c *gin.Context, status int, data interface{}) {
    c.JSON(status, gin.H{"data": data})
}
```

Khusus list tiket, handler mengembalikan `data` dan `meta` pada level body yang sama. Jadi bukan `res.data.meta`.

### 29.4 Relasi handler, service, dan repository

Contoh pola handler update:

```go
var req domain.UpdateTicketReq
if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Payload tidak valid"})
    return
}
```

Setelah JSON terbaca, handler memberikan `c.Request.Context()`, pengguna dari middleware, ID path, dan DTO ke service. Service menentukan boleh/tidaknya aksi. Repository tidak mengetahui komponen Vue atau HTTP status; repository menerima nilai Go dan mengembalikan entity/error.

`QueryRowContext().Scan()` mengisi field struct dari urutan kolom SELECT. `ExecContext()` menjalankan INSERT/UPDATE. Keduanya menggunakan argumen SQL berparameter. Ketidaksesuaian urutan `Scan()` dengan SELECT adalah masalah repository, bukan masalah binding JSON frontend.

### 29.5 Error kembali melalui jalur yang sama

```text
Repository/service mengembalikan error
  → handler respondError()
  → HTTP 400/401/403/404/409/422/500 + {error: pesan}
  → Axios response interceptor
  → Promise.reject(new Error(pesan))
  → catch pada view/store
  → error inline, ErrorAlert, toast, atau fallback sunyi
```

401 juga menghapus sesi dan menjalankan navigasi penuh ke `/login`. Karena interceptor membungkus ulang error sebagai `Error`, view tidak lagi menerima `error.response.status` asli. Download/SSE tidak melalui alur ini.

## 30. Bedah API Autentikasi dan Integrasi Frontend

### 30.1 POST /api/auth/register

**Rantai kode:**

```text
RegisterView.handleRegister()
  → useAuthStore().register(form)
  → authService.register(payload)
  → AuthHandler.Register()
  → AuthService.Register()
  → UserRepository.FindByEmail()
  → UserRepository.Create()
  → generateToken()
  → AuthResponse
```

Sumber frontend: [`RegisterView.vue`](../frontend/src/views/RegisterView.vue#L59-L83), [`stores/auth.js`](../frontend/src/stores/auth.js#L27-L35), [`auth.service.js`](../frontend/src/services/auth.service.js#L7-L8). Backend: [`auth_handler.go`](../backend/internal/handler/auth_handler.go#L20-L33), [`auth_service.go`](../backend/internal/service/auth_service.go#L30-L71).

**Relasi field form dengan DTO dan database:**

| Frontend | DTO Go | Pemeriksaan/transformasi | Database |
|---|---|---|---|
| `form.name` | `RegisterReq.Name` | Required pada HTTP; service memangkas spasi | `users.name`. |
| `form.email` | `RegisterReq.Email` | Required/email; trim dan lowercase | `users.email` unik. |
| `form.password` | `RegisterReq.Password` | Required/minimal 8 pada HTTP; bcrypt | `users.password_hash`, bukan password asli. |
| Tidak ada | Tidak ada input role | Service menetapkan `RoleRequester` | `users.role = requester`. |
| Tidak ada | Tidak ada input ID | UUID dibuat backend | `users.id`. |

Frontend memeriksa ketiga field, pola email sederhana, dan panjang password sebelum mengirim form. Pemeriksaan frontend memperbaiki pengalaman pengguna; validasi HTTP dan aturan service tetap diperlukan untuk klien selain UI.

Response 201 menggunakan `AuthResponse`:

```json
{
  "data": {
    "token": "TOKEN_JWT_ILUSTRATIF_BUKAN_TOKEN_ASLI",
    "user": {
      "id": "20000000-0000-4000-8000-000000000001",
      "name": "Pelapor Contoh",
      "email": "requester@example.test",
      "role": "requester",
      "created_at": "2026-01-01T08:00:00Z"
    }
  }
}
```

Store menjalankan `setAuth(res.data)`. `token` masuk state dan localStorage; `user` menjadi sumber nama, role, menu, dan guard browser. View kemudian menuju `/`; router memilih `/tickets` untuk requester.

**Error dan efek samping:**

- Form invalid: tidak ada request dari handler frontend.
- Binding HTTP invalid: 400 dan pesan error.
- Email sudah ditemukan pada precheck: 409.
- Penyimpanan gagal: error diteruskan, tidak ada sesi baru pada frontend.
- Create user dan penerbitan token tidak berada pada satu transaksi bisnis; jika tahap setelah penyimpanan gagal, jangan otomatis menganggap row user belum dibuat.
- Registrasi tidak menyiarkan event user baru melalui SSE.

### 30.2 POST /api/auth/login

Sumber: [`LoginView.vue`](../frontend/src/views/LoginView.vue#L95-L113), [`stores/auth.js`](../frontend/src/stores/auth.js#L17-L25), [`AuthHandler.Login`](../backend/internal/handler/auth_handler.go#L35-L48), [`AuthService.Login`](../backend/internal/service/auth_service.go#L73-L100).

Alur:

1. Submit form memanggil `handleLogin()`.
2. Email/password kosong ditolak frontend.
3. Store memanggil `authService.login({ email, password })`.
4. Handler melakukan binding `LoginReq`.
5. Service memuat `users` menurut email ternormalisasi.
6. bcrypt membandingkan password kiriman dengan hash.
7. JWT dibuat dari identitas/role user database saat itu.
8. Response memiliki bentuk yang sama seperti registrasi.
9. Store menyimpan sesi.
10. View menuju query `redirect` bila tersedia, atau landing sesuai role.

```mermaid
sequenceDiagram
    actor U as Pengguna
    participant V as LoginView
    participant A as Auth store dan service HTTP
    participant H as AuthHandler
    participant S as AuthService
    participant R as UserRepository
    U->>V: Submit email dan password
    V->>A: login(credentials)
    A->>H: POST /api/auth/login
    H->>S: Login(LoginReq)
    S->>R: FindByEmail(email ternormalisasi)
    R-->>S: User beserta hash password
    S->>S: bcrypt compare dan tanda tangani JWT
    S-->>A: AuthResponse melalui handler
    A->>A: Simpan token dan user
    A-->>V: Promise selesai
    V->>V: Navigasi sesuai redirect atau role
```

`auth.user.role` yang diterima menentukan tombol/route browser. Backend tidak mempercayai role dari localStorage; backend mempercayai claim token yang telah diverifikasi. Namun claim tersebut juga merupakan snapshot saat token diterbitkan, bukan pembacaan role terbaru setiap request.

Email tidak ditemukan atau password salah sama-sama menjadi 401. View menampilkan pesan error dan mengakhiri loading. Fitur Demo Accounts hanya mengisi input, bukan melewati proses login/API.

### 30.3 GET /api/me

Sumber: [`authService.me()`](../frontend/src/services/auth.service.js#L10-L11), [`auth.fetchMe()`](../frontend/src/stores/auth.js#L37-L47), [`AuthHandler.Me()`](../backend/internal/handler/auth_handler.go#L50-L62).

Endpoint ini tidak memanggil `AuthService` ataupun `UserRepository`. Jalurnya:

```text
JWT → middleware.Auth() → currentUser → AuthHandler.Me() → UserResponse
```

Response berisi `id`, `name`, `email`, dan `role`. `created_at` pada `UserResponse` dibiarkan kosong dan dihilangkan oleh `omitempty`, berbeda dari response login/register yang mengisinya.

Jika `fetchMe()` dipanggil, ia memperbarui user pada store/storage; kegagalan apa pun membuat store logout. **Namun tidak ada pemanggilan `fetchMe()` pada bootstrap, guard, layout, atau view saat ini.** Tersedianya method bukan berarti request profil otomatis terjadi saat reload.

Bahkan pemanggilan endpoint ini dengan JWT lama tidak memperbarui role dari database, karena sumber datanya tetap claim JWT.

## 31. Bedah API Tiket dan Integrasi Frontend

### 31.1 GET /api/tickets

**Rantai kode:** [`TicketsView.fetchTickets()`](../frontend/src/views/TicketsView.vue#L351-L373) → [`ticketService.list()`](../frontend/src/services/ticket.service.js#L4-L5) → [`TicketHandler.List()`](../backend/internal/handler/ticket_handler.go#L31-L63) → [`TicketService.List()`](../backend/internal/service/ticket_service.go#L38-L75) → [`TicketRepository.List()`](../backend/internal/repository/ticket_repo.go#L94-L232).

**Query yang benar-benar dibentuk UI:**

```js
const params = {
  page: meta.value.current_page,
  limit: meta.value.limit,
  search: filters.search || undefined,
  status: filters.status || undefined,
  priority: filters.priority || undefined
}
if (!isRequester.value && filters.assignee_id) {
  params.assignee_id = filters.assignee_id
}
```

Nilai `undefined` tidak dimaksudkan sebagai teks literal pada URL. Axios membentuk query dari nilai yang tersedia.

| Sumber input | Nilai yang dikirim | Transformasi server | Kondisi SQL |
|---|---|---|---|
| Search | Teks `filters.search` | Lowercase dan dibungkus `%` | `LOWER(t.title) LIKE $n`. |
| Filter status | `open`, dll. | Diteruskan sebagai nilai parameter | `t.status = $n`. |
| Filter prioritas kosong | Tidak dikirim | Tidak menambah filter | Semua prioritas. |
| Pilihan prioritas unset | `unset` | Sentinel khusus | Priority `IS NULL` atau kosong. |
| Assignee saya | `me` | Handler mengganti dengan ID JWT | `t.assignee_id = $n`. |
| Assignee belum ditentukan | `unassigned` | Sentinel khusus | `t.assignee_id IS NULL`. |
| Assignee user tertentu | UUID user | Diteruskan | `t.assignee_id = $n`. |
| Kepemilikan requester | Tidak berasal dari form/query UI | Service memaksa ID aktor | `t.requester_id = $n`. |

Handler memakai `Atoi()` untuk page/limit dan mengabaikan error parsing. Nilai nol/nonpositif kemudian dinormalisasi menjadi page 1 dan limit 10. Belum ada batas maksimum limit.

Repository melakukan count lalu query daftar. Query daftar menggabungkan `tickets` dengan dua alias `users`: `u1` untuk requester dan `u2` untuk assignee. Tidak ada query detail per baris untuk mengambil nama mereka.

**Response → state:**

```js
const res = await ticketService.list(params)
tickets.value = res.data || []
meta.value = res.meta || { total_items: 0, total_pages: 0, current_page: 1, limit: 10 }
```

Pemetaan tampilan:

- `tickets[]` → row tabel desktop atau kartu mobile.
- `requester.name/email` → kolom pelapor staf.
- `assignee.name` → nama penanggung jawab atau label unassigned.
- `status` → prop `StatusBadge`.
- `priority` → prop `PriorityBadge`.
- `is_overdue` → badge SLA.
- `description` → ringkasan expandable dari response list, tanpa GET detail tambahan.
- `meta` → rentang hasil dan tombol pagination.

**Pemicu request:** mount; retry; pencarian debounce 300 ms; perubahan filter; reset; pergantian halaman; event `ticket_created` bagi non-requester; event `ticket_updated`.

**API tersedia tetapi tidak dipakai form saat ini:** handler mendukung `sort_by` dan `order`, sedangkan `fetchTickets()` tidak mengirim keduanya. Akibatnya daftar memakai default `created_at desc`. Sort prioritas di repository berupa urutan string, bukan bobot urgensi.

**Error:** skeleton digantikan `ErrorAlert` dengan retry `fetchTickets()`. Gagal memuat statistik/pilihan assignee tidak disatukan dengan error daftar.

### 31.2 POST /api/tickets

Sumber frontend: [`CreateTicketView.handleSubmit()`](../frontend/src/views/CreateTicketView.vue#L188-L225). Backend: [`TicketHandler.Create()`](../backend/internal/handler/ticket_handler.go#L80-L99), [`TicketService.Create()`](../backend/internal/service/ticket_service.go#L109-L154), [`TicketRepository.Create()`](../backend/internal/repository/ticket_repo.go#L21-L39).

Payload aktual frontend:

```js
const payload = {
  title: form.title,
  category: form.category,
  description: form.description
}
if (auth.isAdmin && form.requester_id) {
  payload.requester_id = form.requester_id
}
```

| Field | Siapa menentukan | Efek penyimpanan |
|---|---|---|
| `title`, `description`, `category` | Form | Disimpan ke kolom tiket; required pada DTO HTTP. |
| `requester_id` | Aktor secara default; dapat dipilih admin | Foreign key ke `users`. |
| `status` | Service | Selalu `open` untuk create. |
| `priority` | Opsional API admin; tidak ada field pada form | Requester selalu dikosongkan; SQL menyimpan kosong sebagai NULL. |
| `assignee_id` | Tidak diisi create service | NULL. |
| `id` dan timestamp | Service | UUID dan waktu UTC. |

Agent ditolak service dengan 403. Jika admin memilih pelapor, service memuat target user untuk memeriksa keberadaan dan mengisi relasi requester pada response. Backend tidak membatasi role user target tersebut.

Handler binding masih memiliki pesan lama yang menyebut prioritas wajib. Tag `CreateTicketReq` tidak mewajibkan prioritas; pesan tersebut bukan kontrak field yang benar.

**Response 201 dan penggunaan frontend:**

```js
const res = await ticketService.create(payload)
const ticketId = res.data.id
```

Hanya ID response yang diperlukan view untuk:

1. Menjadi target upload opsional.
2. Membentuk URL `/tickets/${ticketId}`.
3. Memicu lifecycle detail yang mengambil GET detail baru.

Create mengembalikan objek yang baru dibentuk, bukan hasil `Get()`. `comments`/`attachments` dapat masih `null`, `assignee` tidak dimuat, dan indikator overdue belum dihitung ulang pada jalur create ini.

**Relasi event:** setelah INSERT sukses, service menyiarkan `ticket_created` dengan `id` dan `title` kepada staf. Browser pemilik yang baru membuat tiket tidak memerlukan event itu untuk navigasi, karena sudah menerima ID dari response create.

**Relasi kegagalan:** upload gagal tidak membatalkan create. Alur create menangkap error upload secara terpisah, menulis ke console, dan tetap membuka detail.

### 31.3 GET /api/tickets/:id

Sumber: [`TicketDetailView.fetchTicket()`](../frontend/src/views/TicketDetailView.vue#L421-L432), [`TicketHandler.Get()`](../backend/internal/handler/ticket_handler.go#L65-L78), [`TicketService.Get()`](../backend/internal/service/ticket_service.go#L77-L107).

Service menggabungkan tiga pembacaan:

```text
TicketRepository.FindByID(id)
  → tickets + requester + assignee
CommentRepository.ListByTicket(id, includeInternal)
  → comments + author
AttachmentRepository.ListByTicket(id)
  → attachments + uploader
```

Cek ownership dilakukan setelah menemukan tiket dan sebelum memuat komentar/lampiran. Requester nonpemilik mendapat 403. Staf dapat membaca tiket terlepas dari siapa assignee-nya.

`includeInternal` hanya true bagi agent/admin. Repository komentar memasukkan filter `is_internal = FALSE` untuk requester. Lampiran tidak mempunyai filter internal karena model lampirannya memang tidak memiliki field itu.

State detail diperbarui dengan:

```js
const res = await ticketService.get(route.params.id)
ticket.value = res.data
```

Artinya view mengganti seluruh objek detail, bukan hanya status. Komentar/lampiran terbaru ikut masuk pada refetch.

**Pemicu refetch:** mount, retry, selesai submit komentar/file, selesai upload langsung, dan SSE terkait ID tiket. Loading detail diaktifkan setiap refetch; ini dapat membuat skeleton muncul kembali, bukan hanya mengubah satu badge.

**Sumber ID yang penting:** read memakai `route.params.id`, sedangkan update/komentar/upload memakai `ticket.value.id`. View belum mengawasi perubahan parameter route. Jika instance dipakai ulang untuk ID baru tanpa remount, object tiket lama dapat tertinggal dan target mutasi dapat berbeda dari URL. Ini potensi berdasarkan alur kode, bukan skenario yang sudah direproduksi pada browser dalam lampiran ini.

### 31.4 PATCH /api/tickets/:id

Sumber: [`TicketHandler.Update()`](../backend/internal/handler/ticket_handler.go#L101-L120), [`TicketService.Update()`](../backend/internal/service/ticket_service.go#L156-L235), [`TicketRepository.Update()`](../backend/internal/repository/ticket_repo.go#L234-L263).

**Empat pemanggil UI ke satu endpoint:**

| Fungsi pada detail | Pemicu | Payload aktual | Kewenangan |
|---|---|---|---|
| `updateStatus(status)` | Tombol aksi → dialog → `confirmAction()` | `{ status }` | Matriks transisi. |
| `updatePriority(priority)` | Select prioritas | `{ priority }` | Agent/admin. |
| `updateAssignee(assigneeId)` | Select assignee | `{ assignee_id: assigneeId || null }` | Admin. |
| `updateRequester(requesterId)` | Select pelapor | `{ requester_id: requesterId }` | Admin. |

Rujukan frontend: [`TicketDetailView.vue:447–509`](../frontend/src/views/TicketDetailView.vue#L447-L509).

**Semantik pointer DTO:**

| JSON | Hasil binding | Konsekuensi |
|---|---|---|
| Field tidak ada | Pointer nil | Field tidak diubah. |
| `"assignee_id": null` | Pointer nil | Tidak melepas assignee. |
| `"assignee_id": ""` | Pointer ke string kosong | Admin melepas assignee. |
| `"assignee_id": "unassigned"` | Pointer ke sentinel | Admin melepas assignee. |
| `"priority": ""` | Pointer ke string kosong | Staf dapat mengosongkan prioritas via API; UI tidak menawarkan opsi clear. |
| `"requester_id": ""` | Pointer non-nil | Admin mendapat not found; target pelapor tidak boleh kosong. |
| `"status": ""` | Pointer non-nil | Tidak cocok transisi dari status normal, sehingga 422. |

**Urutan bisnis:** muat tiket → cek role/ownership dan field → validasi status → auto-assign agent → terapkan prioritas → validasi user assignee/pelapor bila diubah → UPDATE → event → GET detail ulang.

Satu request API dapat berisi beberapa field walaupun UI mengirim satu per aksi. Judul/deskripsi/kategori tidak tersedia pada DTO PATCH. Repository memang menulis ulang kolom-kolom itu dari entity yang telah dibaca, bukan dari input edit konten baru.

Cuplikan inti auto-assignment:

```go
if *req.Status == domain.StatusInProgress && actor.Role == domain.RoleAgent {
    ticket.AssigneeID = &actor.ID
    ticket.Assignee = actor
}
```

Cuplikan ini berjalan hanya di dalam cabang status berubah dan telah lolos `ValidateTransition()`. Admin yang memulai pengerjaan tidak otomatis ditugaskan.

**Response → tampilan:** keempat fungsi frontend menjalankan `ticket.value = res.data`, lalu menampilkan toast sukses. Response detail memuat komentar/lampiran sesuai role aktor. Karena itu server sengaja memanggil `Get()` setelah UPDATE, bukan hanya mengembalikan field yang diubah.

**Event:** `ticket_updated` hanya membawa `id`. Layout menampilkan toast; list mengambil ulang data/statistik; detail yang cocok mengambil ulang data. Pemanggil PATCH dapat menerima response sekaligus refetch SSE, sehingga ada lebih dari satu pembaruan state untuk aksi yang sama.

**Failure dan edge case:**

- 403 untuk field yang tidak boleh diubah aktor.
- 422 untuk transisi ilegal, bukan semua kegagalan role.
- 404 jika user target/tiket tidak ada; format UUID invalid dapat menjadi error SQL umum karena belum divalidasi khusus di handler.
- JSON kosong `{}` tetap dapat memperbarui `updated_at` dan memublikasikan event pada implementasi ini.
- Kegagalan GET detail setelah UPDATE tidak otomatis membatalkan perubahan yang sudah tersimpan. Response error pada akhir alur bukan bukti bahwa UPDATE belum terjadi.
- Tidak ada busy lock untuk empat fungsi PATCH. Select gagal tidak memiliki rollback DOM eksplisit; data object lama tetap ada.
- `updateStatus()` menangkap error sendiri, sehingga `confirmAction()` tetap menutup dialog setelah fungsi itu selesai, termasuk ketika PATCH gagal.

## 32. Bedah API Komentar dan Relasi Diskusi

### 32.1 POST /api/tickets/:id/comments

Sumber: [`TicketDetailView.submitComment()`](../frontend/src/views/TicketDetailView.vue#L541-L561), [`ticketService.addComment()`](../frontend/src/services/ticket.service.js#L16-L17), [`TicketHandler.AddComment()`](../backend/internal/handler/ticket_handler.go#L122-L141), [`TicketService.AddComment()`](../backend/internal/service/ticket_service.go#L237-L275), [`CommentRepository`](../backend/internal/repository/comment_repo.go#L19-L84).

**Relasi data:**

```text
Ticket.id ← Comment.ticket_id
User.id   ← Comment.author_id
Comment.is_internal → menentukan visibilitas saat GET detail
```

Form memiliki object reaktif:

```js
const newComment = reactive({ body: '', is_internal: false })
```

Submit hanya memanggil API komentar jika `newComment.body.trim()` tidak kosong. Body yang benar-benar dikirim tetap nilai asli `newComment.body`; pengecekan trim tidak berarti teks sudah ditrim sebelum dikirim.

Checkbox internal hanya tampil bagi staf. Backend tetap memeriksa requester tidak boleh membuat internal note. Binding `required` memeriksa string tidak kosong, bukan aturan konten whitespace yang sama dengan UI.

**Langkah server:**

1. Bind `CreateCommentReq`.
2. Muat tiket untuk memastikan ada dan aktor boleh mengakses.
3. Tolak requester nonpemilik atau requester dengan `is_internal=true`.
4. Bentuk UUID, author ID dari aktor JWT, dan waktu UTC.
5. INSERT row `comments`.
6. Broadcast `comment_added` dengan `ticket_id`.
7. Return HTTP 201 dengan objek komentar dan relasi author dari aktor.

**Response frontend:** tidak di-append langsung ke `ticket.comments`. Setelah komentar dan upload opsional seluruhnya selesai, view mengosongkan form lalu menjalankan GET detail lagi. Author pada hasil GET berasal dari join database; author pada response create awal berasal dari context JWT. Keduanya tidak harus memiliki seluruh field terisi identik.

**Hubungan dengan file:**

| Isi form | Request yang dilakukan |
|---|---|
| Teks tanpa file | POST komentar → GET detail. |
| File tanpa teks | POST attachment → GET detail; tidak ada row komentar baru. |
| Teks dan file | POST komentar → POST attachment → GET detail. |
| Teks kosong/whitespace tanpa file | Tidak submit. |

Tidak ada foreign key dari attachment ke comment. Tidak ada API komentar bertingkat, edit komentar, atau delete komentar.

### 32.2 Relasi internal note dengan event dan tampilan

- Backend memfilter **isi komentar** pada GET, bukan pada EventSource klien.
- Semua `comment_added`, termasuk internal, memakai audience staf + pemilik tiket.
- Payload SSE tidak memuat `body`, `author`, atau `is_internal`.
- Requester dapat menerima toast lalu refetch tanpa melihat komentar baru karena yang ditambahkan adalah internal note.
- `comment.is_internal` di template hanya mengubah warna/badge; komponen bukan lapisan keamanan.

### 32.3 Partial success yang berpengaruh pada retry

Jika komentar berhasil lalu upload gagal, catch menampilkan toast error tetapi form belum dibersihkan. Retry submit dapat menghasilkan row komentar kedua. Tidak ada idempotency key atau transaksi gabungan antara API komentar dan API lampiran.

Jika GET detail setelah submit gagal, mutasi komentar mungkin sudah sukses. `fetchTicket()` menangkap error sendiri, sehingga UI dapat menampilkan toast sukses bersamaan dengan error pemuatan detail. Perbedaan antara **gagal menulis** dan **gagal membaca ulang** penting saat menelusuri masalah.

## 33. Bedah API Lampiran dan Relasi Berkas

### 33.1 POST /api/tickets/:id/attachments

Wrapper bersama pada [`ticket.service.js`](../frontend/src/services/ticket.service.js#L19-L24):

```js
const formData = new FormData()
formData.append('file', file)
return api.post(`/tickets/${id}/attachments`, formData, {
  headers: { 'Content-Type': 'multipart/form-data' }
}).then((r) => r.data)
```

Ini bukan JSON berisi path file browser. `File` dari input diubah menjadi multipart yang membawa byte berkas. Browser/Axios menangani boundary multipart; service hanya menentukan field `file` dan jenis request.

**Tiga pemakai wrapper yang sama:**

| Jalur UI | Sumber ID tiket | Validasi frontend | Sesudah sukses |
|---|---|---|---|
| Create | ID response create | Ukuran dan ekstensi | Navigasi ke detail. |
| Form balasan | `ticket.value.id` | Ukuran dan ekstensi | Reset form dan GET detail. |
| Upload langsung | `ticket.value.id` | Ukuran saja | Toast dan GET detail. |

Sumber: [`CreateTicketView`](../frontend/src/views/CreateTicketView.vue#L156-L225), [`file komentar`](../frontend/src/views/TicketDetailView.vue#L511-L561), dan [`upload langsung`](../frontend/src/views/TicketDetailView.vue#L592-L615).

**Rantai backend:** [`TicketHandler.UploadAttachment()`](../backend/internal/handler/ticket_handler.go#L178-L255) → [`TicketService.AddAttachment()`](../backend/internal/service/ticket_service.go#L277-L306) → [`AttachmentRepository.Create()`](../backend/internal/repository/attachment_repo.go#L18-L25).

| Tahap | Data/aksi | Pemilik tanggung jawab |
|---|---|---|
| Parse | Field multipart `file` | Handler. |
| Batas ukuran | Maksimum 5 MiB berdasarkan header file | Handler. |
| Ekstensi | Lowercase extension dan whitelist | Handler. |
| MIME metadata | Diambil dari map ekstensi | Handler; bukan deteksi byte. |
| Nama storage | UUID prefix + `filepath.Base(nama)` | Handler. |
| Simpan byte | `SaveUploadedFile()` ke disk | Handler. |
| Akses tiket | Pemilik untuk requester; staf diperbolehkan | Service. |
| ID metadata/relasi | UUID baru, ticket ID, uploader ID, waktu | Service. |
| Simpan metadata | INSERT `attachments` | Repository. |
| Notifikasi | `attachment_added` | Service. |

File disimpan **sebelum** service melakukan pengecekan akses tiket. Jika service gagal, handler mencoba `os.Remove(storedPath)`. Ini adalah cleanup kompensasi, bukan transaksi atomik filesystem+database. Kegagalan penghapusan diabaikan oleh kode.

**Metadata dan relasinya:**

```text
attachments.ticket_id   → tickets.id
attachments.uploader_id → users.id
attachments.file_path   → lokasi byte pada disk server
```

Nama prefix file dan `attachments.id` tidak sama: UUID storage dibuat handler, sedangkan UUID metadata dibuat service.

Response 201 mengembalikan entity attachment. Ketiga pemakai frontend saat ini mengabaikan isi response upload; daftar file diperbarui dengan GET detail atau navigasi, bukan mengambil path storage response sebagai URL download.

**Kegagalan yang harus dibedakan:**

- 400: tidak ada field `file`, ukuran melampaui limit, atau ekstensi ditolak.
- 403/404 dari service: tidak memiliki akses atau target tidak ada; berkas dicoba dibersihkan.
- 500: gagal menyiapkan folder, menulis berkas, atau menyimpan metadata.
- 413 dari proxy: mungkin belum mencapai handler Gin. Nginx proyek tidak mengatur `client_max_body_size`, sehingga batas efektif dapat lebih rendah dari 5 MiB.

### 33.2 GET /api/attachments/:id/download

Frontend: [`downloadAttachment()`](../frontend/src/services/ticket.service.js#L26-L30), dipanggil tombol detail menggunakan `att.id`, **bukan ticket ID atau file path**.

```js
const token = localStorage.getItem('token')
const baseURL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api'
window.open(`${baseURL}/attachments/${attachmentId}/download?token=${token}`, '_blank')
```

Backend: [`TicketHandler.DownloadAttachment()`](../backend/internal/handler/ticket_handler.go#L257-L273) → [`TicketService.GetAttachment()`](../backend/internal/service/ticket_service.go#L308-L324).

Urutan verifikasi:

1. Middleware membaca token query dan memverifikasinya.
2. Repository mencari attachment menurut ID.
3. Service mencari tiket dari `att.TicketID`.
4. Kepemilikan requester diperiksa terhadap tiket tersebut.
5. Handler memasang `Content-Disposition: attachment` dan MIME metadata.
6. `c.File()` membaca path disk dari metadata dan mengirim berkas.

**Bentuk response bukan `{data: ...}`**, melainkan isi file. Tab baru/built-in download browser yang menanganinya. Tidak ada loading/catch download pada view dan tidak ada auto-logout Axios pada jalur ini.

SSE memakai `encodeURIComponent(token)`, sedangkan URL download saat ini tidak melakukannya. Keduanya menempatkan token pada URL; hindari membiarkan URL token masuk log atau saluran yang tidak aman.

### 33.3 Privasi tidak diwarisi dari komentar

`is_internal` hanya ada pada komentar. Upload tidak menerima flag itu, tabel attachment tidak menyimpannya, dan GET detail menampilkan semua attachment yang terkait tiket sesuai akses tiket. Menandai komentar sebagai internal **tidak membuat file yang dikirim bersama komentar menjadi internal**.

## 34. Bedah API Statistik dan Pemetaan Dashboard

### 34.1 GET /api/stats

Pemakai: [`TicketsView.fetchStats()`](../frontend/src/views/TicketsView.vue#L375-L382) → [`ticketService.stats()`](../frontend/src/services/ticket.service.js#L32-L33).

Backend: [`TicketHandler.Stats()`](../backend/internal/handler/ticket_handler.go#L143-L156) → [`TicketService.Stats()`](../backend/internal/service/ticket_service.go#L326-L332) → [`TicketRepository.Stats()`](../backend/internal/repository/ticket_repo.go#L265-L296).

Service memberikan `requesterID=actor.ID` untuk requester atau string kosong untuk staf. Repository menambahkan WHERE kepemilikan hanya bila ID terisi.

| Field response | Asal perhitungan SQL | Pemakai kartu daftar |
|---|---|---|
| `total_tickets` | COUNT seluruh tiket pada cakupan | Tidak ditampilkan sebagai kartu daftar saat ini. |
| `open_tickets` | Status open | Ya. |
| `in_progress_tickets` | Status in_progress | Ya. |
| `resolved_tickets` | Status resolved | Ya. |
| `closed_tickets` | Status closed | Tidak pada kartu daftar. |
| `urgent_tickets` | Priority urgent, bukan resolved/closed | Ya. |
| `overdue_tickets` | Tidak diisi query umum; zero value 0 | Tidak pada kartu daftar. |

Semua field ada pada `StatsResponse`, tetapi keberadaan field tidak membuktikan setiap endpoint mengisi perhitungannya. Khusus overdue lengkap dihitung pada endpoint admin.

**Penyimpanan frontend yang berbeda dari view lain:**

```js
stats.value = await ticketService.stats()
```

`stats` menyimpan **body envelope**, sehingga template menggunakan `stats.data.open_tickets`, bukan `stats.open_tickets`.

Requester tidak meminta endpoint ini dari `TicketsView`, walaupun API mengizinkan requester meminta statistik miliknya. Error statistik ditangkap dan membuat kartu hilang (`stats=null`) tanpa ErrorAlert daftar.

Statistik ini tidak mengikuti filter/search/page tabel; wrapper tidak mengirim parameter filter.

### 34.2 GET /api/tickets/stats

Ini alias server ke `TicketHandler.Stats()`, dengan middleware autentikasi yang sama. Tidak ada pemanggil khusus dalam frontend; `ticketService.stats()` selalu memakai `/stats`.

Alias ini **bukan** `/api/admin/stats`. Mengganti panggilan umum menjadi statistik admin akan mengubah kewenangan serta cakupan data yang tersedia bagi role lain.

### 34.3 GET /api/admin/stats

Pemakai: [`AdminStatsView.fetchStats()`](../frontend/src/views/AdminStatsView.vue#L187-L200) → [`ticketService.adminStats()`](../frontend/src/services/ticket.service.js#L35-L36).

Backend memeriksa admin pada handler dan service, lalu menjalankan [`TicketRepository.AdminStats()`](../backend/internal/repository/ticket_repo.go#L298-L362).

Lima pembacaan utama:

1. Ringkasan global lewat `Stats(ctx, "")`.
2. Overdue urgent-open berdasarkan waktu PostgreSQL dan SLA konfigurasi.
3. GROUP BY kategori dari seluruh tiket.
4. GROUP BY prioritas terisi dari tiket bukan closed.
5. LEFT JOIN user agent dengan tiket bukan resolved/closed untuk beban kerja.

Contoh response ilustratif:

```json
{
  "data": {
    "total_tickets": 10,
    "open_tickets": 4,
    "in_progress_tickets": 2,
    "resolved_tickets": 3,
    "closed_tickets": 1,
    "urgent_tickets": 2,
    "overdue_tickets": 1,
    "categories": [
      { "category": "Network", "count": 6 },
      { "category": "IT Support", "count": 4 }
    ],
    "priorities": [
      { "priority": "high", "count": 4 },
      { "priority": "urgent", "count": 2 },
      { "priority": "medium", "count": 2 }
    ],
    "agents": [
      {
        "agent_id": "30000000-0000-4000-8000-000000000001",
        "agent_name": "Agent Contoh",
        "count": 2
      }
    ]
  }
}
```

`AdminDetailedStats` melakukan embedding `StatsResponse` pada Go. Field ringkasan karena itu berada pada level JSON yang sama dengan `categories`, `priorities`, dan `agents`, bukan di bawah objek `stats_response` tambahan.

**Pemetaan frontend:**

```js
const res = await ticketService.adminStats()
stats.value = res.data
```

Di sini `stats` menyimpan **isi entity**, berbeda dari `TicketsView` yang menyimpan envelope.

| Data | Perhitungan frontend | Hasil tampilan |
|---|---|---|
| Enam field total/status/overdue | Langsung | Kartu KPI. |
| `categories[].count` | `round(count / total_tickets × 100)` | Jumlah dan lebar bar kategori. |
| `priorities[].count` | Dibagi jumlah semua count prioritas, bukan total tiket | Lebar bar prioritas. |
| `agents[].count` | `>=5` tinggi; `>=2` normal; lainnya ringan | Label kapasitas visual. |
| `agents[].agent_name` | Huruf pertama untuk avatar | Identitas agent. |

Frontend tidak memfilter populasi agregat lagi. Kata “aktif” pada label prioritas/workload harus dibaca mengikuti query yang berbeda: distribusi prioritas masih menyertakan resolved, workload tidak.

Mount, tombol refresh, dan retry memanggil `fetchStats()`. SSE layout hanya menampilkan toast; halaman admin ini tidak otomatis mengambil ulang statistik ketika event diterima.

## 35. Bedah API Pengguna dan Relasi Penugasan

### 35.1 GET /api/users

Wrapper: [`userService.list(role)`](../frontend/src/services/user.service.js#L4-L5). Handler: [`UserHandler.List()`](../backend/internal/handler/user_handler.go#L20-L55). Repository: [`UserRepository.List()`](../backend/internal/repository/user_repo.go#L56-L80).

Handler menentukan cakupan:

- Requester: 403.
- Admin: satu pemanggilan service dengan query `role` opsional.
- Agent: dua pemanggilan service untuk `agent` dan `admin`, lalu menggabungkan hasil. Query role dari klien tidak dipakai untuk membebaskan agent membaca requester.

Repository memilih user tanpa kolom password hash untuk daftar, memfilter role jika diminta, dan mengurutkan nama. Untuk agent, hasil gabungan berupa daftar agent lalu daftar admin, bukan satu sort global gabungan.

**Seluruh pemakai UI:**

| Pemakai | Kapan request dilakukan | Transformasi hasil | Relasi yang memakai ID |
|---|---|---|---|
| [`CreateTicketView`](../frontend/src/views/CreateTicketView.vue#L139-L146) | Mount, hanya admin | Semua user → `requesters` | `form.requester_id` → create tiket. |
| [`TicketsView.fetchAssignees()`](../frontend/src/views/TicketsView.vue#L384-L392) | Mount, hanya admin | Filter role agent/admin → `assignees` | Filter query `assignee_id`. |
| [`TicketDetailView.fetchUsers()`](../frontend/src/views/TicketDetailView.vue#L434-L445) | Sesudah fetch detail, hanya admin | Filter staf untuk assignees; semua user untuk requesters | PATCH `assignee_id` / `requester_id`. |
| [`UsersView.fetchUsers()`](../frontend/src/views/UsersView.vue#L83-L94) | Mount, retry, pemulihan gagal update | Semua user → `users` | PATCH role user. |

Semua callsite saat ini memanggil `list()` tanpa argumen, meskipun wrapper mendukung parameter role. Nama variabel `requesters` tidak berarti isinya dibatasi ke user role requester.

Filter assignee terlihat pada agent, tetapi `fetchAssignees()` berhenti jika bukan admin. Dengan demikian agent memakai pilihan statis `me`/`unassigned`, bukan daftar nama dari endpoint tersebut pada halaman daftar saat ini.

**Response → option:** `u.id` menjadi `value`, `u.name`/`u.email` menjadi label. ID adalah foreign key yang dikirim pada request lanjutan; nama/email hanya presentasi dan tidak menentukan relasi SQL.

### 35.2 PATCH /api/users/:id/role

Pemakai: [`UsersView.changeRole()`](../frontend/src/views/UsersView.vue#L96-L109) → [`userService.updateRole()`](../frontend/src/services/user.service.js#L7-L8).

Backend: [`UserHandler.UpdateRole()`](../backend/internal/handler/user_handler.go#L57-L79) → [`UserService.UpdateRole()`](../backend/internal/service/user_service.go#L21-L42) → [`UserRepository.UpdateRole()`](../backend/internal/repository/user_repo.go#L88-L101).

Payload:

```json
{
  "role": "agent"
}
```

Langkah server:

1. Verifikasi aktor admin.
2. Bind role required, belum ada enum whitelist lengkap.
3. Cari user target.
4. Bila admin menurunkan role dirinya sendiri, hitung jumlah admin dan tolak jika tinggal satu.
5. UPDATE `users.role`.
6. Return 200 dengan `{data: {message: "Role pengguna berhasil diperbarui"}}`.

Frontend tidak memakai body response sebagai profil baru. Setelah await sukses, ia menjalankan `user.role = newRole` pada row daftar yang sedang dirender. `savingId` menonaktifkan dropdown row itu selama proses. Jika gagal, toast ditampilkan dan daftar user diminta ulang.

### 35.3 Relasi satu user dengan banyak fitur

```mermaid
flowchart TD
    U[users.id dan users.role] --> R[Pelapor pada tickets.requester_id]
    U --> A[Penanggung jawab pada tickets.assignee_id]
    U --> C[Penulis pada comments.author_id]
    U --> F[Pengunggah pada attachments.uploader_id]
    U --> W[Populasi agent pada statistik workload]
    U --> J[Claim saat login atau register]
    J --> G[Guard dan menu frontend]
    J --> M[Middleware dan audience SSE]
```

Mengubah role tidak mengubah ID atau seluruh foreign key tiket/komentar/lampiran. Contohnya, menurunkan agent menjadi requester tidak otomatis mengosongkan assignment lamanya. Namun query workload hanya memilih user yang saat pembacaan ber-role agent, sehingga populasi statistik dapat berubah.

Pada GET detail berikutnya, relasi user dari SQL dapat menampilkan role terbaru, sementara `auth.user`/JWT aktor masih memuat role lama. Ini dua sumber data berbeda, bukan bukti JWT otomatis diperbarui.

Tidak ada event `user_updated`/`role_changed` saat PATCH role. Menu, dropdown user pada view lain, dan sesi browser lain tidak otomatis disinkronkan oleh SSE.

## 36. Bedah API SSE dan Sinkronisasi Frontend

### 36.1 GET /api/events

Sumber: [`realtime.service.js`](../frontend/src/services/realtime.service.js#L7-L59), [`StreamEvents()`](../backend/internal/handler/ticket_handler.go#L275-L314), dan [`Broker`](../backend/internal/realtime/broker.go#L34-L85).

Koneksi dibuat saat consumer mendaftarkan event pertama:

```js
function onEvent(type, fn) {
  if (!_handlers.has(type)) _handlers.set(type, [])
  _handlers.get(type).push(fn)
  connect()
  return () => {
    const list = _handlers.get(type) || []
    const idx = list.indexOf(fn)
    if (idx > -1) list.splice(idx, 1)
  }
}
```

`_es` dan `_handlers` berada pada scope module. Tiga pemakai `useRealtime()` tidak berarti selalu ada tiga koneksi HTTP. Selama `_es` sudah terisi, `connect()` tidak membuat koneksi baru.

Urutan backend:

1. Token query diverifikasi middleware.
2. Header stream dikirim, kemudian komentar `: connected` dan flush.
3. Broker membuat subscriber berdasarkan role dan user ID.
4. Loop menunggu pesan, heartbeat 25 detik, atau request context ditutup.
5. Saat berakhir, fungsi unsubscribe membuang subscriber dan menutup channel.

Tidak ada query database pada stream ini. Audience ditentukan dari claim saat koneksi terbentuk, bukan dari pemeriksaan user ulang pada tiap notifikasi.

### 36.2 Dari mutasi SQL sampai refetch Vue

```mermaid
sequenceDiagram
    participant S as TicketService
    participant DB as Repository dan PostgreSQL
    participant B as DefaultBroker
    participant H as Handler stream
    participant E as EventSource singleton
    participant V as Callback layout atau view
    S->>DB: Simpan perubahan
    DB-->>S: Berhasil
    S->>B: BroadcastTo(audience, event, payload)
    B->>H: Kirim byte melalui channel subscriber
    H->>E: Tulis event SSE dan flush
    E->>E: Parse JSON dan dispatch berdasarkan jenis event
    E->>V: Panggil callback yang terdaftar
    V->>V: Toast dan/atau jalankan fetch data
    V->>DB: Pembacaan baru melalui HTTP API dan service
```

Panah terakhir mewakili jalur API lengkap, bukan akses database langsung dari browser.

### 36.3 Relasi publisher, audience, payload, dan callback

| Publisher | Event dan payload | Audience backend | Efek layout | Efek list/detail |
|---|---|---|---|---|
| `TicketService.Create` | `ticket_created`: `{id,title}` | Agent/admin | Toast judul bagi staf | List non-requester refetch list+stats; detail tidak subscribe event ini. |
| `TicketService.Update` | `ticket_updated`: `{id}` | Agent/admin atau user ID requester baru/saat ini | Toast | List refetch list+stats; detail refetch bila `p.id` cocok. |
| `TicketService.AddComment` | `comment_added`: `{ticket_id}` | Agent/admin atau pemilik tiket | Toast | Detail refetch bila `p.ticket_id` cocok; list tidak subscribe. |
| `TicketService.AddAttachment` | `attachment_added`: `{ticket_id}` | Agent/admin atau pemilik tiket | Toast | Detail refetch bila `p.ticket_id` cocok; list tidak subscribe. |

Audience menggunakan kecocokan role **atau** user ID, bukan keduanya harus cocok. Pemilik bukan staf tetap dapat menerima perubahan tiketnya tanpa menerima semua tiket staf.

Rujukan subscription:

- [`MainLayout`](../frontend/src/components/shared/MainLayout.vue#L388-L413).
- [`TicketsView`](../frontend/src/views/TicketsView.vue#L441-L466).
- [`TicketDetailView`](../frontend/src/views/TicketDetailView.vue#L621-L647).

### 36.4 Lifecycle dan batas jaminan

- Unmount view melepas callback yang dimilikinya.
- Koneksi bersama tidak otomatis ditutup hanya karena callback suatu view habis.
- Logout manual layout memanggil disconnect, menghapus sesi, lalu navigasi login.
- HTTP 401 pada Axios tidak melewati callback error SSE; keduanya transport berbeda.
- Handler `_es.onerror` kosong. Komentar “5s” bukan implementasi timer reconnect.
- Tidak ada event ID/replay. Reconnect tidak menjamin seluruh perubahan yang terlewat dibaca ulang otomatis.
- Channel buffer penuh membuat broker drop event.
- Subscriber lama tidak diputus secara khusus ketika token lewat waktu expiry atau role database berubah setelah koneksi terbuka.
- Dashboard admin tidak terdaftar sebagai consumer refetch statistik; toast tidak sama dengan data dashboard terbaru.

## 37. Peta Field Database ke JSON dan Komponen Vue

### 37.1 Lima relasi berbasis user

| Foreign key | Join saat membaca | Nama JSON | Pemakai frontend |
|---|---|---|---|
| `tickets.requester_id` | JOIN users `u1` | `ticket.requester` | Nama/email pelapor, select requester admin. |
| `tickets.assignee_id` | LEFT JOIN users `u2` | `ticket.assignee` | Nama/email penanggung jawab, select assignee. |
| `comments.author_id` | JOIN users | `comment.author` | Avatar awal huruf dan nama pada diskusi. |
| `attachments.uploader_id` | JOIN users | `attachment.uploader` | Nama pengunggah di daftar berkas. |
| User role sebagai kondisi agregasi | LEFT JOIN tickets pada user agent | `stats.agents[]` | Tabel workload; ini DTO agregat, bukan entity user lengkap. |

Foreign key menyimpan ID, bukan nama pengguna. Penggantian nama user pada database akan terlihat dari join berikutnya tanpa mengubah semua row tiket/komentar, walaupun aplikasi belum menyediakan endpoint edit profil tersebut.

### 37.2 Matriks hydration: field mana dimuat pada operasi mana

Hydration berarti mengisi object relasi dari hasil pembacaan/penyusunan data.

| Objek/relasi | List tiket | Create tiket | Get detail | Patch tiket | Create komentar | Upload |
|---|---|---|---|---|---|---|
| Field dasar tiket | Ya | Ya | Ya | Ya | Bukan response tiket | Bukan response tiket |
| Requester | SQL join | Aktor atau target user admin | SQL join | GET ulang | Tidak | Tidak |
| Assignee | SQL left join jika ada | Belum ada | SQL left join jika ada | GET ulang | Tidak | Tidak |
| Comments | Tidak dimuat | Tidak dimuat | Query terpisah | GET ulang | Hanya komentar baru | Tidak |
| Attachments | Tidak dimuat | Tidak dimuat | Query terpisah | GET ulang | Tidak | Hanya attachment baru |
| Author komentar | Tidak | Tidak | SQL join | GET ulang | Aktor JWT | Tidak |
| Uploader file | Tidak | Tidak | SQL join | GET ulang | Tidak | Aktor JWT |
| IsOverdue | Service menghitung | Zero value | Service menghitung | GET ulang | Tidak | Tidak |

Implikasi: response create/list tidak boleh diasumsikan memiliki bentuk terisi selengkap detail. API memakai entity yang sama, tetapi relasi opsional bergantung pada jalur pemuatan.

### 37.3 Relasi `nil`, `null`, `omitempty`, dan zero value

Sumber struktur JSON: [`entity.go`](../backend/internal/domain/entity.go#L33-L84).

- `assignee_id` berupa pointer tanpa `omitempty`: jika nil akan muncul `null`.
- `assignee` bertag `omitempty`: relasi nil dapat hilang dari JSON, bukan selalu `null`.
- `comments`/`attachments` tidak memakai `omitempty`: slice nil terserialisasi `null`.
- Repository komentar detail menginisialisasi array kosong, sehingga hasil kosongnya `[]`.
- Repository list tiket dan attachment dapat mengembalikan slice nil saat kosong.
- `priority` tetap string pada JSON; SQL NULL dibaca menjadi `""`.
- `is_overdue` selalu boolean, bukan timestamp deadline.
- `User.PasswordHash` tidak masuk JSON karena `json:"-"`.
- `Attachment.FilePath` tidak disembunyikan; jika terisi, `file_path` dapat masuk response.
- `User.CreatedAt` berupa `time.Time` tanpa `omitempty`. Jika user dibentuk dari JWT atau query tidak memuat waktu user, field dapat berisi waktu nol `0001-01-01T00:00:00Z`.
- `UserResponse.CreatedAt` berbeda: string dengan `omitempty`, sehingga `/me` dapat tidak menyertakan field itu.

Query attachment memuat uploader ID/nama/email/role, tetapi tidak `u.created_at`. Karena itu timestamp nol uploader pada detail merupakan keterbatasan hydration, bukan bukti akun dibuat pada tahun satu.

### 37.4 Contoh utuh bentuk detail tiket berelasi

Contoh berikut menggambarkan response **GET detail**, bukan response create. Semua ID, waktu, nama, dan lokasi file adalah ilustrasi. Tiket in-progress tidak overdue menurut aturan aktual, walaupun prioritas urgent. Uploader memakai timestamp nol sesuai field yang tidak diisi query attachment saat ini.

```json
{
  "data": {
    "id": "10000000-0000-4000-8000-000000000001",
    "title": "Koneksi VPN tidak tersedia",
    "description": "Koneksi terputus setelah memasukkan OTP.",
    "category": "Network",
    "priority": "urgent",
    "status": "in_progress",
    "requester_id": "20000000-0000-4000-8000-000000000001",
    "assignee_id": "30000000-0000-4000-8000-000000000001",
    "created_at": "2026-01-02T09:00:00Z",
    "updated_at": "2026-01-02T10:00:00Z",
    "requester": {
      "id": "20000000-0000-4000-8000-000000000001",
      "name": "Pelapor Contoh",
      "email": "requester@example.test",
      "role": "requester",
      "created_at": "2026-01-01T08:00:00Z"
    },
    "assignee": {
      "id": "30000000-0000-4000-8000-000000000001",
      "name": "Agent Contoh",
      "email": "agent@example.test",
      "role": "agent",
      "created_at": "2026-01-01T08:00:00Z"
    },
    "comments": [
      {
        "id": "40000000-0000-4000-8000-000000000001",
        "ticket_id": "10000000-0000-4000-8000-000000000001",
        "author_id": "30000000-0000-4000-8000-000000000001",
        "body": "Sedang memeriksa konfigurasi VPN.",
        "is_internal": false,
        "created_at": "2026-01-02T10:01:00Z",
        "author": {
          "id": "30000000-0000-4000-8000-000000000001",
          "name": "Agent Contoh",
          "email": "agent@example.test",
          "role": "agent",
          "created_at": "2026-01-01T08:00:00Z"
        }
      }
    ],
    "attachments": [
      {
        "id": "50000000-0000-4000-8000-000000000001",
        "ticket_id": "10000000-0000-4000-8000-000000000001",
        "uploader_id": "20000000-0000-4000-8000-000000000001",
        "filename": "laporan.png",
        "file_path": "uploads/60000000-0000-4000-8000-000000000001_laporan.png",
        "file_size": 2048,
        "content_type": "image/png",
        "created_at": "2026-01-02T10:02:00Z",
        "uploader": {
          "id": "20000000-0000-4000-8000-000000000001",
          "name": "Pelapor Contoh",
          "email": "requester@example.test",
          "role": "requester",
          "created_at": "0001-01-01T00:00:00Z"
        }
      }
    ],
    "is_overdue": false
  }
}
```

`updated_at` tiket pada contoh mendahului waktu komentar/lampiran karena penambahan keduanya tidak memperbarui row tiket pada service saat ini.

### 37.5 Dari response detail ke props komponen

```text
ticket.status
  → <StatusBadge :status="ticket.status" />
  → computed label/kelas badge

ticket.priority
  → <PriorityBadge :priority="ticket.priority" />
  → label prioritas atau belum ditentukan

ticket.comments[]
  → v-for comment
  → comment.author.name, body, created_at, is_internal

ticket.attachments[]
  → v-for att
  → filename, ukuran, uploader.name
  → tombol download(att.id)

pendingAction dari tombol status
  → props ConfirmDialog
  → emit confirm
  → confirmAction()
  → PATCH tiket
```

`StatusBadge`, `PriorityBadge`, `ConfirmDialog`, `ErrorAlert`, `LoadingSkeleton`, `EmptyState`, dan `Toast` tidak memanggil API sendiri. View menentukan data, kapan request dijalankan, dan cara menanggapi event komponen.

### 37.6 Data yang tidak dikirim atau belum dipakai UI

- Password/hash tidak ditampilkan atau diperlukan untuk daftar tiket/user.
- `file_path` dapat terkirim tetapi tidak dipakai UI sebagai URL unduh.
- `updated_at` ada pada entity, tetapi tampilan waktu utama daftar/detail memakai `created_at`.
- API statistik mengirim total/closed/overdue umum sebagai field struct, tetapi kartu daftar hanya memakai empat field tertentu.
- API filter sort tersedia, tetapi form list belum mengirim sort.
- API user mendukung filter role, tetapi seluruh callsite frontend meminta daftar tanpa argumen lalu menyaring lokal bila perlu.

## 38. Ketergantungan Request dan Propagasi Perubahan

### 38.1 Urutan pemuatan setiap halaman

| Halaman | Urutan request/lifecycle | Hubungan ketergantungan |
|---|---|---|
| Login/register | Submit → auth API → simpan sesi → navigasi | Landing bergantung pada sesi response. |
| Daftar tiket | `await fetchTickets()` → `Promise.all(fetchStats(), fetchAssignees())` → subscribe event | Stats/user tidak memakai ID response list, tetapi sengaja dimulai sesudah fetch pertama selesai. |
| Create tiket admin | Mount mengambil users; submit create → upload opsional → navigasi | Upload memerlukan ID create; daftar users hanya sumber pilihan pelapor. |
| Detail admin | `await fetchTicket()` → `await fetchUsers()` → subscribe event | Pilihan user untuk kontrol; detail berisi ID relasi aktif. |
| Detail non-admin | Fetch detail; helper users langsung return; subscribe | Tidak perlu endpoint users untuk membaca relasi user yang sudah embedded. |
| Users | Mount fetch users; PATCH row saat perubahan; refetch jika PATCH gagal | Respons list menjadi sumber dropdown/tabel. |
| Statistik admin | Mount fetch stats, tombol refresh/retry | Tidak bergantung pada daftar tiket dan tidak berlangganan refetch SSE. |

Helper fetch umumnya menangkap error sendiri. `await fetchTicket()` yang gagal secara HTTP tidak selalu melempar error ke callback mount, sehingga tahap sesudahnya tetap dapat berjalan.

### 38.2 Peta invalidasi data setelah mutasi

| Mutasi berhasil | Update state langsung pemanggil | Pembacaan lanjutan eksplisit | Efek event |
|---|---|---|---|
| Register/login | Store token/user | Navigasi membuat halaman tujuan meminta data | Tidak ada event auth. |
| Create tiket | Menyimpan ID lokal untuk navigasi/upload | GET detail dari halaman tujuan | List staf refetch. |
| PATCH tiket | Mengganti `ticket` dengan response detail | Tidak ada refetch eksplisit terpisah dari fungsi PATCH UI | SSE dapat memicu refetch tambahan. |
| Komentar | Tidak append response langsung | GET detail setelah seluruh submit selesai | Detail cocok dapat refetch sebelum/sesudah GET eksplisit. |
| Upload create | Tidak memakai response metadata | Navigasi detail | Toast staf/pemilik yang menerima event. |
| Upload dari detail | Tidak memakai response metadata | GET detail | Detail cocok dapat refetch lagi. |
| PATCH role | Mengubah row `user.role` lokal | Tidak fetchMe; list dimuat ulang hanya jika gagal | Tidak ada event role. |

Tidak ada cache global yang otomatis menginvalidasi seluruh data terkait. Setiap view menulis callback refetch sendiri. Sebuah perubahan dapat tercermin pada satu layar tetapi belum pada layar lain jika tidak ada pemicu refresh yang sesuai.

### 38.3 Perpindahan requester memindahkan kepemilikan

Misalnya admin mengubah pelapor dari A ke B:

1. UPDATE mengganti `tickets.requester_id` menjadi B.
2. GET/list selanjutnya untuk A tidak lagi memberikan tiket tersebut menurut pemeriksaan ownership.
3. B menjadi pemilik yang boleh membaca, mengomentari, dan mengakses file.
4. Komentar/lampiran lama tetap terkait ke tiket; penulis/pengunggah historis tidak berubah.
5. Event update dikirim kepada staf dan B, bukan otomatis A.
6. Browser A yang sedang membuka tiket bisa tetap menampilkan snapshot lama sampai request berikutnya; otorisasi server untuk akses berikutnya tetap mengikuti pemilik baru.

Tidak ada pemindahan file fisik karena file dihubungkan melalui ticket ID, bukan folder per requester.

### 38.4 Assignee tidak sama dengan ownership

- `requester_id` memengaruhi isolasi data pemohon.
- `assignee_id` menunjukkan siapa yang ditugaskan, tetapi tidak membatasi akses agent lain.
- Mengubah assignee mengubah tampilan penanggung jawab, filter assignee, dan workload jika target ber-role agent.
- Agent yang memulai tiket dapat mengganti assignment sebelumnya karena auto-assignment menetapkan ID aktor.
- Menghapus assignment tidak mengubah requester ataupun status secara otomatis.
- Backend belum memvalidasi target assignee harus staf; UI melakukan filter, tetapi API langsung tetap perlu ditinjau.

### 38.5 Perkiraan operasi SQL per endpoint

Berikut berasal dari penelusuran kode, **bukan hasil benchmark/profiling**. Jumlah adalah jalur normal satu request, tidak termasuk query tambahan browser akibat SSE, migrasi, atau seed.

| Operasi | Pembacaan SQL utama | Penulisan SQL utama |
|---|---|---|
| Register | Cari email | INSERT user. |
| Login | Cari email | Tidak ada. |
| Me / events | Tidak ada | Tidak ada. |
| List tiket | COUNT + SELECT berelasi | Tidak ada. |
| Create tiket | Tambahan cari user hanya jika admin memilih pelapor | INSERT tiket. |
| Get detail | SELECT tiket+user, SELECT komentar+author, SELECT attachment+uploader | Tidak ada. |
| PATCH tiket | SELECT awal; lookup target user bila perlu; tiga SELECT detail ulang | UPDATE tiket. |
| Tambah komentar | SELECT tiket | INSERT komentar. |
| Upload | SELECT tiket | INSERT metadata; byte file ditulis ke disk. |
| Download | SELECT attachment, SELECT tiket | Tidak ada; file dibaca dari disk. |
| Stats umum | Satu agregasi | Tidak ada. |
| Stats admin | Lima agregasi/pengelompokan | Tidak ada. |
| Daftar users admin | Satu SELECT | Tidak ada. |
| Daftar users agent | Dua SELECT role terpisah | Tidak ada. |
| PATCH role | SELECT target; COUNT admin untuk self-demotion | UPDATE user. |

Jumlah ini membantu menjelaskan mengapa “satu klik” dapat menimbulkan banyak aktivitas: PATCH sudah melakukan pembacaan detail ulang di server, kemudian event SSE dapat memicu GET lain dari browser.

### 38.6 Konkurensi dan urutan response

Contoh kemungkinan:

```text
Request GET A mulai
PATCH prioritas selesai → state menjadi versi baru
Event SSE memulai GET B
GET B selesai → state baru
GET A yang lambat selesai belakangan → dapat menimpa state
```

Tidak ada cancellation atau nomor versi response di view. Demikian pula, boolean loading tunggal tidak menghitung jumlah request aktif; satu request selesai dapat mematikan loading ketika request lain masih berjalan.

Pada backend, validasi transisi dilakukan sesudah SELECT dan sebelum UPDATE tanpa kondisi status lama/version pada WHERE. Validasi ini benar untuk state yang dibaca, tetapi belum mencegah konflik dua request yang membaca versi lama bersamaan.

### 38.7 Skenario gagal tidak selalu berarti tanpa efek

| Titik kegagalan | Efek yang mungkin sudah terjadi | Perilaku klien saat ini |
|---|---|---|
| Create sukses, upload gagal | Tiket sudah ada | Tetap navigasi; error upload hanya console. |
| Komentar sukses, upload gagal | Komentar sudah ada | Toast error, form masih berisi komentar/file. |
| PATCH sukses, GET detail server gagal | Row tiket sudah berubah dan event sudah dicoba dikirim | PATCH tampak error bagi pemanggil. |
| Mutasi sukses, refetch browser gagal | Data tersimpan | Tampilan dapat menunjukkan error load. |
| Metadata upload gagal setelah file ditulis | Berkas telah ditulis lalu dicoba dihapus | Response error; cleanup bukan transaksi atomik. |
| Event hilang tetapi response mutasi sukses | Database sudah berubah | Pemanggil mengetahui sukses, browser lain mungkin stale. |

Jangan melakukan retry mutasi secara membabi buta berdasarkan pesan error generik tanpa memeriksa apakah efek sebelumnya sudah tersimpan.

## 39. Penelusuran Masalah dan Verifikasi Kontrak API

### 39.1 Checklist menelusuri satu fitur

1. **Route halaman:** pastikan komponen yang tepat aktif dan guard mengizinkan role lokal.
2. **Pemicu:** cek submit/change/mount/callback SSE yang memanggil fungsi.
3. **ID sumber:** bedakan `route.params.id`, `ticket.value.id`, `user.id`, dan `attachment.id`.
4. **Wrapper:** periksa method HTTP, path relatif, query, JSON/multipart, dan base URL build.
5. **Autentikasi:** pastikan Bearer token atau query token sesuai transport.
6. **Handler:** periksa input path/query/body dan binding DTO.
7. **Service:** periksa ownership, role, status, dan urutan validasi/mutasi.
8. **Repository:** cocokkan nilai parameter, SQL, kolom SELECT, dan `Scan()`.
9. **Response:** pisahkan response entity, envelope JSON, response Axios, file, dan stream.
10. **State:** lihat apakah view menyimpan `res`, `res.data`, atau hanya ID.
11. **Template/props:** cek field yang benar-benar dirender dan fallback untuk null.
12. **Refetch/event:** periksa kemungkinan request susulan, stale state, atau partial success.

### 39.2 Contoh investigasi dropdown unassigned

```text
Option UI kosong
  → event.target.value adalah ""
  → updateAssignee("")
  → payload {assignee_id: null}
  → ShouldBindJSON ke *string menghasilkan nil
  → service tidak masuk cabang perubahan assignee
  → UPDATE menyimpan assignee lama
  → response sukses tetapi assignment tidak berubah
```

Akar masalah ada pada perbedaan kontrak nilai kosong, bukan permission admin, query filter `unassigned`, atau badge nama user. API langsung menerima `""`/`"unassigned"`; dokumen ini menjelaskan kondisi tersebut tanpa mengubah kode implementasinya.

### 39.3 Contoh investigasi statistik tidak sesuai filter

```text
Pengguna memilih filter priority=urgent
  → fetchTickets() mengirim query prioritas
  → daftar berubah
  → fetchStats() tidak dipanggil oleh perubahan filter
  → ketika dipanggil pun stats() tidak mengirim filter
  → angka ringkasan tetap untuk seluruh cakupan role
```

Ini berasal dari pemisahan kontrak endpoint, bukan otomatis kesalahan perhitungan count SQL.

### 39.4 Contoh investigasi role sudah diubah tetapi menu belum berubah

```text
PATCH role mengubah users.role di database
  → UsersView hanya mengubah row user.role lokal
  → auth.user tidak diganti
  → JWT lama tidak diterbitkan ulang
  → menu/guard membaca role auth lama
  → middleware membaca claim token lama
```

`/me` dengan token lama bukan solusi sinkronisasi database karena endpoint itu juga membaca claim. Perlu kebijakan sesi yang memang memperbarui sumber identitas; implementasi sekarang menerbitkan snapshot baru ketika login ulang.

### 39.5 Relasi dengan test yang tersedia

| Kelompok kontrak | Test yang berkaitan | Batas bukti |
|---|---|---|
| Register/login/me | `TestAPI_AuthEndpoints`, `TestAuthRegisterAndLogin` | Tidak membuktikan bootstrap browser memanggil me. |
| Tiket create/list/get/update | `TestAPI_TicketsEndpoints`, `TestTicketIsolation` | Repository mock, bukan query PostgreSQL nyata atau seluruh parameter filter. |
| State machine dan assignment agent | `TestValidStatusTransitions`, `TestInvalidStatusTransitions`, `TestAgentCannotCreateTicket`, `TestAgentCanManagePriority` | Bukan pengujian konflik update bersamaan di database. |
| Ganti pelapor | `TestAdminCanChangeRequester` | Tidak menguji seluruh perubahan state browser pemilik lama/baru. |
| Komentar internal | `TestAPI_CommentsEndpoint`, `TestInternalNoteSecurity` | Privasi komentar bukan privasi file yang ikut submit. |
| Upload/download | `TestAPI_AttachmentsUploadAndDownload`, `TestAttachmentUploadAndAccess` | Bukan bukti MIME sniffing, antivirus, atau batas body Nginx. |
| Statistik dan SLA | `TestAPI_StatsEndpoints`, `TestSLAOverdueCalculation` | Agregat mock tidak membuktikan seluruh semantik SQL admin. |
| Pengguna | `TestAPI_UsersEndpoints` | Tidak mencakup seluruh konkurensi self-demotion dan sesi lama. |
| Stream dan audience | `TestAPI_EventsEndpoint`, `TestBrokerAudienceScoping` | Bukan jaminan delivery/replay atau integrasi proxy/browser. |

Lampiran ini disusun melalui pembacaan kode dan penelusuran callsite. Tidak ada perubahan implementasi API/frontend, dan tidak ada klaim pengujian runtime baru. Validasi dokumentasi memeriksa konsistensi tautan, contoh JSON, dan kelengkapan pemetaan route; pengujian aplikasi tetap perlu dijalankan terpisah jika perilaku diubah.

### 39.6 Indeks sumber utama

**Jalur HTTP backend:**

- [Router dan pendaftaran seluruh endpoint](../backend/internal/handler/router.go).
- [Middleware JWT dan currentUser](../backend/internal/middleware/auth_middleware.go).
- [Handler autentikasi](../backend/internal/handler/auth_handler.go).
- [Handler tiket, komentar, upload/download, statistik, SSE](../backend/internal/handler/ticket_handler.go).
- [Handler user](../backend/internal/handler/user_handler.go).
- [Envelope response dan mapping error](../backend/internal/handler/response.go).

**Kontrak dan bisnis:**

- [Entity dan tag JSON](../backend/internal/domain/entity.go).
- [DTO request, filter, pagination, dan statistik](../backend/internal/domain/dto.go).
- [Interface service/repository](../backend/internal/domain/interfaces.go).
- [AuthService](../backend/internal/service/auth_service.go).
- [TicketService](../backend/internal/service/ticket_service.go).
- [UserService](../backend/internal/service/user_service.go).
- [State machine](../backend/internal/service/state_machine.go).
- [Broker audience](../backend/internal/realtime/broker.go).

**Penyimpanan:**

- [TicketRepository](../backend/internal/repository/ticket_repo.go).
- [UserRepository](../backend/internal/repository/user_repo.go).
- [CommentRepository](../backend/internal/repository/comment_repo.go).
- [AttachmentRepository](../backend/internal/repository/attachment_repo.go).

**Pemakai frontend:**

- [Axios dan interceptor](../frontend/src/services/api.js).
- [Auth service HTTP](../frontend/src/services/auth.service.js).
- [Ticket service HTTP dan download](../frontend/src/services/ticket.service.js).
- [User service HTTP](../frontend/src/services/user.service.js).
- [EventSource dan dispatch](../frontend/src/services/realtime.service.js).
- [Pinia auth](../frontend/src/stores/auth.js).
- [Route guard frontend](../frontend/src/router/index.js).
- [LoginView](../frontend/src/views/LoginView.vue).
- [RegisterView](../frontend/src/views/RegisterView.vue).
- [TicketsView](../frontend/src/views/TicketsView.vue).
- [CreateTicketView](../frontend/src/views/CreateTicketView.vue).
- [TicketDetailView](../frontend/src/views/TicketDetailView.vue).
- [UsersView](../frontend/src/views/UsersView.vue).
- [AdminStatsView](../frontend/src/views/AdminStatsView.vue).
- [MainLayout dan notifikasi global](../frontend/src/components/shared/MainLayout.vue).

Dengan peta ini, setiap field atau aksi UI dapat ditelusuri kembali sampai sumber data dan aturan aksesnya, tanpa menganggap kemiripan nama fungsi atau keberadaan wrapper sebagai bukti bahwa jalur itu selalu dijalankan.
