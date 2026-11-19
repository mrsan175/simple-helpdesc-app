# Internal Helpdesk System

Aplikasi ini digunakan untuk melaporkan masalah dan memantau penanganannya. Setiap laporan disebut **tiket**. Contohnya: laptop tidak bisa terhubung ke Wi-Fi, printer bermasalah, atau permintaan bantuan akun kantor.

Ada tiga jenis pengguna:

- **Pelapor (requester):** membuat laporan dan melihat perkembangan laporannya sendiri.
- **Petugas (agent):** memeriksa dan menangani laporan yang masuk.
- **Pengelola (admin):** mengatur pengguna, pembagian tugas, dan melihat ringkasan laporan.

## Daftar Isi

1. [Cara Menjalankan Aplikasi](#1-cara-menjalankan-aplikasi)
2. [Akun Contoh untuk Mencoba Aplikasi](#2-akun-contoh-untuk-mencoba-aplikasi)
3. [Daftar Layanan API](#3-daftar-layanan-api)
4. [Cara Menjalankan Pengujian](#4-cara-menjalankan-pengujian)

## 1. Cara Menjalankan Aplikasi

### Cara paling mudah: menggunakan Docker

Docker membantu menjalankan semua bagian aplikasi sekaligus, sehingga Anda tidak perlu memasangnya satu per satu.

**Yang perlu disiapkan:**

- Docker Desktop, atau Docker Engine dengan Docker Compose v2, sudah terpasang dan berjalan.
- Koneksi internet untuk mengunduh kebutuhan aplikasi saat pertama kali dijalankan.
- Folder proyek ini sudah tersedia di komputer Anda.

#### Langkah 1 — Buka terminal di folder proyek

Terminal adalah tempat untuk mengetik perintah. Anda dapat menggunakan terminal di editor kode atau aplikasi terminal bawaan komputer.

Pastikan terminal berada di folder utama proyek, yaitu folder yang berisi file `docker-compose.yml` dan `README.md` ini.

#### Langkah 2 — Jalankan aplikasi

Ketik perintah berikut, lalu tekan Enter:

```bash
docker compose up --build -d
```

Saat pertama kali dijalankan, proses ini bisa memerlukan beberapa menit. Aplikasi akan menyiapkan tempat penyimpanan data, akun contoh, dan beberapa laporan contoh secara otomatis.

#### Langkah 3 — Buka aplikasi di browser

Buka **[http://localhost:3000](http://localhost:3000)** melalui Chrome, Firefox, atau browser lain.

Masuk menggunakan salah satu [akun contoh](#2-akun-contoh-untuk-mencoba-aplikasi). Untuk mencoba sebagai pengelola, gunakan `admin@helpdesk.local` dengan password `Password123!`.

Alamat `localhost` berarti aplikasi berjalan di komputer Anda sendiri.

#### Jika aplikasi belum bisa dibuka

Periksa apakah semua bagian aplikasi sudah berjalan:

```bash
docker compose ps
```

Jika masih ada masalah, lihat pesan dari bagian pengolah data aplikasi:

```bash
docker compose logs --tail=100 backend
```

Pastikan Docker aktif. Alamat layanan ini memakai port `3000`, `8080`, dan `5433`; pastikan nomor tersebut tidak sedang dipakai aplikasi lain. Jika masalah belum teratasi, kirimkan pesan kesalahannya kepada pengembang.

#### Cara menghentikan aplikasi

Dari folder utama proyek, jalankan:

```bash
docker compose down
```

Perintah ini menghentikan aplikasi tanpa menghapus data yang tersimpan. Untuk menjalankannya kembali, gunakan perintah pada langkah 2.

**Penting:** jangan menambahkan `-v` pada perintah tersebut jika data masih diperlukan. Perintah `docker compose down -v` akan menghapus data laporan, akun, dan file lampiran yang disimpan oleh pengaturan Docker proyek ini.

### Pilihan lain: menjalankan bagian aplikasi secara terpisah

Bagian ini ditujukan bagi pengembang yang ingin mengubah kode. **Jika hanya ingin mencoba aplikasi, cukup gunakan cara Docker di atas.**

Siapkan Go 1.22 atau lebih baru, Node.js 20 beserta npm, serta Docker untuk menjalankan PostgreSQL sebagai tempat penyimpanan data. Perintah berikut menggunakan terminal Linux/macOS.

**1. Jalankan penyimpanan data** dari folder utama proyek:

```bash
docker compose up -d db
```

Tunggu sampai siap. Perintah berikut akan menampilkan `accepting connections` jika sudah siap:

```bash
docker compose exec db pg_isready -U postgres -d helpdesk
```

**2. Jalankan backend**, yaitu bagian yang mengolah laporan dan menyimpan data. Buka terminal baru di folder utama proyek:

```bash
cd backend
cp .env.example .env
```

Buka file `backend/.env`, lalu sesuaikan pengaturan berikut:

```dotenv
PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5433/helpdesk?sslmode=disable
JWT_SECRET=ganti-dengan-kunci-rahasia-acak
SEED_ON_START=true
```

`DATABASE_URL` adalah alamat penyimpanan data. `JWT_SECRET` adalah kunci rahasia untuk memeriksa kode akses pengguna; ganti nilai contoh tersebut dengan nilai acak yang tidak dibagikan. `SEED_ON_START=true` berarti akun dan laporan contoh akan ditambahkan jika belum ada.

Masih dari folder `backend/`, jalankan:

```bash
go mod download
go run ./cmd/api
```

Biarkan terminal ini tetap berjalan. Layanan backend tersedia di `http://localhost:8080/api`.

**3. Jalankan frontend**, yaitu tampilan aplikasi yang dibuka di browser. Buka terminal baru lagi di folder utama proyek:

```bash
cd frontend
npm ci
VITE_API_BASE_URL=/api npm run dev
```

Biarkan terminal ini tetap berjalan, lalu buka **[http://localhost:5173](http://localhost:5173)**. Alamat ini berbeda dari cara Docker karena tampilan aplikasi dijalankan secara terpisah.

### Hal yang perlu diketahui

- File lampiran dibatasi **5 MiB per file**, atau sekitar **5 MB**. File yang lebih besar akan ditolak.
- Akun, password, dan kunci rahasia bawaan hanya untuk mencoba aplikasi. **Jangan langsung memakai pengaturan contoh untuk penggunaan kantor yang sebenarnya.**
- Sebelum dipakai sungguhan, minta pengembang atau pengelola server mengganti password dan kunci rahasia, mematikan penambahan akun contoh, mengaktifkan HTTPS agar koneksi terlindungi, serta meninjau pembatasan akses dan pencadangan data beserta lampiran.

## 2. Akun Contoh untuk Mencoba Aplikasi

Akun contoh dibuat otomatis saat aplikasi dijalankan dengan pengaturan bawaan. Dalam kode, data contoh ini disebut **seed**.

**Password semua akun di bawah ini: `Password123!`**

| Peran | Nama | Email | Yang bisa dilakukan |
|---|---|---|---|
| Pengelola (`admin`) | Admin Helpdesk | `admin@helpdesk.local` | Melihat seluruh laporan, membuat laporan, membagi tugas, mengatur pengguna, dan melihat statistik. |
| Petugas (`agent`) | Agent Helpdesk | `agent@helpdesk.local` | Melihat dan menangani laporan, mengatur prioritas, serta menulis komentar dan catatan internal. Tidak bisa membuat laporan baru. |
| Pelapor (`requester`) | Budi | `requester1@helpdesk.local` | Membuat laporan, melihat laporan miliknya sendiri, serta mengirim komentar dan lampiran. |
| Pelapor (`requester`) | Siti | `requester2@helpdesk.local` | Sama seperti Budi, tetapi tidak bisa melihat laporan milik Budi. |

**Saran mencoba:** masuk sebagai Budi untuk membuat laporan, lalu masuk sebagai petugas untuk melihat dan menanganinya.

Catatan:

- Pada penyimpanan data yang masih kosong, aplikasi menambahkan **empat akun dan 13 laporan contoh**.
- Akun yang sudah ada tidak dibuat ulang. Password dan peran akun tersebut juga tidak dikembalikan ke pengaturan awal.
- Tombol akun demo di halaman masuk hanya mengisi email dan password. Anda tetap perlu menekan tombol masuk.
- Pendaftaran akun baru melalui aplikasi menghasilkan akun pelapor, bukan petugas atau pengelola.

## 3. Daftar Layanan API

**API** adalah jalur komunikasi antara tampilan aplikasi dan bagian yang mengolah data. **Endpoint** adalah alamat untuk meminta suatu tindakan, misalnya mengambil daftar laporan atau mengirim komentar.

**Jika hanya memakai aplikasi melalui browser, Anda tidak perlu mengetik alamat-alamat di bawah ini.** Daftar ini disediakan bagi pengembang atau pihak yang ingin menghubungkan aplikasi dengan sistem lain.

Alamat backend adalah `http://localhost:8080`. Jika melalui layanan tampilan Docker, alamatnya `http://localhost:3000`. Tambahkan alamat pada tabel berikut setelah alamat tersebut.

Arti jenis permintaan pada kolom **Metode**:

- **GET:** melihat atau mengambil informasi.
- **POST:** mengirim data, misalnya membuat laporan atau masuk ke akun.
- **PATCH:** mengubah sebagian informasi yang sudah ada.

Tulisan `{id}` harus diganti dengan kode unik laporan, lampiran, atau pengguna yang dituju, sesuai keterangannya.

### Akun dan proses masuk

| Metode | Alamat endpoint | Siapa yang boleh memakai | Kegunaan |
|---|---|---|---|
| POST | `/api/auth/register` | Siapa saja | Mendaftarkan akun pelapor baru. |
| POST | `/api/auth/login` | Siapa saja | Masuk dengan email dan password untuk mendapatkan kode akses. |
| GET | `/api/me` | Pengguna yang sudah masuk | Melihat informasi akun berdasarkan sesi masuk saat ini. |

### Laporan dan komentar

| Metode | Alamat endpoint | Siapa yang boleh memakai | Kegunaan |
|---|---|---|---|
| GET | `/api/tickets` | Pengguna yang sudah masuk | Melihat daftar laporan. Pelapor hanya melihat miliknya sendiri; petugas dan pengelola dapat melihat seluruh laporan. |
| POST | `/api/tickets` | Pelapor dan pengelola | Membuat laporan baru. |
| GET | `/api/tickets/{id}` | Pemilik laporan, petugas, pengelola | Melihat isi laporan, komentar, dan lampirannya. |
| PATCH | `/api/tickets/{id}` | Sesuai peran dan kepemilikan laporan | Mengubah status sesuai aturan. Prioritas hanya diatur petugas/pengelola; pembagian tugas dan penggantian pelapor hanya oleh pengelola. |
| POST | `/api/tickets/{id}/comments` | Pemilik laporan, petugas, pengelola | Menambahkan komentar. Catatan internal hanya bisa dibuat dan dibaca petugas/pengelola. |

Pada tabel ini, `{id}` adalah kode unik laporan. Daftar laporan mendukung pencarian judul, penyaringan, pengurutan, dan pembagian hasil ke beberapa halaman.

### Lampiran dan pemberitahuan

| Metode | Alamat endpoint | Siapa yang boleh memakai | Kegunaan |
|---|---|---|---|
| POST | `/api/tickets/{id}/attachments` | Pemilik laporan, petugas, pengelola | Mengunggah file ke laporan yang dituju, maksimal 5 MiB per file. `{id}` adalah kode laporan. |
| GET | `/api/attachments/{id}/download` | Pemilik laporan terkait, petugas, pengelola | Mengunduh file lampiran. `{id}` adalah kode lampiran, bukan kode laporan. |
| GET | `/api/events` | Pengguna yang sudah masuk | Menerima pemberitahuan perubahan laporan sesuai hak aksesnya. |

**Penting:** file lampiran melekat pada laporan, bukan pada komentar. File yang dikirim bersama catatan internal tetap bisa dilihat pemilik laporan. Jangan melampirkan file rahasia melalui cara tersebut.

### Ringkasan laporan dan pengaturan pengguna

| Metode | Alamat endpoint | Siapa yang boleh memakai | Kegunaan |
|---|---|---|---|
| GET | `/api/stats` | Pengguna yang sudah masuk | Melihat jumlah laporan berdasarkan status dan laporan mendesak yang belum selesai. Pelapor hanya mendapat ringkasan laporannya sendiri. |
| GET | `/api/tickets/stats` | Pengguna yang sudah masuk | Alamat lain dengan fungsi yang sama seperti `/api/stats`. |
| GET | `/api/admin/stats` | Pengelola | Melihat ringkasan lebih lengkap, termasuk laporan terlambat dan pembagian pekerjaan petugas. |
| GET | `/api/users` | Petugas dan pengelola | Pengelola melihat seluruh pengguna; petugas hanya melihat daftar petugas dan pengelola. |
| PATCH | `/api/users/{id}/role` | Pengelola | Mengubah peran pengguna. `{id}` adalah kode pengguna. |

### Catatan untuk pengguna API

- Selain daftar dan masuk akun, semua layanan memerlukan kode akses hasil login, yang disebut **token**. Untuk pemanggilan API langsung, kirim melalui header berikut:

  ```http
  Authorization: Bearer <token-hasil-login>
  ```

- Jika peran pengguna diubah, pengguna perlu masuk ulang untuk mendapatkan kode akses dengan peran terbaru. Kode akses lama tidak langsung dicabut oleh perubahan tersebut.
- Angka laporan terlambat pada `/api/stats` dan `/api/tickets/stats` belum dihitung. Gunakan statistik pengelola untuk melihat jumlah tersebut.
- Belum tersedia layanan untuk menghapus laporan, mengubah/menghapus komentar, mengatur ulang password, atau memperbarui kode akses tanpa masuk ulang.

## 4. Cara Menjalankan Pengujian

Pengujian otomatis membantu memeriksa apakah fungsi aplikasi masih bekerja sesuai aturan. Bagian ini terutama ditujukan bagi pengembang; **pengguna biasa tidak perlu menjalankannya untuk memakai aplikasi**.

### Menguji bagian pengolah data (backend)

Pastikan Go 1.22 atau lebih baru sudah terpasang. Buka terminal di folder utama proyek, lalu jalankan:

```bash
cd backend
go test -v ./...
```

Tunggu sampai selesai. Tulisan `PASS` dan hasil akhir `ok` menunjukkan pengujian berhasil. Tulisan `FAIL` menunjukkan ada pengujian yang gagal; pesan di atasnya menjelaskan masalah yang ditemukan. Baris `[no test files]` bukan kegagalan, melainkan berarti bagian tersebut tidak memiliki file pengujian sendiri.

Hal yang diperiksa antara lain:

- Proses pendaftaran dan masuk akun.
- Pembatasan laporan agar pelapor tidak bisa melihat laporan orang lain.
- Tindakan yang boleh dilakukan setiap peran.
- Perubahan status laporan dan perlindungan catatan internal.
- Pengiriman dan pengunduhan lampiran, termasuk batas ukuran 5 MiB.
- Penandaan laporan terlambat dan penerima pemberitahuan.

Pengujian ini memakai data tiruan, sehingga **tidak perlu menyalakan aplikasi, Docker, atau PostgreSQL, maupun membuat file pengaturan `.env`**. Koneksi internet mungkin diperlukan saat pertama kali mengunduh kebutuhan pengujian. Pengujian ini belum memeriksa penyimpanan data ke PostgreSQL yang sebenarnya.

### Memeriksa apakah tampilan aplikasi bisa disiapkan

Buka terminal baru di folder utama proyek. Pastikan Node.js 20 dan npm sudah terpasang, lalu jalankan:

```bash
cd frontend
npm ci
npm run build
```

Perintah ini menyiapkan file tampilan aplikasi untuk dijalankan. Jika berhasil, hasilnya tersedia di folder `frontend/dist`.

**Pemeriksaan ini belum menguji klik tombol atau perilaku halaman di browser.** Pengujian tampilan otomatis belum tersedia, sehingga alur penggunaan tetap perlu dicoba langsung.

Pemeriksaan kode backend, pengujian backend, dan penyiapan tampilan juga dijalankan otomatis oleh GitHub ketika perubahan dikirim atau diajukan untuk digabungkan ke cabang `main` atau `master`.
