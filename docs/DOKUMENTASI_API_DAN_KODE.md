# Panduan API dan Kode Aplikasi Helpdesk

**Dokumentasi mandiri dengan bahasa sederhana untuk memahami backend Go, frontend Vue, dan hubungan data di antara keduanya.**

Dokumen ini menjelaskan kode yang sudah ada pada proyek. Tujuannya agar pembaca bisa menjawab:

- Tombol pada halaman ini memanggil API apa?
- Data apa yang dikirim browser dan apa yang dikembalikan server?
- Fungsi Go mana yang menjalankan aturannya?
- Tabel database mana yang dibaca atau diubah?
- Mengapa sebuah aksi diperbolehkan atau ditolak?
- Bagaimana hasil API menjadi tulisan, badge, daftar, dan notifikasi pada layar?

> **Cara membaca contoh:** potongan bertanda kode asli adalah cuplikan implementasi, bukan selalu satu file utuh yang bisa langsung dijalankan sendiri. Contoh JSON memakai data buatan. Jika disebut “field pilihan”, response asli memiliki field tambahan. Semua path berkas dihitung dari root repositori.
>
> **Batas cakupan:** penjelasan mengikuti implementasi saat ini, termasuk keterbatasannya. Dokumen tidak menyatakan semua fitur sudah siap produksi atau sudah diuji secara menyeluruh.

## Daftar Isi

1. [Konsep dasar yang perlu dipahami](#1-konsep-dasar-yang-perlu-dipahami)
2. [Gambaran aplikasi dan pembagian kode](#2-gambaran-aplikasi-dan-pembagian-kode)
3. [Cara frontend mengirim dan membaca data](#3-cara-frontend-mengirim-dan-membaca-data)
4. [API pendaftaran dan login](#4-api-pendaftaran-dan-login)
5. [API pengelolaan tiket](#5-api-pengelolaan-tiket)
6. [API komentar dan catatan internal](#6-api-komentar-dan-catatan-internal)
7. [API lampiran file](#7-api-lampiran-file)
8. [API statistik](#8-api-statistik)
9. [API pengguna dan role](#9-api-pengguna-dan-role)
10. [API notifikasi langsung atau SSE](#10-api-notifikasi-langsung-atau-sse)
11. [Hubungan database dan bentuk data](#11-hubungan-database-dan-bentuk-data)
12. [Peta halaman frontend ke API](#12-peta-halaman-frontend-ke-api)
13. [Contoh perjalanan satu tiket](#13-contoh-perjalanan-satu-tiket)
14. [Mencoba API secara langsung](#14-mencoba-api-secara-langsung)
15. [Error dan hal yang perlu diwaspadai](#15-error-dan-hal-yang-perlu-diwaspadai)
16. [Panduan membaca sumber dan pengujian](#16-panduan-membaca-sumber-dan-pengujian)

---

## 1. Konsep Dasar yang Perlu Dipahami

### 1.1 Apa itu API?

API adalah jalur komunikasi antara halaman yang dibuka pengguna dan program di server.

Contoh sederhana:

1. Pengguna menekan tombol **Buat Tiket**.
2. Browser mengirim judul dan deskripsi ke server.
3. Server memeriksa apakah pengguna boleh membuat tiket.
4. Server menyimpan data.
5. Server memberi tahu browser bahwa tiket berhasil dibuat.
6. Browser membuka halaman detail tiket tersebut.

Frontend tidak langsung mengubah tabel PostgreSQL. Semua perubahan data melewati backend API.

### 1.2 Istilah yang digunakan

| Istilah | Penjelasan sederhana |
|---|---|
| Frontend | Tampilan dan interaksi yang berjalan di browser, dibuat dengan Vue. |
| Backend | Program Go di server yang memeriksa aturan dan memproses data. |
| Endpoint | Alamat API tertentu, misalnya `/api/tickets`. |
| Method HTTP | Jenis tindakan: GET membaca, POST menambah, PATCH mengubah sebagian data. |
| Request | Permintaan dari browser/klien ke server. |
| Response | Jawaban server atas request. |
| Payload/body | Data utama yang dikirim, biasanya JSON atau file multipart. |
| JSON | Format teks terstruktur, misalnya `{"status":"open"}`. |
| Header | Informasi pendamping request, misalnya token Authorization. |
| Path parameter | ID yang menjadi bagian alamat, misalnya `:id` pada `/tickets/:id`. |
| Query parameter | Pilihan setelah tanda `?`, misalnya `?page=2&status=open`. |
| JWT/token | Bukti login bertanda tangan yang dikirim pada request berikutnya. |
| Role | Jenis kewenangan pengguna: requester, agent, atau admin. |
| ID/UUID | Identitas unik sebuah user, tiket, komentar, atau file. |
| State | Data yang sedang dipegang frontend untuk ditampilkan. |
| Relasi | Hubungan antardata, misalnya tiket dimiliki seorang pelapor. |
| SSE | Koneksi yang dipakai server untuk mengirim pemberitahuan ke browser. |

### 1.3 Membaca alamat API

Contoh:

```http
PATCH /api/tickets/10000000-0000-4000-8000-000000000001
Authorization: Bearer TOKEN_HASIL_LOGIN
Content-Type: application/json

{"priority":"urgent"}
```

Artinya:

- `PATCH`: ingin mengubah data.
- `/api/tickets/...`: data yang diubah adalah satu tiket dengan ID tersebut.
- `Authorization`: membawa bukti login.
- `Content-Type`: body berbentuk JSON.
- `priority`: field yang ingin diubah menjadi urgent.

Tulisan `:id` pada dokumentasi harus diganti dengan ID sebenarnya. ID tiket, ID user, dan ID attachment tidak boleh dianggap sama.

## 2. Gambaran Aplikasi dan Pembagian Kode

### 2.1 Peran pengguna

| Role | Contoh tugas |
|---|---|
| `requester` | Membuat laporan dan memantau tiket miliknya sendiri. |
| `agent` | Menangani laporan, menentukan prioritas, dan menyelesaikan tiket. |
| `admin` | Mengelola operasional, penugasan, pelapor, pengguna, dan statistik. |

Agent tidak boleh membuat tiket. Requester tidak boleh membaca tiket requester lain. Admin juga tetap mengikuti aturan perpindahan status, bukan bebas melompati semua status.

### 2.2 Alur dari tombol sampai database

```mermaid
flowchart TD
    U[Pengguna menekan tombol] --> V[View Vue menerima aksi]
    V --> F[Service frontend mengirim HTTP]
    F --> R[Router backend memilih endpoint]
    R --> M[Middleware memeriksa token]
    M --> H[Handler membaca input]
    H --> S[Service backend memeriksa aturan]
    S --> P[Repository menjalankan SQL]
    P --> D[(PostgreSQL)]
    D --> J[Response JSON kembali ke browser]
    J --> T[State Vue diperbarui]
    T --> L[Tampilan berubah]
```

Dua endpoint publik, register dan login, tidak memerlukan pemeriksaan token. Endpoint file download dan SSE memiliki bentuk response berbeda dari JSON biasa.

### 2.3 Apa tugas setiap folder?

| Lokasi | Tugas | Analogi |
|---|---|---|
| `frontend/src/views/` | Halaman, form, dan kapan request dijalankan | Meja tempat pengguna mengisi permintaan. |
| `frontend/src/services/` | Mengirim request HTTP | Pengantar permintaan ke server. |
| `frontend/src/stores/` | Menyimpan sesi dan pengaturan bersama | Catatan yang dipakai banyak halaman. |
| `backend/internal/handler/` | Membaca request dan membuat response | Petugas penerima permintaan. |
| `backend/internal/middleware/` | Memeriksa token sebelum handler | Pemeriksaan identitas. |
| `backend/internal/service/` | Menentukan apakah aksi sah | Pengambil keputusan berdasarkan aturan. |
| `backend/internal/repository/` | Membaca/menulis database | Petugas penyimpanan data. |
| `backend/internal/domain/` | Mendefinisikan bentuk data dan kontrak | Formulir dan definisi istilah bersama. |
| `backend/internal/realtime/` | Membagikan pemberitahuan SSE | Pengumuman kepada penerima yang sesuai. |

### 2.4 Tempat semua endpoint didaftarkan

Sumber: [`backend/internal/handler/router.go`](../backend/internal/handler/router.go).

Contoh kode asli:

```go
api := r.Group("/api")
```

Semua route di dalam grup tersebut mendapat awalan `/api`.

```go
tickets := api.Group("/tickets", middleware.Auth(cfg.JWTSecret))
```

Grup ini menambahkan `/tickets` dan pemeriksaan token. Setiap route di dalamnya berada di bawah `/api/tickets` dan memerlukan token yang valid.

Contoh pendaftaran detail yang sebenarnya:

```go
tickets.GET("/:id", cfg.TicketHandler.Get)
```

Jadi alamat akhirnya adalah `GET /api/tickets/:id`.

## 3. Cara Frontend Mengirim dan Membaca Data

### 3.1 Alamat server API

Sumber: [`frontend/src/services/api.js`](../frontend/src/services/api.js).

Kode asli:

```js
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api',
  headers: { 'Content-Type': 'application/json' }
})
```

Penjelasan:

1. `axios.create()` membuat klien HTTP yang dipakai bersama.
2. `baseURL` adalah awalan alamat request.
3. Jika `VITE_API_BASE_URL` tidak diisi, digunakan `http://localhost:8080/api`.
4. Request JSON memakai header `Content-Type: application/json`.

Jika service memanggil `api.get('/tickets')`, alamat akhirnya adalah `http://localhost:8080/api/tickets` pada fallback tersebut.

Dalam Docker, build frontend menggunakan base URL `/api`, lalu Nginx meneruskannya ke backend. Saat development, base URL `/api` dapat menggunakan proxy Vite ke backend. Nilai ini perlu benar sejak build karena JavaScript hasil build tidak otomatis berubah ketika environment Nginx diganti.

### 3.2 Token ditambahkan secara otomatis

Kode asli:

```js
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})
```

Ini berarti fungsi view tidak perlu menulis header token berulang kali. Sebelum request dikirim, interceptor membaca token tersimpan dan menambahkannya ke header.

Backend membaca token pada [`auth_middleware.go`](../backend/internal/middleware/auth_middleware.go), memverifikasinya, lalu menyediakan identitas pengguna melalui `middleware.CurrentUser(c)`.

### 3.3 Mengapa sering ada `res.data`?

Backend biasanya mengirim:

```json
{
  "data": {
    "status": "open"
  }
}
```

Ini hanya contoh field pilihan untuk menjelaskan pembungkus response.

Service frontend membuka pembungkus milik Axios:

```js
get(id) {
  return api.get(`/tickets/${id}`).then((r) => r.data)
}
```

Tetapi body API masih mempunyai field bernama `data`. Maka view membaca:

```js
const res = await ticketService.get(route.params.id)
ticket.value = res.data
```

Urutannya:

```text
r.data pada service = seluruh body JSON API
res pada view       = seluruh body JSON API
res.data pada view  = objek tiket di dalam body
```

Khusus daftar tiket, `data` berisi array dan `meta` berisi informasi halaman. Keduanya sejajar, bukan `data.meta`.

### 3.4 Arti `async`, `await`, dan `try/catch`

Contoh kode asli dari detail tiket:

```js
const fetchTicket = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await ticketService.get(route.params.id)
    ticket.value = res.data
  } catch (err) {
    error.value = err.message || t('load_ticket_failed')
  } finally {
    loading.value = false
  }
}
```

Penjelasan berurutan:

1. `loading = true`: tampilkan indikator menunggu.
2. Kosongkan error lama.
3. `await`: tunggu request selesai sebelum melanjutkan bagian ini.
4. Sukses: isi state `ticket` dengan data server.
5. Gagal: isi pesan error.
6. `finally`: matikan loading, baik sukses maupun gagal.

`async/await` tidak membuat seluruh browser berhenti; pengguna masih berada pada aplikasi sementara request diproses.

### 3.5 Penanganan error bersama

Interceptor response Axios:

- Menghapus sesi ketika response 401 diterima.
- Mengarahkan browser ke `/login` jika belum berada di sana.
- Mengambil pesan `error` dari body server.
- Mengubahnya menjadi `Error(message)` untuk ditangkap view.

Error API biasa ditangani Axios. Error download tab baru dan SSE tidak melewati interceptor yang sama.

## 4. API Pendaftaran dan Login

### 4.1 POST /api/auth/register

**Tujuan:** membuat akun requester dan langsung memberikan token login.

**Dipakai oleh:** `RegisterView.handleRegister()` → store `auth.register()` → `authService.register()`.

**Perlu token?** Tidak.

**Data request:**

```json
{
  "name": "Dewi Contoh",
  "email": "dewi@example.test",
  "password": "PasswordContoh123!"
}
```

| Field | Ketentuan |
|---|---|
| `name` | Wajib. |
| `email` | Wajib dan format email valid. |
| `password` | Wajib, minimal delapan karakter pada validasi HTTP. |

**Kode frontend asli:**

```js
register(payload) {
  return api.post('/auth/register', payload).then((r) => r.data)
}
```

**Proses backend:**

1. `AuthHandler.Register()` membaca JSON ke `RegisterReq`.
2. `AuthService.Register()` merapikan email dan mengubahnya menjadi huruf kecil.
3. Repository mencari apakah email sudah digunakan.
4. Password diubah menjadi hash bcrypt.
5. Akun baru disimpan dengan role `requester`.
6. Server membuat JWT.
7. Token dan profil dikembalikan dengan status 201.

Cuplikan kode asli:

```go
hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
```

Password tidak disimpan sebagai teks asli. Yang disimpan pada `users.password_hash` adalah hasil bcrypt.

**Contoh response sukses:**

```json
{
  "data": {
    "token": "TOKEN_JWT_CONTOH_BUKAN_TOKEN_ASLI",
    "user": {
      "id": "20000000-0000-4000-8000-000000000001",
      "name": "Dewi Contoh",
      "email": "dewi@example.test",
      "role": "requester",
      "created_at": "2026-01-01T08:00:00Z"
    }
  }
}
```

**Sesudah response diterima:** store menyimpan `token` dan `user`, lalu view membuka `/`. Router mengarahkan requester ke `/tickets`.

**Kemungkinan gagal:** 400 untuk input tidak sesuai, 409 jika email ditemukan sudah terdaftar, atau 500 untuk kegagalan internal. Konflik unik akibat request registrasi bersamaan belum dipetakan khusus di semua jalur.

**Relasi data:** akun yang dibuat akan dipakai sebagai pelapor tiket, penulis komentar, atau pengunggah file melalui ID-nya.

Sumber: [view](../frontend/src/views/RegisterView.vue), [auth store](../frontend/src/stores/auth.js), [handler](../backend/internal/handler/auth_handler.go), [service](../backend/internal/service/auth_service.go), [repository user](../backend/internal/repository/user_repo.go).

### 4.2 POST /api/auth/login

**Tujuan:** memeriksa email/password dan memberikan token.

**Dipakai oleh:** `LoginView.handleLogin()` → store `auth.login()` → `authService.login()`.

**Perlu token?** Tidak.

**Request:**

```json
{
  "email": "dewi@example.test",
  "password": "PasswordContoh123!"
}
```

**Kode frontend asli:**

```js
login(payload) {
  return api.post('/auth/login', payload).then((r) => r.data)
}
```

**Proses backend:**

1. Mencari user berdasarkan email ternormalisasi.
2. Membandingkan password dengan hash bcrypt.
3. Membuat token jika cocok.
4. Mengembalikan token dan profil dengan bentuk seperti response registrasi, tetapi status 200.

Cuplikan kode asli:

```go
if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
    return nil, domain.ErrUnauthorized
}
```

Jika email atau password salah, keduanya menghasilkan 401. Server tidak memberikan jawaban berbeda untuk membantu menebak email yang terdaftar.

**Apa isi token?** Token memuat ID, nama, email, role, waktu dibuat, dan waktu kedaluwarsa. Token ditandatangani dengan HS256, bukan dienkripsi. Isinya bukan tempat menyimpan password atau rahasia lain.

**Bagaimana frontend menyimpannya?**

Kode asli `setAuth()`:

```js
setAuth(authData) {
  this.token = authData.token
  this.user = authData.user
  localStorage.setItem('token', authData.token)
  localStorage.setItem('user', JSON.stringify(authData.user))
}
```

Landing admin adalah `/admin/stats`, sedangkan agent/requester `/tickets`, kecuali login memiliki tujuan `redirect` dari navigasi sebelumnya.

Tombol akun demo hanya mengisi email/password pada form, bukan melewati API login.

Sumber: [LoginView](../frontend/src/views/LoginView.vue), [auth service frontend](../frontend/src/services/auth.service.js), [AuthService backend](../backend/internal/service/auth_service.go).

### 4.3 GET /api/me

**Tujuan:** mengembalikan profil yang terkandung pada token.

**Perlu token?** Ya.

**Request body:** tidak ada.

**Kode frontend asli:**

```js
me() {
  return api.get('/me').then((r) => r.data)
}
```

**Contoh response:**

```json
{
  "data": {
    "id": "20000000-0000-4000-8000-000000000001",
    "name": "Dewi Contoh",
    "email": "dewi@example.test",
    "role": "requester"
  }
}
```

**Hal penting:** handler membaca `currentUser` dari hasil verifikasi JWT. Endpoint ini tidak mengambil user terbaru dari database.

Action `fetchMe()` ada di auth store, tetapi belum dipanggil otomatis pada startup atau guard. Jadi adanya fungsi `me()` tidak berarti browser selalu memanggil endpoint ini saat reload.

Jika admin mengubah role pengguna, `/me` dengan token lama tetap membaca role lama. Login ulang menerbitkan token dengan role yang dibaca dari database saat itu.

Sumber: [AuthHandler.Me](../backend/internal/handler/auth_handler.go), [middleware JWT](../backend/internal/middleware/auth_middleware.go), [auth store](../frontend/src/stores/auth.js).

## 5. API Pengelolaan Tiket

### 5.1 POST /api/tickets

**Tujuan:** membuat laporan baru.

**Dipakai oleh:** `CreateTicketView.handleSubmit()`.

**Boleh memakai:** requester dan admin. Agent ditolak dengan 403.

**Request sederhana:**

```json
{
  "title": "VPN kantor tidak dapat digunakan",
  "description": "Koneksi terputus sesudah memasukkan OTP.",
  "category": "Network"
}
```

| Field | Ketentuan |
|---|---|
| `title` | Wajib. |
| `description` | Wajib. |
| `category` | Wajib; UI menyediakan IT Support, Hardware, Software, Network, HR/Facility. |
| `priority` | Opsional pada API; kiriman requester diabaikan. Form create tidak mengisinya. |
| `requester_id` | Opsional dan digunakan hanya jika aktor admin memilih user lain. |

Kategori disimpan sebagai string. Opsi UI bukan berarti backend mempunyai tabel master atau enum kategori yang sama.

**Kode frontend asli saat menyusun payload:**

```js
const payload = {
  title: form.title,
  category: form.category,
  description: form.description
}
if (auth.isAdmin && form.requester_id) {
  payload.requester_id = form.requester_id
}
const res = await ticketService.create(payload)
const ticketId = res.data.id
```

`ticketId` digunakan untuk upload file opsional dan membuka halaman detail.

**Proses backend:**

1. Memeriksa aktor bukan agent.
2. Menentukan requester dari token, kecuali admin memilih pelapor lain.
3. Mengosongkan prioritas kiriman requester.
4. Membuat ID dan waktu UTC.
5. Mengatur status awal `open`.
6. Menyimpan tiket.
7. Mengirim event `ticket_created` kepada agent/admin.
8. Mengembalikan tiket dengan status HTTP 201.

Cuplikan aturan asli:

```go
if actor.Role == domain.RoleAgent {
    return nil, domain.ErrForbidden
}
priority := req.Priority
if actor.Role == domain.RoleRequester {
    priority = ""
}
```

**Contoh response, field pilihan:**

```json
{
  "data": {
    "id": "10000000-0000-4000-8000-000000000001",
    "title": "VPN kantor tidak dapat digunakan",
    "description": "Koneksi terputus sesudah memasukkan OTP.",
    "category": "Network",
    "priority": "",
    "status": "open",
    "requester_id": "20000000-0000-4000-8000-000000000001",
    "assignee_id": null
  }
}
```

Prioritas kosong pada Go/JSON disimpan sebagai NULL di PostgreSQL menggunakan `NULLIF(..., '')`.

**Catatan file:** create tiket tidak menerima file pada JSON ini. File diunggah dengan request terpisah setelah ID tiket didapat. Jika upload gagal, tiket yang sudah dibuat tidak otomatis dibatalkan.

Sumber: [CreateTicketView](../frontend/src/views/CreateTicketView.vue), [TicketHandler.Create](../backend/internal/handler/ticket_handler.go), [TicketService.Create](../backend/internal/service/ticket_service.go), [TicketRepository.Create](../backend/internal/repository/ticket_repo.go).

### 5.2 GET /api/tickets

**Tujuan:** membaca daftar tiket dan informasi halaman.

**Dipakai oleh:** `TicketsView.fetchTickets()`.

**Hak akses:** requester hanya mendapat tiket miliknya; agent/admin dapat membaca semua tiket.

**Contoh request:**

```http
GET /api/tickets?page=1&limit=10&status=open&priority=urgent
Authorization: Bearer TOKEN_HASIL_LOGIN
```

| Query | Arti | Default/catatan |
|---|---|---|
| `page` | Halaman yang diminta | 1. |
| `limit` | Jumlah tiket per halaman | 10; belum ada maksimum eksplisit. |
| `status` | Filter status | Kosong berarti semua. |
| `priority` | Filter prioritas | `unset` berarti belum ditentukan. |
| `assignee_id` | Filter penanggung jawab | UUID, `me`, atau `unassigned`. |
| `search` | Mencari judul | Bukan pencarian deskripsi/komentar. |
| `sort_by` | Kolom pengurutan | `created_at`, `updated_at`, `priority`. |
| `order` | Arah urutan | `asc` atau default `desc`. |

Form frontend saat ini mengirim filter dan pagination, tetapi tidak mengirim pilihan sort. Daftar karena itu menggunakan default sort server.

**Kode frontend asli:**

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

**Bagaimana isolasi requester dipastikan?** Service memaksa ID pelapor berdasarkan token:

```go
if actor.Role == domain.RoleRequester {
    filter.RequesterID = actor.ID
}
```

Jadi keamanan bukan sekadar menyembunyikan pilihan di frontend.

Handler menerjemahkan `assignee_id=me` menjadi ID aktor. Repository menerjemahkan `unassigned` menjadi kondisi SQL `assignee_id IS NULL`.

**Proses database:**

1. COUNT untuk menghitung seluruh hasil yang cocok.
2. SELECT tiket dengan filter, sort, limit, dan offset.
3. JOIN user pelapor dan LEFT JOIN user assignee agar nama/email tersedia.

**Contoh response, field tiket dipilih untuk ringkasnya:**

```json
{
  "data": [
    {
      "id": "10000000-0000-4000-8000-000000000001",
      "title": "VPN kantor tidak dapat digunakan",
      "status": "open",
      "priority": "urgent",
      "requester": {
        "name": "Dewi Contoh",
        "email": "dewi@example.test"
      },
      "is_overdue": false
    }
  ],
  "meta": {
    "total_items": 1,
    "total_pages": 1,
    "current_page": 1,
    "limit": 10
  }
}
```

`meta` dipakai untuk nomor halaman, rentang hasil, serta tombol sebelumnya/berikutnya.

**Kapan dimuat ulang?** Saat mount, retry, ganti filter, ganti halaman, pencarian setelah jeda 300 ms, dan event tiket tertentu. Klik membuka ringkasan baris memakai deskripsi dari response daftar; tidak langsung melakukan GET detail baru.

Sumber: [TicketsView](../frontend/src/views/TicketsView.vue), [TicketHandler.List](../backend/internal/handler/ticket_handler.go), [TicketService.List](../backend/internal/service/ticket_service.go), [query daftar](../backend/internal/repository/ticket_repo.go).

### 5.3 GET /api/tickets/:id

**Tujuan:** mendapatkan satu tiket beserta pelapor, assignee, komentar, dan lampiran.

**Dipakai oleh:** `TicketDetailView.fetchTicket()`.

**Request:**

```http
GET /api/tickets/10000000-0000-4000-8000-000000000001
Authorization: Bearer TOKEN_HASIL_LOGIN
```

**Kode frontend asli:**

```js
const res = await ticketService.get(route.params.id)
ticket.value = res.data
```

`route.params.id` berasal dari URL halaman Vue `/tickets/:id`.

**Proses backend:**

1. Mengambil row tiket beserta user pelapor/assignee.
2. Memeriksa requester adalah pemilik.
3. Mengambil komentar yang boleh dibaca aktor.
4. Mengambil lampiran.
5. Menghitung indikator overdue.
6. Mengembalikan detail.

Cuplikan kode asli untuk komentar:

```go
includeInternal := actor.Role == domain.RoleAgent || actor.Role == domain.RoleAdmin
comments, err := s.commentRepo.ListByTicket(ctx, id, includeInternal)
```

Untuk requester, repository hanya mengambil komentar `is_internal = FALSE`. Isi internal note tidak ikut payload detail requester.

**Di frontend:**

| Field | Tampilan |
|---|---|
| `title`, `description` | Judul dan uraian masalah. |
| `requester.name/email` | Identitas pelapor. |
| `assignee.name/email` | Identitas penanggung jawab. |
| `status` | Badge dan tombol aksi yang diperbolehkan. |
| `priority` | Badge atau dropdown prioritas staf. |
| `comments[]` | Daftar diskusi beserta lampiran komentar. |
| `attachments[]` | Lampiran tiket utama dengan tombol pratinjau dan unduh. |
| `is_overdue` | Tanda tiket terlambat. |

Contoh bentuk data lengkap dibahas pada bagian 11.

**Kapan detail dimuat ulang?** Setelah komentar/upload, saat retry, dan jika ada event SSE untuk ID tiket tersebut.

**Batasan saat ini:** view belum memantau perubahan `route.params.id` dengan watcher. Jika komponen dipakai ulang saat URL berubah ke tiket lain, data lama dapat tertinggal sampai ada pemuatan ulang yang sesuai.

### 5.4 PATCH /api/tickets/:id

**Tujuan:** mengubah status, prioritas, assignee, atau pelapor.

**Dipakai oleh:** empat fungsi pada `TicketDetailView`.

| Aksi UI | Fungsi | JSON yang dikirim |
|---|---|---|
| Tombol aksi status | `updateStatus()` | `{"status":"in_progress"}`. |
| Dropdown prioritas | `updatePriority()` | `{"priority":"urgent"}`. |
| Dropdown assignee admin | `updateAssignee()` | `{"assignee_id":"UUID_USER"}`. |
| Dropdown pelapor admin | `updateRequester()` | `{"requester_id":"UUID_USER"}`. |

Contoh berisi `UUID_USER` adalah penanda, bukan ID yang boleh dikirim apa adanya ke database.

**Aturan field:** requester hanya dapat melakukan transisi yang diizinkan pada tiketnya sendiri. Agent dapat mengubah status sesuai aturan dan prioritas. Admin juga dapat mengubah assignee/pelapor.

**Kode wrapper frontend asli:**

```js
update(id, payload) {
  return api.patch(`/tickets/${id}`, payload).then((r) => r.data)
}
```

**Contoh fungsi UI asli:**

```js
const updatePriority = async (priority) => {
  try {
    const res = await ticketService.update(ticket.value.id, { priority })
    ticket.value = res.data
    showToast(t('priority_updated'))
  } catch (err) {
    showToast(err.message || t('priority_update_failed'), 'error')
  }
}
```

Frontend mengganti seluruh detail dengan response terbaru, bukan hanya menulis prioritas ke layar secara optimistis.

#### Aturan perpindahan status

| Dari | Ke | Role yang diperbolehkan |
|---|---|---|
| `open` | `in_progress` | Agent/admin. |
| `open` | `closed` | Requester/admin. |
| `in_progress` | `resolved` | Agent/admin. |
| `resolved` | `closed` | Requester/agent/admin. |
| `resolved` | `open` | Requester saja. |

```mermaid
flowchart TD
    O[open] -->|Mulai ditangani| P[in_progress]
    P -->|Selesai ditangani| R[resolved]
    R -->|Pemilik membuka kembali| O
    O -->|Dibatalkan| C[closed]
    R -->|Ditutup| C
```

`closed` tidak memiliki transisi keluar. Mengirim status yang sama tidak dianggap transisi baru. Transisi ilegal ditolak dengan 422.

Sumber aturan backend: [`state_machine.go`](../backend/internal/service/state_machine.go). Frontend memiliki peta yang sama untuk menentukan tombol di [`constants/ticket.js`](../frontend/src/constants/ticket.js). Tombol UI bukan pengganti validasi server.

#### Auto-assignment agent

Saat agent mengubah open menjadi in-progress, service juga mengatur assignee:

```go
if *req.Status == domain.StatusInProgress && actor.Role == domain.RoleAgent {
    ticket.AssigneeID = &actor.ID
    ticket.Assignee = actor
}
```

Cuplikan ini berada di dalam cabang perubahan status yang sudah divalidasi. Jika admin yang memulai pengerjaan, admin tidak otomatis dijadikan assignee.

#### Field tidak dikirim berbeda dari string kosong

DTO update menggunakan pointer:

```go
type UpdateTicketReq struct {
    Status      *TicketStatus   `json:"status,omitempty"`
    Priority    *TicketPriority `json:"priority,omitempty"`
    AssigneeID  *string         `json:"assignee_id,omitempty"`
    RequesterID *string         `json:"requester_id,omitempty"`
}
```

Pada binding ini:

- Field tidak ada → nil → tidak diubah.
- Field bernilai JSON null → nil → juga tidak diubah.
- Field string kosong → pointer ke string kosong → dapat diproses sesuai aturan field.

Untuk melepas assignee melalui API sebagai admin:

```json
{
  "assignee_id": "unassigned"
}
```

String kosong juga diterima. **UI saat ini mengirim null ketika opsi assignee kosong dipilih, sehingga belum benar-benar melepas assignment.**

#### Hasil update

Service menyimpan perubahan, menerbitkan `ticket_updated`, lalu mengambil detail lagi. Response 200 berbentuk detail tiket terbaru. Karena frontend juga berlangganan SSE, aksi yang sama bisa diikuti GET detail tambahan.

PATCH tidak menyediakan edit judul, deskripsi, atau kategori. Tidak ada endpoint hapus tiket. Tiket `closed` mengunci diskusi dan lampiran komentar: komentar baru, edit/hapus komentar, serta unggah/ganti/hapus lampiran komentar ditolak. Lampiran tiket utama tetap dapat dikelola pemilik tiket.

Sumber: [DTO](../backend/internal/domain/dto.go), [TicketService.Update](../backend/internal/service/ticket_service.go), [TicketDetailView](../frontend/src/views/TicketDetailView.vue).

## 6. API Komentar dan Catatan Internal

### 6.1 POST /api/tickets/:id/comments

**Tujuan:** menambahkan informasi atau balasan pada tiket.

**Dipakai oleh:** `TicketDetailView.submitComment()`.

**Request komentar publik:**

```json
{
  "body": "Saya sudah restart laptop, tetapi VPN masih terputus.",
  "is_internal": false
}
```

**Request internal note oleh agent/admin:**

```json
{
  "body": "Sedang menelusuri konfigurasi gateway.",
  "is_internal": true
}
```

`body` wajib diisi. Requester tidak boleh membuat internal note dan hanya boleh berkomentar pada tiket miliknya.

**Proses backend:** memuat tiket, memeriksa akses, membuat ID komentar, mengambil author dari aktor JWT, menyimpan komentar, lalu menyiarkan `comment_added`.

**Relasi:** `comments.ticket_id` menunjuk tiket, sedangkan `comments.author_id` menunjuk user penulis.

**Contoh response 201, field pilihan:**

```json
{
  "data": {
    "id": "40000000-0000-4000-8000-000000000001",
    "ticket_id": "10000000-0000-4000-8000-000000000001",
    "author_id": "30000000-0000-4000-8000-000000000001",
    "body": "Sedang menelusuri konfigurasi gateway.",
    "is_internal": true,
    "created_at": "2026-01-02T10:00:00Z",
    "attachments": []
  }
}
```

Response asli juga dapat memuat object `author` dan `updated_at` bila komentar sudah diedit.

**Endpoint diskusi lainnya:**

| Method | Endpoint | Keterangan |
|---|---|---|
| PATCH | `/api/comments/:id` | Mengedit `body` komentar milik sendiri. |
| DELETE | `/api/comments/:id` | Menghapus komentar milik sendiri beserta berkas lampirannya. |
| POST | `/api/comments/:id/attachments` | Menambahkan berkas ke komentar milik sendiri. |

Edit/hapus komentar dan unggah lampiran komentar ditolak dengan HTTP 409 bila tiket sudah `closed`.

**Kode asli frontend yang menentukan urutan pengiriman:**

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

Komentar dibuat lebih dulu agar berkas dapat disimpan dengan `comment_id` yang benar. Setelah semuanya berhasil, frontend membersihkan form; daftar diperbarui lewat GET detail atau event SSE.

| Isi form | Request |
|---|---|
| Hanya teks | API komentar. |
| Hanya file | Form komentar mewajibkan teks; kolom lampiran tidak dapat dikirim sendiri. |
| Teks dan file | API komentar, lalu API lampiran komentar memakai ID komentar baru. |

**Dua hal penting:**

1. Jika komentar sukses tetapi upload gagal, komentar sudah tersimpan. Frontend menyimpan `pendingCommentId`, sehingga tombol **Selesaikan Komentar** mencoba unggah ulang tanpa menggandakan komentar.
2. File terhubung ke komentar melalui `comment_id` dan mewarisi visibilitas komentarnya. Berkas pada internal note tidak dikirim kepada requester.

Preview lokal sebelum pengiriman diproses di browser oleh `DraftFilePreview`; setelah terkirim, berkas dapat dilihat lewat endpoint pratinjau tanpa diunduh.

Event internal note hanya membawa `ticket_id`, bukan isi catatan. Pemilik tiket dapat menerima toast perubahan walaupun catatan internal tidak ikut muncul di detailnya.

Sumber: [handler](../backend/internal/handler/ticket_handler.go), [TicketService.AddComment](../backend/internal/service/ticket_service.go), [repository komentar](../backend/internal/repository/comment_repo.go), [form detail](../frontend/src/views/TicketDetailView.vue).

## 7. API Lampiran File

### 7.1 POST /api/tickets/:id/attachments

**Tujuan:** mengunggah satu file ke tiket.

**Dipakai oleh:** form create dan tombol tambah/ganti lampiran pada detail tiket. Lampiran komentar memakai endpoint `POST /api/comments/:id/attachments`.

**Jenis body:** `multipart/form-data`, bukan JSON.

**Nama field:** `file`.

**Kode frontend asli:**

```js
uploadAttachment(id, file) {
  const formData = new FormData()
  formData.append('file', file)
  return api.post(`/tickets/${id}/attachments`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' }
  }).then((r) => r.data)
}
```

`file` berasal dari input berkas browser. `FormData` membawa byte file, bukan sekadar path lokal komputer pengguna.

**Batas ukuran:** 5 MiB, yaitu 5.242.880 byte, pada handler backend.

**Ekstensi yang diizinkan:** JPG/JPEG, PNG, GIF, WEBP, PDF, TXT, CSV, DOC/DOCX, XLS/XLSX, ZIP.

**Proses server:**

1. Mengambil field `file`.
2. Memeriksa ukuran dan ekstensi.
3. Membuat nama penyimpanan dengan UUID prefix.
4. Menulis file ke direktori upload.
5. Memeriksa akses tiket melalui service.
6. Menyimpan metadata ke tabel `attachments`.
7. Menghapus file yang baru ditulis jika service gagal, sebatas upaya cleanup.
8. Mengirim event `attachment_added` dan response 201 jika berhasil.

Kode asli penamaan file:

```go
fileID := uuid.NewString()
sanitizedName := filepath.Base(fileHeader.Filename)
storedFilename := fmt.Sprintf("%s_%s", fileID, sanitizedName)
storedPath := filepath.Join(uploadDir, storedFilename)
```

Tujuannya membuang komponen folder pada nama asli dan mengurangi benturan nama file.

**Contoh response, field pilihan:**

```json
{
  "data": {
    "id": "50000000-0000-4000-8000-000000000001",
    "ticket_id": "10000000-0000-4000-8000-000000000001",
    "uploader_id": "20000000-0000-4000-8000-000000000001",
    "filename": "screenshot.png",
    "file_size": 2048,
    "content_type": "image/png"
  }
}
```

Byte file disimpan di disk. Database menyimpan identitas, nama, ukuran, jenis, path, dan relasinya. Response asli juga dapat memuat path, waktu, dan uploader.

**Validasi yang belum ada:** MIME metadata ditentukan dari ekstensi, bukan pemeriksaan isi file atau antivirus. Ekstensi aman tidak menjamin isi aman.

**Catatan Nginx:** konfigurasi proyek belum menetapkan `client_max_body_size`; default Nginx `1m` dapat menolak file sebelum batas 5 MiB backend tercapai. Error 413 dari proxy berbeda dari error 400 validasi file pada handler.

Sumber: [wrapper upload](../frontend/src/services/ticket.service.js), [handler upload](../backend/internal/handler/ticket_handler.go), [service lampiran](../backend/internal/service/ticket_service.go), [repository lampiran](../backend/internal/repository/attachment_repo.go).

### 7.2 GET /api/attachments/:id/download

**Tujuan:** mengunduh file yang sudah dilampirkan.

**ID pada alamat:** ID attachment, bukan ID tiket.

**Dipakai oleh:** tombol download pada `TicketDetailView`.

Kode frontend asli:

```js
const token = localStorage.getItem('token')
const baseURL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api'
window.open(`${baseURL}/attachments/${attachmentId}/download?token=${token}`, '_blank')
```

Browser membuka URL dengan query token karena pembukaan tab baru tidak memakai interceptor Axios.

**Proses backend:**

1. Memverifikasi token.
2. Mengambil metadata attachment.
3. Mengambil tiket dari `attachment.ticket_id`.
4. Memeriksa kepemilikan requester jika diperlukan.
5. Mengirim isi file dari path server.

Cuplikan kode asli:

```go
c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, url.PathEscape(att.Filename)))
c.Header("Content-Type", att.ContentType)
c.File(att.FilePath)
```

**Response bukan JSON**, tetapi isi berkas. Error pada tab baru tidak masuk ke catch Axios halaman asal.

Metadata database tidak cukup jika file fisiknya hilang. Backup harus mencakup database dan direktori/volume uploads.

### 7.3 POST /api/comments/:id/attachments

**Tujuan:** melampirkan berkas pada komentar yang sudah dibuat.

**Dipakai oleh:** `TicketDetailView.submitComment()` setelah `addComment()` mengembalikan ID komentar.

**Jenis body:** `multipart/form-data`, field `file`, batas dan ekstensi sama seperti lampiran tiket.

**Aturan akses:** hanya penulis komentar yang boleh mengunggah. Permintaan ditolak bila tiket sudah `closed`. Response 201 memuat metadata dengan `comment_id` yang terisi.

Berkas ini muncul pada komentar yang bersangkutan, bukan pada daftar **Lampiran Tiket**. Berkas pada catatan internal tidak dikirim dalam payload detail requester.

### 7.4 GET /api/attachments/:id/preview

**Tujuan:** melihat isi lampiran langsung di aplikasi tanpa mengunduhnya.

**Dipakai oleh:** tombol **Pratinjau** pada lampiran tiket, lampiran komentar, dan pratinjau draft sebelum pengiriman.

**Kode frontend asli:**

```js
previewAttachment(attachmentId, signal) {
  return api.get(`/attachments/${attachmentId}/preview`, { responseType: 'blob', signal }).then((r) => r.data)
}
```

Backend memakai header `Content-Disposition: inline`, `X-Content-Type-Options: nosniff`, `Cache-Control: private, no-store`, serta `Content-Security-Policy: sandbox` untuk berkas yang ditampilkan. Jenis konten dibatasi pada MIME aman; tipe lain jatuh ke `application/octet-stream` dan ditampilkan dengan pesan fallback di frontend.

Akses mengikuti aturan yang sama dengan unduhan: requester hanya dapat mempratinjau lampiran tiketnya sendiri dan komentar publik. File fisik harus tetap ada di direktori upload. Pratinjau tidak memakai layanan konversi pihak ketiga.

## 8. API Statistik

### 8.1 GET /api/stats

**Tujuan:** mengambil jumlah tiket berdasarkan cakupan pengguna.

**Dipakai oleh:** kartu ringkasan pada `TicketsView`, hanya diminta view untuk non-requester.

API sendiri tetap mengizinkan requester meminta ringkasan miliknya. Agent/admin mendapat ringkasan global.

**Body request:** tidak ada.

**Contoh response:**

```json
{
  "data": {
    "total_tickets": 10,
    "open_tickets": 4,
    "in_progress_tickets": 2,
    "resolved_tickets": 3,
    "closed_tickets": 1,
    "urgent_tickets": 2,
    "overdue_tickets": 0
  }
}
```

`urgent_tickets` menghitung urgent yang belum resolved/closed. `overdue_tickets` pada endpoint umum ini belum dihitung query, sehingga bernilai nol; jangan mengartikannya sebagai bukti tidak ada tiket overdue.

**Kode wrapper asli:**

```js
stats() {
  return api.get('/stats').then((r) => r.data)
}
```

**Cara view menyimpan:**

```js
stats.value = await ticketService.stats()
```

Karena seluruh body disimpan, template kartu daftar membaca `stats.data.open_tickets` dan field lain di bawah `stats.data`.

Ringkasan tidak mengikuti filter tabel karena request statistik tidak mengirim filter yang sama.

Sumber: [TicketsView](../frontend/src/views/TicketsView.vue), [service statistik](../backend/internal/service/ticket_service.go), [query statistik](../backend/internal/repository/ticket_repo.go).

### 8.2 GET /api/tickets/stats

**Tujuan:** alamat alternatif atau alias untuk `/api/stats`.

Keduanya menunjuk `TicketHandler.Stats()` dan memakai pemeriksaan token. Frontend saat ini menggunakan `/api/stats`, bukan alias ini.

Alias umum ini tidak sama dengan endpoint statistik admin.

### 8.3 GET /api/admin/stats

**Tujuan:** mengambil ringkasan operasional lengkap.

**Boleh memakai:** admin saja.

**Dipakai oleh:** `AdminStatsView.fetchStats()`.

**Isi response:** field statistik umum ditambah:

| Field | Isi |
|---|---|
| `categories` | Jumlah tiket per kategori, seluruh status. |
| `priorities` | Jumlah per prioritas yang terisi untuk tiket bukan closed; resolved masih termasuk. |
| `agents` | Jumlah penugasan per user ber-role agent untuk tiket bukan resolved/closed. |
| `overdue_tickets` | Jumlah urgent-open yang melewati SLA. |

**Contoh response:**

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

**Kode frontend asli:**

```js
const res = await ticketService.adminStats()
stats.value = res.data
```

Di halaman admin, state menyimpan isi `data`, bukan envelope. Template membaca `stats.open_tickets`, berbeda dari `stats.data.open_tickets` pada daftar tiket.

**Perhitungan grafik frontend:**

- Persentase kategori: jumlah kategori dibagi total tiket.
- Lebar bar prioritas: jumlah prioritas dibagi jumlah seluruh kelompok prioritas yang dikembalikan.
- Label workload: tinggi jika minimal 5, normal jika minimal 2, ringan jika kurang dari 2.

Ini bukan library analitik yang menghitung semua metrik otomatis. Resolution rate dan SLA breach rate khusus belum disediakan.

#### Aturan SLA sebenarnya

Tiket overdue jika:

```text
priority adalah urgent
DAN status adalah open
DAN umur sejak created_at lebih besar dari SLA_HOURS
```

Default `SLA_HOURS` adalah 2 jam. High/medium/low tidak mempunyai ambang SLA sendiri pada kode sekarang. Tiket urgent in-progress juga tidak memenuhi aturan overdue ini. Reopen tetap memakai waktu create awal.

Dashboard dimuat saat mount, refresh, atau retry. Toast SSE tidak otomatis memperbarui angka dashboard admin.

Sumber: [AdminStatsView](../frontend/src/views/AdminStatsView.vue), [DTO statistik](../backend/internal/domain/dto.go), [query admin](../backend/internal/repository/ticket_repo.go), [konfigurasi SLA](../backend/internal/config/config.go).

## 9. API Pengguna dan Role

### 9.1 GET /api/users

**Tujuan:** mengambil daftar pengguna untuk pilihan pelapor/assignee atau halaman manajemen.

**Aturan akses:**

| Aktor | Data yang boleh diterima |
|---|---|
| Requester | Tidak boleh; 403. |
| Agent | Hanya daftar user agent dan admin. |
| Admin | Semua user, dengan filter role opsional. |

**Contoh request admin:**

```http
GET /api/users?role=agent
Authorization: Bearer TOKEN_ADMIN
```

**Kode frontend asli:**

```js
list(role) {
  return api.get('/users', { params: role ? { role } : {} }).then((r) => r.data)
}
```

Seluruh callsite frontend saat ini memanggil `list()` tanpa argumen, lalu melakukan filter lokal jika perlu.

**Contoh response:**

```json
{
  "data": [
    {
      "id": "30000000-0000-4000-8000-000000000001",
      "name": "Agent Contoh",
      "email": "agent@example.test",
      "role": "agent",
      "created_at": "2026-01-01T08:00:00Z"
    }
  ]
}
```

Password/hash tidak disertakan.

**Pemakai frontend:**

| Halaman | Kegunaan response |
|---|---|
| Create admin | Mengisi pilihan pelapor tiket. |
| Daftar tiket admin | Mengisi pilihan filter assignee. |
| Detail admin | Mengisi pilihan assignee dan pelapor. |
| Users | Mengisi tabel pengguna dan pilihan role. |

Pilihan assignee disaring ke agent/admin di UI. Pilihan pelapor admin berisi semua user, bukan hanya role requester. Backend saat ini memeriksa user target ada, tetapi belum mewajibkan role assignee harus staf.

Sumber: [UserHandler.List](../backend/internal/handler/user_handler.go), [UserRepository.List](../backend/internal/repository/user_repo.go), [user service frontend](../frontend/src/services/user.service.js).

### 9.2 PATCH /api/users/:id/role

**Tujuan:** mengubah role pengguna.

**Boleh memakai:** admin.

**Dipakai oleh:** `UsersView.changeRole(user, newRole)`.

**Request:**

```json
{
  "role": "agent"
}
```

**Kode frontend asli:**

```js
updateRole(id, role) {
  return api.patch(`/users/${id}/role`, { role }).then((r) => r.data)
}
```

**Proses backend:**

1. Memastikan aktor admin.
2. Memuat user target.
3. Jika admin ingin menurunkan role dirinya sendiri, menghitung admin yang tersisa.
4. Menolak jika ia admin terakhir.
5. Menyimpan role baru.

**Contoh response:**

```json
{
  "data": {
    "message": "Role pengguna berhasil diperbarui"
  }
}
```

Frontend tidak mengganti sesi dari response itu. Ia hanya mengubah `user.role` pada row tabel. Jika gagal, pesan toast ditampilkan dan daftar user diminta ulang.

**Konsekuensi penting:** token lama dan `auth.user` tidak otomatis berubah. Pengguna perlu mendapatkan token baru melalui login ulang untuk membawa snapshot role terbaru. Proteksi admin terakhir juga belum memakai lock/transaksi untuk semua perubahan serentak.

Daftar nilai role pada UI bukan validasi enum lengkap pada backend; hal ini masih menjadi batasan implementasi.

Sumber: [UsersView](../frontend/src/views/UsersView.vue), [UserService.UpdateRole](../backend/internal/service/user_service.go), [query update role](../backend/internal/repository/user_repo.go).

## 10. API Notifikasi Langsung atau SSE

### 10.1 GET /api/events

**Tujuan:** mengirim pemberitahuan perubahan dari server ke browser tanpa polling berulang.

**Dipakai oleh:** `MainLayout`, `TicketsView`, dan `TicketDetailView` melalui `useRealtime()`.

**Jenis response:** `text/event-stream`, bukan JSON response sekali selesai.

Kode frontend asli:

```js
_es = new EventSource(`${BASE_URL}/events?token=${encodeURIComponent(token)}`)
```

Query token digunakan karena `EventSource` bawaan tidak menyediakan pengisian header Authorization khusus seperti request Axios.

### 10.2 Apa yang disebut event?

Contoh data yang dikirim server:

```text
event: ticket_updated
data: {"id":"10000000-0000-4000-8000-000000000001"}

```

Event hanya memberi tanda bahwa sesuatu berubah. Browser kemudian meminta data terbaru lewat API biasa.

| Event | Data yang dibawa | Penerima |
|---|---|---|
| `ticket_created` | `id`, `title` | Agent/admin. |
| `ticket_updated` | `id` | Agent/admin dan requester tiket saat ini. |
| `comment_added` | `ticket_id` | Agent/admin dan requester tiket. |
| `attachment_added` | `ticket_id` | Agent/admin dan requester tiket. |

“Agent/admin dan requester” di sini berarti penerima yang role-nya staf **atau** user ID-nya pemilik tiket, bukan harus memenuhi keduanya.

### 10.3 Kode penerima event pada frontend

Cuplikan asli dari detail tiket:

```js
realtime.onEvent('comment_added', (p) => {
  if (p?.ticket_id === route.params.id) fetchTicket()
})
```

Artinya: hanya muat ulang detail jika komentar baru milik tiket yang sedang dibuka.

| Pemakai | Reaksi |
|---|---|
| MainLayout | Menampilkan toast pemberitahuan. |
| TicketsView | Memuat ulang daftar/statistik untuk event tiket dibuat/diubah. |
| TicketDetailView | Memuat ulang detail jika ID cocok. |
| AdminStatsView | Tidak berlangganan refetch otomatis saat ini. |

### 10.4 Cara broker bekerja

Backend mempunyai broker di memori. Setiap koneksi terdaftar dengan role dan ID user. Service menyiarkan event sesudah perubahan berhasil disimpan.

Kode asli buffer subscriber:

```go
c := &client{ch: make(chan []byte, 32), role: role, userID: userID}
```

Masing-masing subscriber memiliki penampung 32 pesan. Jika buffer penuh, broker membuang event agar satu klien lambat tidak menahan semuanya.

Handler mengirim heartbeat setiap 25 detik untuk menjaga stream tetap aktif. Browser dapat melakukan reconnect bawaan, tetapi kode tidak membuat timer reconnect lima detik yang pasti.

### 10.5 Batasan SSE

- Tidak ada riwayat event atau replay.
- Restart menghilangkan broker dan subscriber dalam memori.
- Banyak instance backend belum berbagi broker bersama.
- Token diperiksa saat membuka stream, bukan diperiksa ulang pada setiap event.
- Callback view dilepas ketika unmount; logout manual layout menutup koneksi.
- Response mutasi dan event bisa sama-sama menyebabkan pembaruan state/refetch.

Sumber: [realtime frontend](../frontend/src/services/realtime.service.js), [handler SSE](../backend/internal/handler/ticket_handler.go), [broker](../backend/internal/realtime/broker.go), [MainLayout](../frontend/src/components/shared/MainLayout.vue).

## 11. Hubungan Database dan Bentuk Data

### 11.1 Hubungan utama

```mermaid
erDiagram
    users ||--o{ tickets : menjadi_pelapor
    users o|--o{ tickets : menjadi_assignee
    tickets ||--o{ comments : memiliki
    users ||--o{ comments : menulis
    tickets ||--o{ attachments : memiliki
    users ||--o{ attachments : mengunggah
```

Penjelasan:

- Satu user dapat menjadi pelapor banyak tiket.
- Satu tiket wajib punya satu pelapor.
- Tiket boleh belum memiliki assignee.
- Satu tiket dapat memiliki banyak komentar dan lampiran.
- Komentar dan attachment masing-masing menyimpan ID pembuatnya.
- Attachment tidak memiliki hubungan langsung ke komentar.

### 11.2 Mengapa ada ID dan object sekaligus?

Contoh:

```json
{
  "requester_id": "20000000-0000-4000-8000-000000000001",
  "requester": {
    "name": "Dewi Contoh",
    "email": "dewi@example.test"
  }
}
```

`requester_id` adalah identitas relasi yang disimpan. `requester` adalah data tambahan yang dibaca dari tabel user agar frontend bisa langsung menampilkan nama/email.

Dengan JOIN, frontend tidak perlu meminta profil user satu per satu untuk setiap baris tiket.

### 11.3 Daftar field entity

| Data | Field |
|---|---|
| User | `id`, `name`, `email`, `role`, `created_at`; hash password tidak masuk JSON. |
| Ticket | `id`, `title`, `description`, `category`, `priority`, `status`, `requester_id`, `assignee_id`, `created_at`, `updated_at`, relasi requester/assignee/comments/attachments, `is_overdue`. |
| Comment | `id`, `ticket_id`, `author_id`, `body`, `is_internal`, `created_at`, relasi `author`. |
| Attachment | `id`, `ticket_id`, `uploader_id`, `filename`, `file_path`, `file_size`, `content_type`, `created_at`, relasi `uploader`. |

`is_overdue` adalah hasil perhitungan service, bukan kolom tersimpan. `file_path` dapat ikut response saat terisi, tetapi bukan URL download yang digunakan UI.

### 11.4 Contoh detail berelasi

Berikut contoh field pilihan yang menunjukkan seluruh jenis relasi. Response asli juga berisi field ID/waktu lain pada object user, komentar, dan attachment sesuai entity.

```json
{
  "data": {
    "id": "10000000-0000-4000-8000-000000000001",
    "title": "VPN kantor tidak dapat digunakan",
    "description": "Koneksi terputus sesudah OTP.",
    "category": "Network",
    "priority": "urgent",
    "status": "in_progress",
    "requester_id": "20000000-0000-4000-8000-000000000001",
    "assignee_id": "30000000-0000-4000-8000-000000000001",
    "created_at": "2026-01-02T09:00:00Z",
    "updated_at": "2026-01-02T10:00:00Z",
    "requester": {
      "id": "20000000-0000-4000-8000-000000000001",
      "name": "Dewi Contoh",
      "email": "dewi@example.test",
      "role": "requester"
    },
    "assignee": {
      "id": "30000000-0000-4000-8000-000000000001",
      "name": "Agent Contoh",
      "email": "agent@example.test",
      "role": "agent"
    },
    "comments": [
      {
        "id": "40000000-0000-4000-8000-000000000001",
        "ticket_id": "10000000-0000-4000-8000-000000000001",
        "author_id": "30000000-0000-4000-8000-000000000001",
        "body": "Sedang diperiksa.",
        "is_internal": false,
        "attachments": [
          {
            "id": "50000000-0000-4000-8000-000000000002",
            "ticket_id": "10000000-0000-4000-8000-000000000001",
            "comment_id": "40000000-0000-4000-8000-000000000001",
            "filename": "hasil-pemeriksaan.txt",
            "file_size": 512,
            "content_type": "text/plain"
          }
        ],
        "author": {
          "name": "Agent Contoh"
        }
      }
    ],
    "attachments": [
      {
        "id": "50000000-0000-4000-8000-000000000001",
        "ticket_id": "10000000-0000-4000-8000-000000000001",
        "comment_id": null,
        "uploader_id": "20000000-0000-4000-8000-000000000001",
        "filename": "screenshot.png",
        "file_size": 2048,
        "content_type": "image/png",
        "uploader": {
          "name": "Dewi Contoh"
        }
      }
    ],
    "is_overdue": false
  }
}
```

Tiket in-progress pada contoh tidak overdue menurut aturan aktual meskipun prioritasnya urgent.

### 11.5 Perbedaan response create, list, dan detail

| Operasi | Relasi yang biasanya sudah diisi |
|---|---|
| Create tiket | Requester dari aktor/target admin; komentar dan attachment belum dimuat. |
| List tiket | Requester dan assignee; bukan seluruh diskusi dan lampiran. |
| Get detail | Requester, assignee, komentar+author, attachment+uploader. |
| Patch tiket | Detail diambil ulang setelah update. |

Jangan menganggap semua endpoint mengisi seluruh relasi hanya karena memakai struct `Ticket` yang sama.

### 11.6 Arti data kosong

- `priority: ""`: belum ditentukan.
- `assignee_id: null`: belum ada penanggung jawab.
- Object `assignee` yang nil dapat tidak disertakan karena `omitempty`.
- Array kosong dapat berbentuk `[]` atau `null`, tergantung cara slice repository dibentuk.
- Waktu nol `0001-01-01T00:00:00Z` pada user relasi dapat terjadi jika field waktu tidak diisi, bukan karena akun benar-benar dibuat pada tahun satu.

Query attachment tidak memuat waktu pembuatan uploader, sedangkan entity user tetap memiliki field tersebut. Ini contoh perbedaan bentuk entity dengan kelengkapan data yang dimuat.

Sumber: [entity](../backend/internal/domain/entity.go), [DTO](../backend/internal/domain/dto.go), [repository tiket](../backend/internal/repository/ticket_repo.go), [repository komentar](../backend/internal/repository/comment_repo.go), [repository lampiran](../backend/internal/repository/attachment_repo.go).

## 12. Peta Halaman Frontend ke API

### 12.1 Setiap halaman meminta data apa?

| Halaman | Fungsi penting | API yang dipakai |
|---|---|---|
| Login | `handleLogin()` | POST login melalui auth store. |
| Register | `handleRegister()` | POST register melalui auth store. |
| Daftar tiket | `fetchTickets()`, `fetchStats()`, `fetchAssignees()` | GET tiket, statistik umum non-requester, user untuk admin. |
| Create tiket | `handleSubmit()` dan pemuatan user admin | POST tiket, POST attachment opsional, GET user admin. |
| Detail tiket | `fetchTicket()`, empat update, `submitComment()`, upload/download | GET/PATCH tiket, komentar, attachment, download; GET user untuk admin. |
| Users | `fetchUsers()`, `changeRole()` | GET user dan PATCH role. |
| Statistik admin | `fetchStats()` | GET statistik admin. |
| MainLayout | Pendaftaran event dan logout | SSE; logout sendiri tidak memanggil endpoint server. |

`GET /api/me` mempunyai wrapper/action tetapi belum dipanggil otomatis. Alias `/api/tickets/stats` tidak digunakan frontend saat ini.

### 12.2 Data lokal dan data bersama

Pinia `auth` menyimpan sesi yang dipakai banyak halaman. Pinia `settings` menyimpan bahasa/tema. Daftar tiket dan detail justru disimpan pada state masing-masing view, bukan satu ticket store global.

Contoh kode asli:

```js
const ticket = ref(null)
const loading = ref(false)
const newComment = reactive({ body: '', is_internal: false })
const isAgentOrAdmin = computed(() => auth.isAgent || auth.isAdmin)
```

Penjelasan:

- `ref`: satu nilai reaktif; di script diakses memakai `.value`.
- `reactive`: object form yang field-nya dapat diubah.
- `computed`: nilai turunan yang dihitung dari state lain.
- `ticket.value` berarti isi ref, bukan pembungkus response API.
- `res.data` berarti isi field `data` pada body API.

Karena masing-masing view menyimpan datanya sendiri, perubahan satu data tidak otomatis mengganti seluruh layar. View lain memerlukan pemuatan ulang atau callback SSE yang sesuai.

### 12.3 Komponen bersama tidak memanggil API sendiri

| Komponen | Data/aksi yang diterima |
|---|---|
| `StatusBadge` | Prop status → tulisan/warna. |
| `PriorityBadge` | Prop priority → tulisan/warna atau belum ditentukan. |
| `ConfirmDialog` | Judul/pesan → emit confirm/close. |
| `ErrorAlert` | Pesan → emit retry. |
| `LoadingSkeleton` | Banyak baris placeholder. |
| `EmptyState` | Keterangan hasil kosong. |
| `Toast` | Pesan, jenis, dan status terlihat. |

Contoh hubungan:

```text
Pengguna menekan tombol status
  → view mengisi pendingAction
  → ConfirmDialog tampil
  → pengguna konfirmasi
  → dialog emit confirm
  → view menjalankan confirmAction()
  → updateStatus()
  → PATCH API
  → response mengganti ticket
  → StatusBadge menerima status terbaru
```

Pemisahan ini membuat komponen tampilan bisa dipakai kembali tanpa mengetahui alamat server.

### 12.4 Peran router guard

[`router/index.js`](../frontend/src/router/index.js) membatasi navigasi berdasarkan snapshot sesi/role lokal. Contohnya, route create hanya requester/admin dan halaman users/stats hanya admin.

Guard bukan pengaman database. Orang tetap bisa mengirim HTTP langsung tanpa tombol UI. Karena itu backend memeriksa token, role, dan kepemilikan lagi.

## 13. Contoh Perjalanan Satu Tiket

Anggap Dewi adalah requester dan Raka adalah agent.

### Langkah 1 — Dewi login

Frontend mengirim POST login. Token disimpan. Semua request Axios berikutnya membawa Bearer token.

### Langkah 2 — Dewi membuat tiket

Frontend mengirim POST tiket. Backend menetapkan:

- Pelapor: ID Dewi.
- Status: open.
- Prioritas: belum ditentukan.
- Assignee: belum ada.

Server memberi ID tiket dan memberitahu staf melalui `ticket_created`.

### Langkah 3 — File opsional diunggah

Jika ada screenshot, frontend menggunakan ID tiket untuk POST attachment. Ini operasi terpisah; kegagalan file tidak membatalkan tiket.

### Langkah 4 — Raka membaca dan menentukan prioritas

Agent membuka detail. GET detail memuat relasi pengguna, komentar, dan file. Raka memilih urgent; frontend mengirim PATCH priority.

### Langkah 5 — Raka memulai pengerjaan

Frontend mengirim status in-progress. Backend memvalidasi transisi dan menjadikan ID Raka sebagai assignee.

### Langkah 6 — Diskusi berjalan

Dewi menambah komentar publik. Raka dapat menulis komentar publik atau internal. Setiap komentar terkait tiket dan user penulis. Catatan internal disaring dari GET Dewi.

### Langkah 7 — Raka menyelesaikan penanganan

Status berubah dari in-progress ke resolved. Event memicu pembacaan ulang pada browser yang relevan.

### Langkah 8 — Dewi menutup atau membuka kembali

- Jika selesai: resolved → closed.
- Jika belum selesai: resolved → open.

Admin/agent tidak memiliki jalur reopen tersebut pada aturan sekarang. Closed tidak bisa dibuka kembali melalui state machine yang tersedia.

```mermaid
sequenceDiagram
    actor D as Dewi Requester
    actor A as Raka Agent
    participant V as Frontend
    participant B as Backend API
    participant DB as PostgreSQL
    D->>V: Buat tiket
    V->>B: POST tiket
    B->>DB: INSERT tiket open
    B-->>V: ID tiket
    A->>V: Mulai pengerjaan
    V->>B: PATCH status in_progress
    B->>DB: UPDATE status dan assignee
    B-->>V: Detail terbaru
    A->>V: Selesaikan
    V->>B: PATCH status resolved
    B->>DB: UPDATE tiket
    D->>V: Konfirmasi selesai
    V->>B: PATCH status closed
    B->>DB: UPDATE tiket
```

Diagram menyederhanakan transport; token, SSE, dialog, dan query detail tambahan tetap mengikuti penjelasan sebelumnya.

## 14. Mencoba API Secara Langsung

### 14.1 Persiapan

Pastikan backend/database sudah berjalan. Melalui Docker Compose dari root proyek:

```sh
docker compose up --build
```

Alamat API langsung dari host adalah `http://localhost:8080/api`. Frontend Docker berada di `http://localhost:3000`.

Contoh berikut memakai curl. Anda juga dapat menggunakan Postman atau koleksi [`postman_collection.json`](postman_collection.json).

**Penanda pada contoh:** ganti `TOKEN_REQUESTER`, `TOKEN_AGENT`, `ID_TIKET`, dan path file dengan nilai sebenarnya sebelum menjalankan. Contoh tidak memakai token sungguhan. User contoh perlu dibuat lebih dahulu, atau gunakan akun demo dari seeder pada lingkungan demo saja.

### 14.2 Login

```sh
curl -X POST 'http://localhost:8080/api/auth/login' \
  -H 'Content-Type: application/json' \
  -d '{"email":"dewi@example.test","password":"PasswordContoh123!"}'
```

Salin token dari `data.token` ke header request berikutnya. Jangan mengirim token kepada pihak lain atau memasukkannya ke repositori.

### 14.3 Membuat tiket sebagai requester

```sh
curl -X POST 'http://localhost:8080/api/tickets' \
  -H 'Authorization: Bearer TOKEN_REQUESTER' \
  -H 'Content-Type: application/json' \
  -d '{"title":"VPN bermasalah","description":"Terputus sesudah OTP","category":"Network"}'
```

Catat `data.id` sebagai ID tiket.

### 14.4 Melihat daftar dan detail

```sh
curl 'http://localhost:8080/api/tickets?page=1&limit=10' \
  -H 'Authorization: Bearer TOKEN_REQUESTER'

curl 'http://localhost:8080/api/tickets/ID_TIKET' \
  -H 'Authorization: Bearer TOKEN_REQUESTER'
```

### 14.5 Memulai penanganan sebagai agent

Login dahulu dengan akun agent untuk mendapatkan token agent.

```sh
curl -X PATCH 'http://localhost:8080/api/tickets/ID_TIKET' \
  -H 'Authorization: Bearer TOKEN_AGENT' \
  -H 'Content-Type: application/json' \
  -d '{"status":"in_progress"}'
```

Request yang sama menggunakan token requester ditolak oleh aturan transisi dari open ke in-progress.

### 14.6 Menambah komentar

```sh
curl -X POST 'http://localhost:8080/api/tickets/ID_TIKET/comments' \
  -H 'Authorization: Bearer TOKEN_REQUESTER' \
  -H 'Content-Type: application/json' \
  -d '{"body":"Masih bermasalah setelah restart.","is_internal":false}'
```

### 14.7 Mengunggah file

Lampiran tiket utama:

```sh
curl -X POST 'http://localhost:8080/api/tickets/ID_TIKET/attachments' \
  -H 'Authorization: Bearer TOKEN_REQUESTER' \
  -F 'file=@/path/ke/screenshot.png'
```

Lampiran komentar memakai ID komentar:

```sh
curl -X POST 'http://localhost:8080/api/comments/ID_KOMENTAR/attachments' \
  -H 'Authorization: Bearer TOKEN_PENULIS_KOMENTAR' \
  -F 'file=@/path/ke/hasil-pemeriksaan.txt'
```

Pratinjau tanpa menyimpan file ke komputer:

```sh
curl 'http://localhost:8080/api/attachments/ID_LAMPIRAN/preview' \
  -H 'Authorization: Bearer TOKEN'
```

Path pada contoh adalah penanda. `-F` membuat multipart; tidak perlu menulis boundary multipart sendiri.

### 14.8 Melihat stream SSE

```sh
curl -N 'http://localhost:8080/api/events' \
  -H 'Authorization: Bearer TOKEN_AGENT'
```

`-N` membantu menampilkan stream tanpa buffering output curl. Request ini tetap terbuka sampai dihentikan pengguna. Curl dapat memakai header Bearer; query token pada browser dipakai karena keterbatasan EventSource bawaan, bukan kewajiban semua klien SSE.

## 15. Error dan Hal yang Perlu Diwaspadai

### 15.1 Arti status HTTP

| Status | Makna | Contoh |
|---|---|---|
| 200 | Request berhasil | Detail, update, daftar. |
| 201 | Data baru berhasil dibuat | User, tiket, komentar, attachment. |
| 400 | Input tidak sesuai | JSON salah, field wajib kosong, ekstensi file ditolak. |
| 401 | Autentikasi gagal | Token hilang/kedaluwarsa atau login salah. |
| 403 | Tidak berhak | Requester membaca tiket orang lain; agent create tiket. |
| 404 | Data target tidak ada | ID resource valid tetapi tidak ditemukan. |
| 409 | Konflik | Email sudah terdaftar; aksi pada tiket Closed ditolak. |
| 422 | Transisi status tidak sah | Open langsung ke resolved. |
| 500 | Kegagalan internal | Error database/file yang tidak dipetakan khusus. |

ID berformat UUID salah belum selalu menjadi 400/404 karena validasi path belum lengkap; kegagalan cast SQL dapat menjadi error umum.

Response error biasanya:

```json
{
  "error": "Anda tidak memiliki akses untuk aksi ini"
}
```

Mapping utama ada di [`handler/response.go`](../backend/internal/handler/response.go). Binding dan middleware tertentu menulis response langsung.

### 15.2 Hal yang tampak benar di UI belum tentu aman di server

- Menyembunyikan tombol bukan pengganti pemeriksaan role backend.
- UUID yang sulit ditebak bukan pengganti pemeriksaan pemilik tiket.
- Pilihan dropdown bukan validasi enum server.
- Password hash tidak boleh dikembalikan; entity memakai `json:"-"` untuk itu.
- Token localStorage dapat dibaca script yang berjalan pada origin halaman; keamanan frontend tetap penting.

### 15.3 Batasan implementasi penting

| Hal | Kondisi saat ini |
|---|---|
| Perubahan role | JWT lama tetap membawa role lama; `/me` tidak lookup database. |
| Unassign UI | UI mengirim null; backend menganggap field tidak diubah. |
| Lampiran internal | Lampiran komentar mewarisi visibilitas komentar: berkas pada internal note tidak dikirim ke requester. |
| Isi file | Belum diperiksa MIME nyata/antivirus. |
| CORS | `ALLOWED_ORIGINS` dibaca config tetapi router masih mengirim `*`. |
| SLA | Hanya urgent-open, default dua jam, tanpa histori breach. |
| Update bersamaan | Belum memakai version/row lock untuk seluruh read-modify-write. |
| Operasi gabungan | Create+upload dan komentar+upload bukan satu transaksi. |
| Diskusi Closed | Komentar baru, edit/hapus komentar, dan pengelolaan lampiran komentar ditolak; lampiran tiket utama tetap dapat dikelola pemiliknya. |
| Realtime | Broker memori tanpa replay atau koordinasi lintas instance. |
| API tambahan | Belum ada hapus tiket, reset password, refresh token, atau revoke/logout server. |

### 15.4 Jangan menyamakan gagal membaca ulang dengan gagal menyimpan

Contoh:

1. PATCH tiket berhasil menyimpan perubahan.
2. Server mencoba mengambil detail terbaru.
3. Pembacaan detail mengalami error.
4. Klien menerima error, walaupun UPDATE sudah terjadi.

Hal serupa berlaku ketika komentar berhasil tetapi upload gagal, atau refetch browser gagal setelah mutasi. Periksa keadaan server sebelum mengulang mutasi yang berpotensi menggandakan data.

### 15.5 Panduan cepat ketika hasil tidak sesuai

| Gejala | Periksa |
|---|---|
| 401 terus-menerus | Token di header/storage dan waktu kedaluwarsa. |
| Role berubah tetapi menu sama | Snapshot `auth.user` dan JWT lama. |
| Filter tabel berubah tetapi kartu statistik tetap | Statistik memakai endpoint terpisah tanpa filter tabel. |
| Upload besar mendapat 413 | Batas body proxy/Nginx. |
| File internal terlihat requester | Model attachment tidak mempunyai flag internal. |
| High priority lama tidak overdue | Aturan SLA hanya urgent-open. |
| Ada toast tetapi angka dashboard tetap | Dashboard admin belum auto-refetch lewat SSE. |
| Memilih unassigned tidak berpengaruh | Perbedaan null dan string kosong/sentinel. |
| Detail salah setelah ID URL berubah | View belum mengawasi perubahan parameter ID. |

## 16. Panduan Membaca Sumber dan Pengujian

### 16.1 Satu contoh cara menelusuri kode

Jika ingin memahami “mengapa requester tidak bisa mengubah prioritas”, baca dengan urutan:

1. [`TicketDetailView.vue`](../frontend/src/views/TicketDetailView.vue): kontrol prioritas hanya tampil untuk staf.
2. [`ticket.service.js`](../frontend/src/services/ticket.service.js): update mengirim PATCH.
3. [`router.go`](../backend/internal/handler/router.go): PATCH diarahkan ke handler update dengan auth.
4. [`ticket_handler.go`](../backend/internal/handler/ticket_handler.go): JSON dibaca menjadi DTO update.
5. [`ticket_service.go`](../backend/internal/service/ticket_service.go): requester dengan priority non-null ditolak.
6. [`response.go`](../backend/internal/handler/response.go): forbidden menjadi HTTP 403.
7. [`api.js`](../frontend/src/services/api.js): pesan server diteruskan sebagai Error.
8. View menangkap error dan menampilkan toast.

Dengan cara ini, hubungan antara pembatasan tombol dan pengamanan server terlihat jelas.

### 16.2 Indeks sumber backend

| Berkas | Isi penting |
|---|---|
| [`cmd/api/main.go`](../backend/cmd/api/main.go) | Menyusun repository, service, handler, dan server. |
| [`domain/entity.go`](../backend/internal/domain/entity.go) | Bentuk User/Ticket/Comment/Attachment dan tag JSON. |
| [`domain/dto.go`](../backend/internal/domain/dto.go) | Request, response, filter, dan statistik. |
| [`domain/interfaces.go`](../backend/internal/domain/interfaces.go) | Kontrak method antarlapisan. |
| [`middleware/auth_middleware.go`](../backend/internal/middleware/auth_middleware.go) | JWT dan identitas aktor. |
| [`handler/router.go`](../backend/internal/handler/router.go) | Seluruh 16 pasangan method-path. |
| [`handler/auth_handler.go`](../backend/internal/handler/auth_handler.go) | Register/login/me. |
| [`handler/ticket_handler.go`](../backend/internal/handler/ticket_handler.go) | Tiket, komentar, file, statistik, SSE. |
| [`handler/user_handler.go`](../backend/internal/handler/user_handler.go) | Daftar user dan ubah role. |
| [`service/auth_service.go`](../backend/internal/service/auth_service.go) | bcrypt dan penerbitan token. |
| [`service/ticket_service.go`](../backend/internal/service/ticket_service.go) | Aturan tiket dan kepemilikan. |
| [`service/state_machine.go`](../backend/internal/service/state_machine.go) | Perpindahan status. |
| [`service/user_service.go`](../backend/internal/service/user_service.go) | Aturan perubahan role. |
| [`repository/`](../backend/internal/repository) | Query PostgreSQL dan mapping hasil. |
| [`realtime/broker.go`](../backend/internal/realtime/broker.go) | Subscriber dan audience event. |

### 16.3 Indeks sumber frontend

| Berkas | Isi penting |
|---|---|
| [`services/api.js`](../frontend/src/services/api.js) | Base URL, header token, error 401. |
| [`services/auth.service.js`](../frontend/src/services/auth.service.js) | Request login/register/me. |
| [`services/ticket.service.js`](../frontend/src/services/ticket.service.js) | Request tiket, komentar, file, statistik. |
| [`services/user.service.js`](../frontend/src/services/user.service.js) | Request user/role. |
| [`services/realtime.service.js`](../frontend/src/services/realtime.service.js) | EventSource dan callback. |
| [`stores/auth.js`](../frontend/src/stores/auth.js) | Penyimpanan token/user. |
| [`router/index.js`](../frontend/src/router/index.js) | Route halaman dan guard. |
| [`views/`](../frontend/src/views) | Form, pemicu request, dan pemakaian response. |
| [`components/shared/`](../frontend/src/components/shared) | Layout, badge, dialog, error, dan toast. |

### 16.4 Pengujian yang tersedia

Test backend berada di [`backend/tests`](../backend/tests):

- `ticket_test.go`: service, ownership, state machine, prioritas, pelapor, SLA, akses lampiran.
- `api_test.go`: router, middleware, handler, login, tiket, komentar, upload/download, statistik, user, stream dasar.
- `realtime_test.go`: audience broker.
- `mocks.go`: pengganti repository berbasis memori.

Perintah dari folder backend:

```sh
go test -v -race ./...
go vet ./...
```

Test tersebut memakai repository mock. Hasilnya bukan bukti bahwa seluruh SQL, migrasi, proxy Nginx, dan perilaku browser sudah diuji terhadap lingkungan nyata.

Perintah build dari folder frontend:

```sh
npm ci
npm run build
```

Frontend belum mempunyai script unit/E2E test dalam `package.json`. Build berhasil hanya membuktikan proyek dapat dibundle, bukan seluruh interaksi API sudah benar.

### 16.5 Ringkasan akhir

Cara memahami sistem ini adalah mengikuti satu rantai yang sama:

```text
Aksi pengguna
  → fungsi halaman Vue
  → service HTTP frontend
  → middleware dan handler backend
  → aturan service bisnis
  → query repository
  → data PostgreSQL atau file disk
  → response
  → state Vue
  → komponen yang ditampilkan
```

SSE merupakan jalur pemberitahuan tambahan. Ia biasanya memicu pembacaan API lagi, bukan membawa seluruh detail terbaru. Database menyimpan keadaan utama, backend menentukan izin dan aturan, sedangkan frontend mengatur pengalaman pengguna serta cara hasil ditampilkan.

Dokumen ini merupakan penjelasan kode dan kontrak saat disusun. Contoh request adalah panduan, bukan laporan bahwa seluruhnya telah dieksekusi. Untuk memastikan perilaku runtime, jalankan pengujian yang sesuai dan periksa hasil pada aplikasi yang benar-benar berjalan.
