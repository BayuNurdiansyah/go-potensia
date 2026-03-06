# Potensia Backend API

REST API untuk aplikasi bimbel **Potensia** — dibangun dengan Go, Gin, GORM, dan PostgreSQL.

---

## Daftar Isi

- [Setup](#setup)
- [Environment Variables](#environment-variables)
- [Struktur Project](#struktur-project)
- [Autentikasi](#autentikasi)
- [Response Format](#response-format)
- [HTTP Status Codes](#http-status-codes)
- [API Reference](#api-reference)
  - [Health Check](#health-check)
  - [Auth](#auth)
  - [Public — Mentor Search](#public--mentor-search)
  - [Notifications (All Roles)](#notifications-all-roles)
  - [Mentor](#mentor)
  - [Parent](#parent)

---

## Setup

```bash
# 1. Clone & masuk ke direktori
git clone <repo-url>
cd go-potensia

# 2. Copy dan isi environment variables
cp .env.example .env

# 3. Install dependencies
go mod tidy

# 4. Jalankan server
go run main.go
```

Server berjalan di `http://localhost:8080` secara default.

---

## Environment Variables

| Variable              | Wajib | Keterangan                                       |
|-----------------------|-------|--------------------------------------------------|
| `DATABASE_URL`        | ✅    | PostgreSQL connection string                     |
| `JWT_SECRET`          | ✅    | Secret key JWT (minimal 32 karakter)             |
| `PORT`                | ❌    | Port server (default: `8080`)                    |
| `APP_NAME`            | ❌    | Nama aplikasi di email (default: `Potensia`)     |
| `APP_URL`             | ✅    | Base URL frontend, untuk link reset password     |
| `BREVO_API_KEY`       | ✅    | API key dari [brevo.com](https://brevo.com)      |
| `BREVO_SENDER_EMAIL`  | ✅    | Email pengirim yang sudah diverifikasi di Brevo  |
| `BREVO_SENDER_NAME`   | ❌    | Nama pengirim email (default: `APP_NAME`)        |

---

## Struktur Project

```
go-potensia/
├── config/               # Koneksi database & auto-migrate
├── controllers/          # Handler per fitur
│   ├── auth.go
│   ├── mentor.go
│   ├── parent.go
│   ├── course.go
│   └── notification.go
├── middlewares/          # JWT auth & role guard
├── models/               # GORM structs (10 file)
├── routes/               # Definisi routing
├── utils/                # JWT, OTP, Email, Validator, Response helper
└── main.go
```

---

## Autentikasi

API yang membutuhkan login menggunakan **Bearer Token** di header:

```
Authorization: Bearer <token>
```

Token didapat dari response `POST /api/v1/auth/login` atau `POST /api/v1/auth/verify-otp`.

Token berlaku **24 jam**.

---

## Response Format

Semua response menggunakan format JSON. Struktur umum:

**Sukses:**
```json
{
  "message": "Keterangan sukses",
  "data_key": { }
}
```

**Error:**
```json
{
  "message": "Keterangan error"
}
```

**Error dengan detail tambahan:**
```json
{
  "message": "OTP salah. Sisa percobaan: 3",
  "attempts_remaining": 3
}
```

---

## HTTP Status Codes

| Code | Keterangan                                    |
|------|-----------------------------------------------|
| 200  | OK — request berhasil                         |
| 201  | Created — data berhasil dibuat                |
| 400  | Bad Request — input tidak valid               |
| 401  | Unauthorized — token tidak ada / tidak valid  |
| 403  | Forbidden — role tidak punya akses            |
| 404  | Not Found — data tidak ditemukan              |
| 409  | Conflict — data sudah ada (misal email duplikat) |
| 429  | Too Many Requests — rate limit tercapai       |
| 500  | Internal Server Error                         |

---

## API Reference

### Health Check

#### `GET /health`

Cek apakah server berjalan.

**Auth:** Tidak perlu

**Response `200`:**
```json
{
  "status": "ok",
  "service": "go-potensia"
}
```

---

### Auth

Base path: `/api/v1/auth`

---

#### `POST /api/v1/auth/register`

Daftar akun baru. Setelah berhasil, kode OTP 6 digit dikirim ke email untuk verifikasi.

**Auth:** Tidak perlu

**Request Body:**
```json
{
  "name": "Hendra Wijaya",
  "email": "hendra@example.com",
  "phone": "081234567890",
  "password": "hendra123",
  "role": "parent"
}
```

| Field      | Tipe   | Wajib | Keterangan                          |
|------------|--------|-------|-------------------------------------|
| `name`     | string | ✅    | Nama lengkap                        |
| `email`    | string | ✅    | Format email valid                  |
| `phone`    | string | ❌    | Nomor HP (10–15 digit)              |
| `password` | string | ✅    | Min 8 karakter, harus ada huruf & angka |
| `role`     | string | ✅    | `"parent"` atau `"mentor"`          |

**Response `201`:**
```json
{
  "message": "Register berhasil, cek email untuk kode OTP",
  "email": "hendra@example.com"
}
```

**Response `409` — email sudah terdaftar & terverifikasi:**
```json
{
  "message": "Email sudah terdaftar"
}
```

**Response `200` — email terdaftar tapi belum diverifikasi:**
```json
{
  "message": "Email sudah terdaftar tapi belum diverifikasi. OTP baru telah dikirim.",
  "email": "hendra@example.com"
}
```

---

#### `POST /api/v1/auth/verify-otp`

Verifikasi kode OTP yang dikirim ke email saat register. Jika berhasil, langsung mendapat JWT token (auto-login).

**Auth:** Tidak perlu

**Request Body:**
```json
{
  "email": "hendra@example.com",
  "otp": "123456"
}
```

| Field   | Tipe   | Wajib | Keterangan              |
|---------|--------|-------|-------------------------|
| `email` | string | ✅    | Email yang didaftarkan  |
| `otp`   | string | ✅    | Kode OTP 6 digit        |

**Response `200`:**
```json
{
  "message": "Verifikasi berhasil",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "name": "Hendra Wijaya",
    "email": "hendra@example.com",
    "role": "parent"
  }
}
```

**Response `401` — OTP salah:**
```json
{
  "message": "OTP salah. Sisa percobaan: 4",
  "attempts_remaining": 4
}
```

**Response `401` — OTP kadaluarsa:**
```json
{
  "message": "OTP sudah kadaluarsa. Silakan minta OTP baru."
}
```

**Response `429` — terlalu banyak percobaan (maks 5x):**
```json
{
  "message": "Terlalu banyak percobaan. Silakan minta OTP baru.",
  "retry_after": 0
}
```

> OTP berlaku **5 menit** dan maksimal **5x percobaan**.

---

#### `POST /api/v1/auth/resend-otp`

Kirim ulang kode OTP ke email. Hanya bisa dilakukan tiap 60 detik.

**Auth:** Tidak perlu

**Request Body:**
```json
{
  "email": "hendra@example.com"
}
```

**Response `200`:**
```json
{
  "message": "OTP berhasil dikirim ulang"
}
```

**Response `429`:**
```json
{
  "message": "Tunggu 60 detik sebelum kirim ulang OTP",
  "retry_after": 45
}
```

> Selalu return `200` jika email tidak ditemukan (security: tidak bocorkan info email).

---

#### `POST /api/v1/auth/login`

Login dengan email dan password.

**Auth:** Tidak perlu

**Request Body:**
```json
{
  "email": "hendra@example.com",
  "password": "hendra123"
}
```

**Response `200`:**
```json
{
  "message": "Login berhasil",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "name": "Hendra Wijaya",
    "email": "hendra@example.com",
    "phone": "081234567890",
    "role": "parent"
  }
}
```

**Response `401` — email/password salah:**
```json
{
  "message": "Email atau password salah"
}
```

**Response `403` — belum verifikasi OTP:**
```json
{
  "message": "Akun belum diverifikasi. Cek email untuk kode OTP.",
  "email": "hendra@example.com"
}
```

---

#### `POST /api/v1/auth/forgot-password`

Request link reset password. Link dikirim ke email, berlaku **15 menit** dan hanya bisa digunakan **sekali**.

**Auth:** Tidak perlu

**Request Body:**
```json
{
  "email": "hendra@example.com"
}
```

**Response `200`:**
```json
{
  "message": "Jika email terdaftar dan terverifikasi, link reset password akan dikirim"
}
```

> Selalu return `200` apapun kondisinya (security: tidak bocorkan info email).

---

#### `GET /api/v1/auth/verify-reset-token`

Validasi token reset password sebelum menampilkan form reset di frontend.

**Auth:** Tidak perlu

**Query Params:**

| Param   | Wajib | Keterangan                |
|---------|-------|---------------------------|
| `token` | ✅    | Token dari link di email  |

**Contoh:** `GET /api/v1/auth/verify-reset-token?token=abc123...`

**Response `200`:**
```json
{
  "message": "Token valid",
  "email": "hendra@example.com"
}
```

**Response `400` — token tidak valid / kadaluarsa / sudah dipakai:**
```json
{
  "message": "Token sudah kadaluarsa"
}
```

---

#### `POST /api/v1/auth/reset-password`

Set password baru menggunakan token dari email.

**Auth:** Tidak perlu

**Request Body:**
```json
{
  "token": "abc123def456...",
  "new_password": "newpass99",
  "confirm_password": "newpass99"
}
```

| Field              | Tipe   | Wajib | Keterangan                              |
|--------------------|--------|-------|-----------------------------------------|
| `token`            | string | ✅    | Token dari link email                   |
| `new_password`     | string | ✅    | Min 8 karakter, harus ada huruf & angka |
| `confirm_password` | string | ✅    | Harus sama dengan `new_password`        |

**Response `200`:**
```json
{
  "message": "Password berhasil direset. Silakan login dengan password baru."
}
```

---

#### `POST /api/v1/auth/change-password`

Ubah password untuk akun yang sedang login.

**Auth:** ✅ Bearer Token (semua role)

**Request Body:**
```json
{
  "old_password": "hendra123",
  "new_password": "newpass99",
  "confirm_password": "newpass99"
}
```

**Response `200`:**
```json
{
  "message": "Password berhasil diubah"
}
```

**Response `400` — password lama salah:**
```json
{
  "message": "Password lama salah"
}
```

---

#### `POST /api/v1/auth/delete-account`

Hapus akun (soft delete — akun dinonaktifkan, bukan dihapus permanen).

**Auth:** ✅ Bearer Token (semua role)

**Request Body:**
```json
{
  "password": "hendra123"
}
```

**Response `200`:**
```json
{
  "message": "Akun berhasil dihapus"
}
```

---

### Public — Mentor Search

---

#### `GET /api/v1/mentors`

Cari dan filter mentor yang tersedia.

**Auth:** Tidak perlu

**Query Params:**

| Param      | Wajib | Keterangan                                        |
|------------|-------|---------------------------------------------------|
| `search`   | ❌    | Cari berdasarkan nama mentor                      |
| `province` | ❌    | Filter berdasarkan provinsi (contoh: `JAWA BARAT`) |
| `regency`  | ❌    | Filter berdasarkan kabupaten/kota                 |
| `district` | ❌    | Filter berdasarkan kecamatan                      |
| `category` | ❌    | Filter berdasarkan kategori kursus (contoh: `MATEMATIKA`) |
| `page`     | ❌    | Halaman (default: `1`)                            |
| `limit`    | ❌    | Jumlah per halaman (default: `20`, maks: `100`)   |

**Contoh:** `GET /api/v1/mentors?province=JAWA BARAT&category=MATEMATIKA&page=1&limit=10`

**Response `200`:**
```json
{
  "mentors": [
    {
      "id": 1,
      "user_id": 2,
      "user": {
        "id": 2,
        "name": "Budi Santoso",
        "avatar_url": null
      },
      "expertise": "Matematika & Fisika",
      "bio": "Berpengalaman mengajar olimpiade matematika selama 5 tahun.",
      "rating": 4.8,
      "total_review": 124,
      "total_students": 42,
      "province": "JAWA BARAT",
      "regency": "BANDUNG",
      "district": "COBLONG"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 10
}
```

---

#### `GET /api/v1/mentors/:mentor_id`

Lihat profil publik lengkap seorang mentor beserta kursus dan ulasan terbaru.

**Auth:** Tidak perlu

**Path Param:** `mentor_id` — ID mentor profile

**Contoh:** `GET /api/v1/mentors/1`

**Response `200`:**
```json
{
  "mentor": {
    "id": 1,
    "name": "Budi Santoso",
    "avatar": null,
    "expertise": "Matematika & Fisika",
    "bio": "Berpengalaman mengajar olimpiade matematika selama 5 tahun.",
    "rating": 4.8,
    "total_review": 124,
    "total_students": 42,
    "total_sessions": 148,
    "province": "JAWA BARAT",
    "regency": "BANDUNG",
    "district": "COBLONG"
  },
  "certificates": [
    { "id": 1, "mentor_id": 1, "title": "Certified Educator", "issuer": "Kemendikbud RI", "year": "2021" }
  ],
  "achievements": [
    { "id": 1, "mentor_id": 1, "text": "Mentor Terbaik 2023" }
  ],
  "gallery": [
    { "id": 1, "mentor_id": 1, "image_url": "https://...", "sort_order": 0 }
  ],
  "education": [
    { "id": 1, "mentor_id": 1, "degree": "S1 Pendidikan Matematika", "institution": "Universitas Indonesia", "year": "2018" }
  ],
  "courses": [
    {
      "id": 1,
      "title": "Matematika Olimpiade",
      "category": "MATEMATIKA",
      "status": "active",
      "rating": 4.8,
      "total_review": 124,
      "competencies": [
        { "id": 1, "text": "Logika Dasar", "sort_order": 0 }
      ],
      "packages": [
        { "id": 1, "name": "Paket Starter", "duration_per_session": 60, "total_sessions": 4, "price": 299000, "original_price": 350000, "is_highlight": false },
        { "id": 2, "name": "Paket Reguler", "duration_per_session": 90, "total_sessions": 8, "price": 549000, "original_price": 750000, "is_highlight": true }
      ]
    }
  ],
  "reviews": [
    { "id": 1, "rating": 5, "comment": "Sangat sabar dan mudah dipahami.", "reviewer_name": "Andi Pratama", "course_name": "Matematika Olimpiade", "created_at": "2025-01-15T10:00:00Z" }
  ]
}
```

---

### Notifications (All Roles)

---

#### `GET /api/v1/notifications`

Ambil daftar notifikasi user yang sedang login (maks 50 terbaru).

**Auth:** ✅ Bearer Token (semua role)

**Response `200`:**
```json
{
  "notifications": [
    {
      "id": 1,
      "user_id": 1,
      "type": "schedule",
      "title": "Sesi hari ini jam 09:00",
      "body": "Jangan lupa sesi Matematika bersama Kak Budi hari ini.",
      "is_read": false,
      "ref_id": 5,
      "ref_type": "session",
      "created_at": "2025-01-20T07:00:00Z"
    }
  ],
  "unread_count": 3
}
```

> `type` bisa: `reminder`, `promo`, `schedule`, `info`

---

#### `PUT /api/v1/notifications/:notif_id/read`

Tandai satu notifikasi sebagai sudah dibaca. Gunakan `all` sebagai `notif_id` untuk tandai semua.

**Auth:** ✅ Bearer Token (semua role)

**Contoh satu notif:** `PUT /api/v1/notifications/1/read`

**Contoh semua:** `PUT /api/v1/notifications/all/read`

**Request Body:** Tidak ada

**Response `200`:**
```json
{
  "message": "Notifikasi ditandai sudah dibaca"
}
```

---

#### `GET /api/v1/notifications/preferences`

Ambil preferensi notifikasi user.

**Auth:** ✅ Bearer Token (semua role)

**Response `200`:**
```json
{
  "reminder": true,
  "promo": true,
  "schedule": true,
  "info": false
}
```

---

#### `PUT /api/v1/notifications/preferences`

Update preferensi notifikasi user.

**Auth:** ✅ Bearer Token (semua role)

**Request Body:**
```json
{
  "reminder": true,
  "promo": false,
  "schedule": true,
  "info": true
}
```

**Response `200`:**
```json
{
  "message": "Preferensi notifikasi berhasil diperbarui"
}
```

---

### Mentor

Base path: `/api/v1/mentor`

**Auth:** ✅ Bearer Token dengan role `mentor`

---

#### `GET /api/v1/mentor/dashboard`

Ambil data dashboard mentor: statistik, jadwal hari ini, dan siswa aktif.

**Response `200`:**
```json
{
  "mentor": {
    "id": 1,
    "name": "Budi Santoso",
    "avatar": null,
    "expertise": "Matematika & Fisika",
    "rating": 4.8,
    "total_students": 24
  },
  "stats": {
    "total_students": 24,
    "session_today": 3,
    "session_this_week": 12,
    "completed_sessions": 148,
    "earnings": 3200000
  },
  "today_schedule": [
    {
      "id": 1,
      "order_id": 5,
      "mentor_id": 1,
      "child_id": 2,
      "scheduled_at": "2025-01-20T09:00:00Z",
      "duration": 60,
      "status": "upcoming",
      "meet_link": "https://meet.google.com/xxx",
      "session_number": 3,
      "topic": "",
      "notes": "",
      "stars": 0
    }
  ],
  "active_students": [ ]
}
```

---

#### `GET /api/v1/mentor/profile`

Ambil profil lengkap mentor yang sedang login.

**Response `200`:**
```json
{
  "id": 1,
  "user_id": 2,
  "name": "Budi Santoso",
  "email": "budi@example.com",
  "phone": "081122334455",
  "avatar": null,
  "expertise": "Matematika & Fisika",
  "bio": "Berpengalaman 5 tahun.",
  "rating": 4.8,
  "total_review": 124,
  "total_students": 24,
  "total_sessions": 148,
  "province": "JAWA BARAT",
  "regency": "BANDUNG",
  "district": "COBLONG",
  "address": "Jl. Dipatiukur No. 10",
  "bank": {
    "bank": "BCA",
    "account": "1234567890",
    "account_name": "Budi Santoso"
  },
  "certificates": [ ],
  "achievements": [ ],
  "gallery": [ ],
  "education": [ ]
}
```

---

#### `PUT /api/v1/mentor/profile`

Update profil mentor. Semua field opsional — hanya field yang dikirim yang akan diupdate.

**Request Body:**
```json
{
  "name": "Budi Santoso",
  "phone": "081122334455",
  "expertise": "Matematika, Fisika & Kimia",
  "bio": "Berpengalaman 6 tahun mengajar olimpiade sains.",
  "province": "JAWA BARAT",
  "regency": "BANDUNG",
  "district": "COBLONG",
  "address": "Jl. Dipatiukur No. 10",
  "bank_name": "BCA",
  "bank_account": "1234567890",
  "bank_account_name": "Budi Santoso"
}
```

**Response `200`:**
```json
{
  "message": "Profil berhasil diperbarui"
}
```

---

#### `GET /api/v1/mentor/students`

Ambil daftar siswa (berdasarkan order aktif mentor ini).

**Query Params:**

| Param    | Wajib | Keterangan                                  |
|----------|-------|---------------------------------------------|
| `status` | ❌    | `active`, `completed`, `pending`, `cancelled` |
| `search` | ❌    | Cari berdasarkan nama anak atau nama kursus |

**Response `200`:**
```json
{
  "students": [
    {
      "id": 5,
      "parent_id": 1,
      "child_id": 2,
      "course_id": 1,
      "package_id": 2,
      "mentor_id": 1,
      "total_sessions": 8,
      "completed_sessions": 3,
      "remaining_sessions": 5,
      "status": "active",
      "child": { "id": 2, "name": "ALDI WIJAYA", "education": "SD Kelas 4" },
      "course": { "id": 1, "title": "Matematika Olimpiade", "category": "MATEMATIKA" }
    }
  ],
  "total": 1
}
```

---

#### `GET /api/v1/mentor/students/:order_id`

Lihat detail siswa tertentu berdasarkan order: data anak, sesi, dan skill progress.

**Path Param:** `order_id`

**Response `200`:**
```json
{
  "order": {
    "id": 5,
    "child": { "id": 2, "name": "ALDI WIJAYA", "birth_date": "2011-05-10T00:00:00Z", "gender": "Laki-laki", "education": "SD Kelas 4" },
    "course": { "id": 1, "title": "Matematika Olimpiade" },
    "package": { "id": 2, "name": "Paket Reguler", "total_sessions": 8 },
    "total_sessions": 8,
    "completed_sessions": 3,
    "remaining_sessions": 5,
    "status": "active"
  },
  "sessions": [
    { "id": 1, "session_number": 1, "scheduled_at": "2025-01-10T09:00:00Z", "duration": 90, "status": "completed", "topic": "Logika Dasar", "notes": "Sangat cepat mengerti.", "stars": 5 },
    { "id": 2, "session_number": 2, "scheduled_at": "2025-01-13T09:00:00Z", "duration": 90, "status": "upcoming", "topic": "", "notes": "", "stars": 0 }
  ],
  "skills": [
    { "id": 1, "skill_name": "Logika Dasar", "progress": 80 },
    { "id": 2, "skill_name": "Aljabar", "progress": 40 }
  ]
}
```

---

#### `GET /api/v1/mentor/schedule`

Ambil jadwal sesi mengajar mentor.

**Query Params:**

| Param  | Wajib | Keterangan                                                         |
|--------|-------|--------------------------------------------------------------------|
| `date` | ❌    | Filter tanggal format `YYYY-MM-DD`. Jika kosong, tampilkan 7 hari ke depan. |

**Contoh:** `GET /api/v1/mentor/schedule?date=2025-01-20`

**Response `200`:**
```json
{
  "schedule": [
    {
      "id": 1,
      "order_id": 5,
      "mentor_id": 1,
      "child_id": 2,
      "scheduled_at": "2025-01-20T09:00:00Z",
      "duration": 90,
      "status": "upcoming",
      "meet_link": "https://meet.google.com/xxx",
      "session_number": 4,
      "order": {
        "child": { "name": "ALDI WIJAYA" },
        "course": { "title": "Matematika Olimpiade" }
      }
    }
  ],
  "total": 1
}
```

---

#### `PUT /api/v1/mentor/sessions/:session_id`

Update data sesi: topik, catatan, bintang siswa, atau status sesi.

**Path Param:** `session_id`

**Request Body:** (semua field opsional)
```json
{
  "topic": "Persamaan Kuadrat",
  "notes": "Aldi sudah memahami konsep dasar dengan baik.",
  "stars": 4,
  "status": "completed"
}
```

| Field    | Tipe   | Keterangan                                                        |
|----------|--------|-------------------------------------------------------------------|
| `topic`  | string | Topik yang dipelajari sesi ini                                    |
| `notes`  | string | Catatan mentor untuk orang tua                                    |
| `stars`  | int    | Penilaian bintang untuk siswa: `1`–`5`                           |
| `status` | string | `upcoming`, `ongoing`, `completed`, atau `cancelled`             |

> Ketika `status` diubah ke `completed`, counter `completed_sessions` dan `remaining_sessions` di Order otomatis diperbarui.

**Response `200`:**
```json
{
  "message": "Sesi berhasil diperbarui",
  "session": { ... }
}
```

---

#### `GET /api/v1/mentor/courses`

Ambil semua kursus milik mentor yang sedang login.

**Response `200`:**
```json
{
  "courses": [
    {
      "id": 1,
      "mentor_id": 1,
      "title": "Matematika Olimpiade",
      "category": "MATEMATIKA",
      "status": "active",
      "rating": 4.8,
      "total_review": 124,
      "total_students": 42,
      "active_students": 28,
      "packages": [
        { "id": 1, "name": "Paket Starter", "total_sessions": 4, "price": 299000, "is_active": true }
      ]
    }
  ],
  "total": 1
}
```

---

#### `POST /api/v1/mentor/courses`

Buat kursus baru.

**Request Body:**
```json
{
  "title": "Matematika Olimpiade",
  "category": "MATEMATIKA",
  "description": "Kursus persiapan olimpiade matematika tingkat SMP.",
  "status": "draft",
  "competencies": [
    "Logika & Algoritma Dasar",
    "Pemecahan Masalah",
    "Teori Bilangan"
  ],
  "packages": [
    {
      "name": "Paket Starter",
      "duration_per_session": 60,
      "total_sessions": 4,
      "price": 299000,
      "original_price": 350000,
      "is_highlight": false
    },
    {
      "name": "Paket Reguler",
      "duration_per_session": 90,
      "total_sessions": 8,
      "price": 549000,
      "original_price": 750000,
      "is_highlight": true
    }
  ]
}
```

| Field           | Tipe     | Wajib | Keterangan                        |
|-----------------|----------|-------|-----------------------------------|
| `title`         | string   | ✅    | Judul kursus                      |
| `category`      | string   | ✅    | Kategori (contoh: `MATEMATIKA`)   |
| `description`   | string   | ❌    | Deskripsi kursus                  |
| `status`        | string   | ❌    | `"draft"` (default) atau `"active"` |
| `competencies`  | []string | ❌    | Daftar kompetensi yang dipelajari |
| `packages`      | []object | ❌    | Daftar paket harga                |

**Response `201`:**
```json
{
  "message": "Kursus berhasil dibuat",
  "course_id": 3
}
```

---

#### `GET /api/v1/mentor/courses/:course_id`

Lihat detail satu kursus milik mentor beserta sertifikat, prestasi, dan galeri.

**Path Param:** `course_id`

**Response `200`:**
```json
{
  "course": {
    "id": 1,
    "title": "Matematika Olimpiade",
    "category": "MATEMATIKA",
    "description": "...",
    "status": "active",
    "competencies": [ { "id": 1, "text": "Logika Dasar", "sort_order": 0 } ],
    "packages": [ { "id": 1, "name": "Paket Starter", "price": 299000 } ]
  },
  "certificates": [ ],
  "achievements": [ ],
  "gallery": [ ]
}
```

---

#### `PUT /api/v1/mentor/courses/:course_id`

Update data kursus. Semua field opsional.

**Path Param:** `course_id`

**Request Body:**
```json
{
  "title": "Matematika Olimpiade SMP",
  "category": "MATEMATIKA",
  "description": "Deskripsi diperbarui.",
  "status": "active",
  "competencies": [
    "Logika Dasar",
    "Aljabar",
    "Geometri"
  ]
}
```

> Jika `competencies` dikirim, seluruh data kompetensi lama akan **diganti** dengan yang baru.

**Response `200`:**
```json
{
  "message": "Kursus berhasil diperbarui"
}
```

---

#### `DELETE /api/v1/mentor/courses/:course_id`

Hapus kursus. Tidak bisa dihapus jika masih ada order aktif atau pending.

**Path Param:** `course_id`

**Response `200`:**
```json
{
  "message": "Kursus berhasil dihapus"
}
```

**Response `400` — ada order aktif:**
```json
{
  "message": "Kursus tidak dapat dihapus karena masih ada order aktif"
}
```

---

#### `GET /api/v1/mentor/reviews`

Ambil semua ulasan yang masuk untuk mentor ini.

**Query Params:**

| Param       | Wajib | Keterangan              |
|-------------|-------|-------------------------|
| `course_id` | ❌    | Filter per kursus       |

**Response `200`:**
```json
{
  "reviews": [
    {
      "id": 1,
      "course_id": 1,
      "rating": 5,
      "comment": "Sangat sabar dan mudah dipahami.",
      "reviewer_name": "Andi Pratama",
      "course_name": "Matematika Olimpiade",
      "package_name": "Paket Reguler",
      "created_at": "2025-01-15T10:00:00Z"
    }
  ],
  "total": 1,
  "avg_rating": 4.8
}
```

---

### Parent

Base path: `/api/v1/parent`

**Auth:** ✅ Bearer Token dengan role `parent`

---

#### `GET /api/v1/parent/dashboard`

Ambil data dashboard orang tua: daftar anak dengan jadwal terdekat, statistik, dan tagihan mendatang.

**Response `200`:**
```json
{
  "parent": {
    "id": 1,
    "name": "Hendra Wijaya",
    "avatar": null
  },
  "children": [
    {
      "id": 2,
      "name": "ALDI WIJAYA",
      "education": "SD Kelas 4",
      "mentor_name": "Budi Santoso",
      "course_name": "Matematika Olimpiade",
      "progress": 37,
      "next_session": {
        "id": 4,
        "scheduled_at": "2025-01-20T15:00:00Z",
        "duration": 90,
        "status": "upcoming"
      }
    }
  ],
  "stats": {
    "total_sessions": 20,
    "completed_sessions": 15,
    "upcoming_sessions": 1,
    "total_spent": 1500000
  },
  "upcoming_payment": {
    "id": 1,
    "amount": 549000,
    "description": "Paket Reguler - Matematika Olimpiade",
    "due_date": "2025-01-25T00:00:00Z",
    "status": "unpaid"
  }
}
```

---

#### `GET /api/v1/parent/profile`

Ambil profil orang tua beserta daftar anak.

**Response `200`:**
```json
{
  "id": 1,
  "name": "Hendra Wijaya",
  "email": "hendra@example.com",
  "phone": "081234567890",
  "avatar": null,
  "address": "Karet Kuningan, Jakarta Selatan",
  "children": [
    { "id": 2, "name": "ALDI WIJAYA", "birth_date": "2011-05-10T00:00:00Z", "gender": "Laki-laki", "education": "SD Kelas 4" }
  ]
}
```

---

#### `PUT /api/v1/parent/profile`

Update profil orang tua. Semua field opsional.

**Request Body:**
```json
{
  "name": "Hendra Wijaya",
  "phone": "081234567890",
  "address": "Karet Kuningan, Jakarta Selatan"
}
```

**Response `200`:**
```json
{
  "message": "Profil berhasil diperbarui"
}
```

---

#### `GET /api/v1/parent/children`

Ambil daftar semua anak milik orang tua ini.

**Response `200`:**
```json
{
  "children": [
    {
      "id": 2,
      "parent_id": 1,
      "name": "ALDI WIJAYA",
      "birth_date": "2011-05-10T00:00:00Z",
      "gender": "Laki-laki",
      "education": "SD Kelas 4",
      "avatar_url": null
    }
  ],
  "total": 1
}
```

---

#### `POST /api/v1/parent/children`

Tambah data anak baru.

**Request Body:**
```json
{
  "name": "Aldi Wijaya",
  "birth_date": "2011-05-10",
  "gender": "Laki-laki",
  "education": "SD Kelas 4"
}
```

| Field        | Tipe   | Wajib | Keterangan                                                     |
|--------------|--------|-------|----------------------------------------------------------------|
| `name`       | string | ✅    | Nama anak (otomatis diubah ke UPPERCASE)                      |
| `birth_date` | string | ✅    | Format `YYYY-MM-DD`                                           |
| `gender`     | string | ✅    | `"Laki-laki"` atau `"Perempuan"`                              |
| `education`  | string | ❌    | Contoh: `"SD Kelas 4"`, `"SMP Kelas 7"`, `"Perguruan Tinggi"` |

**Response `201`:**
```json
{
  "message": "Data anak berhasil ditambahkan",
  "child": {
    "id": 3,
    "parent_id": 1,
    "name": "ALDI WIJAYA",
    "birth_date": "2011-05-10T00:00:00Z",
    "gender": "Laki-laki",
    "education": "SD Kelas 4"
  }
}
```

---

#### `PUT /api/v1/parent/children/:child_id`

Update data anak. Semua field opsional.

**Path Param:** `child_id`

**Request Body:**
```json
{
  "name": "Aldi Wijaya",
  "birth_date": "2011-05-10",
  "gender": "Laki-laki",
  "education": "SD Kelas 5"
}
```

**Response `200`:**
```json
{
  "message": "Data anak berhasil diperbarui",
  "child": { ... }
}
```

---

#### `DELETE /api/v1/parent/children/:child_id`

Hapus data anak. Tidak bisa dihapus jika masih ada kursus aktif.

**Path Param:** `child_id`

**Response `200`:**
```json
{
  "message": "Data anak berhasil dihapus"
}
```

**Response `400` — ada kursus aktif:**
```json
{
  "message": "Data anak tidak dapat dihapus karena masih ada kursus aktif"
}
```

---

#### `GET /api/v1/parent/children/:child_id/progress`

Lihat progress belajar anak: semua order aktif, skill progress, dan sesi terbaru.

**Path Param:** `child_id`

**Response `200`:**
```json
{
  "child": {
    "id": 2,
    "name": "ALDI WIJAYA",
    "birth_date": "2011-05-10T00:00:00Z",
    "education": "SD Kelas 4"
  },
  "progress": [
    {
      "id": 5,
      "course": { "title": "Matematika Olimpiade" },
      "package": { "name": "Paket Reguler" },
      "total_sessions": 8,
      "completed_sessions": 3,
      "remaining_sessions": 5,
      "status": "active",
      "skills": [
        { "skill_name": "Logika Dasar", "progress": 80 },
        { "skill_name": "Aljabar", "progress": 40 }
      ],
      "recent_sessions": [
        {
          "id": 3,
          "session_number": 3,
          "scheduled_at": "2025-01-17T09:00:00Z",
          "status": "completed",
          "topic": "Persamaan Kuadrat",
          "notes": "Sudah memahami dengan baik.",
          "stars": 5
        }
      ]
    }
  ]
}
```

---

#### `GET /api/v1/parent/orders`

Ambil riwayat semua order orang tua ini.

**Response `200`:**
```json
{
  "orders": [
    {
      "id": 5,
      "child": { "id": 2, "name": "ALDI WIJAYA" },
      "course": { "id": 1, "title": "Matematika Olimpiade" },
      "package": { "id": 2, "name": "Paket Reguler", "price": 549000 },
      "total_sessions": 8,
      "completed_sessions": 3,
      "remaining_sessions": 5,
      "total_price": 549000,
      "status": "active",
      "preferred_days": "1,3",
      "preferred_time": "09:00",
      "created_at": "2025-01-05T08:00:00Z"
    }
  ],
  "total": 1
}
```

---

#### `POST /api/v1/parent/orders`

Beli paket kursus untuk anak. Setelah order dibuat, invoice otomatis dibuat dengan due date 3 hari.

**Request Body:**
```json
{
  "child_id": 2,
  "course_id": 1,
  "package_id": 2,
  "preferred_days": "1,3",
  "preferred_time": "09:00",
  "notes": "Aldi lebih suka belajar pagi hari."
}
```

| Field            | Tipe   | Wajib | Keterangan                                           |
|------------------|--------|-------|------------------------------------------------------|
| `child_id`       | int    | ✅    | ID anak yang akan belajar                            |
| `course_id`      | int    | ✅    | ID kursus yang dipilih                               |
| `package_id`     | int    | ✅    | ID paket harga (harus milik course tersebut)         |
| `preferred_days` | string | ❌    | Hari pilihan dipisah koma: `"1,3,5"` (Senin,Rabu,Jumat) |
| `preferred_time` | string | ❌    | Jam pilihan format `"HH:MM"`, contoh: `"09:00"`      |
| `notes`          | string | ❌    | Catatan tambahan untuk mentor                        |

**Response `201`:**
```json
{
  "message": "Order berhasil dibuat",
  "order_id": 5,
  "invoice_id": 3
}
```

---

#### `GET /api/v1/parent/payments`

Ambil semua invoice/tagihan orang tua ini.

**Response `200`:**
```json
{
  "payments": [
    {
      "id": 3,
      "order_id": 5,
      "amount": 549000,
      "description": "Paket Reguler - Matematika Olimpiade",
      "period": "January 2025",
      "status": "unpaid",
      "due_date": "2025-01-08T00:00:00Z",
      "paid_at": null,
      "method": "",
      "proof_url": null,
      "order": {
        "child": { "name": "ALDI WIJAYA" },
        "course": { "title": "Matematika Olimpiade" }
      }
    }
  ],
  "total": 1
}
```

> `status` bisa: `unpaid`, `paid`, `expired`

---

#### `POST /api/v1/parent/payments/:invoice_id`

Konfirmasi pembayaran untuk invoice tertentu.

**Path Param:** `invoice_id`

**Request Body:**
```json
{
  "method": "bank_transfer",
  "proof_url": "https://storage.example.com/bukti-transfer.jpg"
}
```

| Field       | Tipe   | Wajib | Keterangan                                          |
|-------------|--------|-------|-----------------------------------------------------|
| `method`    | string | ✅    | `"bank_transfer"`, `"e_wallet"`, atau `"virtual_account"` |
| `proof_url` | string | ❌    | URL foto bukti pembayaran                           |

> Setelah pembayaran dikonfirmasi, status order otomatis berubah menjadi `active`.

**Response `200`:**
```json
{
  "message": "Pembayaran berhasil dikonfirmasi"
}
```

**Response `400` — invoice sudah dibayar:**
```json
{
  "message": "Invoice sudah dibayar atau kadaluarsa"
}
```

---

#### `GET /api/v1/parent/schedule`

Ambil jadwal sesi mendatang untuk semua anak.

**Response `200`:**
```json
{
  "schedule": [
    {
      "id": 4,
      "child_id": 2,
      "scheduled_at": "2025-01-20T09:00:00Z",
      "duration": 90,
      "status": "upcoming",
      "meet_link": "https://meet.google.com/xxx",
      "session_number": 4,
      "order": {
        "child": { "id": 2, "name": "ALDI WIJAYA" },
        "course": { "id": 1, "title": "Matematika Olimpiade" }
      }
    }
  ],
  "total": 1
}
```

---

#### `POST /api/v1/parent/reviews`

Kirim ulasan untuk sebuah order. Satu order hanya bisa diulas **satu kali**.

**Request Body:**
```json
{
  "order_id": 5,
  "rating": 5,
  "comment": "Mentor sangat sabar dan metodenya mudah dipahami anak saya."
}
```

| Field      | Tipe   | Wajib | Keterangan                         |
|------------|--------|-------|------------------------------------|
| `order_id` | int    | ✅    | ID order yang ingin diulas         |
| `rating`   | int    | ✅    | Nilai bintang: `1`–`5`             |
| `comment`  | string | ❌    | Komentar ulasan                    |

> Setelah review dikirim, rata-rata rating di Course dan MentorProfile otomatis diperbarui secara async.

**Response `201`:**
```json
{
  "message": "Ulasan berhasil dikirim",
  "review": {
    "id": 7,
    "order_id": 5,
    "rating": 5,
    "comment": "Mentor sangat sabar dan metodenya mudah dipahami anak saya.",
    "reviewer_name": "ALDI WIJAYA",
    "course_name": "Matematika Olimpiade",
    "package_name": "Paket Reguler",
    "created_at": "2025-01-20T11:00:00Z"
  }
}
```

**Response `409` — sudah pernah review:**
```json
{
  "message": "Kamu sudah memberikan ulasan untuk order ini"
}
```

---

#### `GET /api/v1/parent/reviews`

Ambil semua ulasan yang pernah dikirim orang tua ini.

**Response `200`:**
```json
{
  "reviews": [
    {
      "id": 7,
      "order_id": 5,
      "rating": 5,
      "comment": "Mentor sangat sabar.",
      "reviewer_name": "ALDI WIJAYA",
      "course_name": "Matematika Olimpiade",
      "package_name": "Paket Reguler",
      "created_at": "2025-01-20T11:00:00Z"
    }
  ],
  "total": 1
}
```

---

## Catatan Tambahan

### Password Requirements
- Minimal **8 karakter**
- Harus mengandung minimal **1 huruf** dan **1 angka**

### Rate Limits
| Aksi                        | Limit          |
|-----------------------------|----------------|
| Kirim / resend OTP          | 1x per 60 detik |
| Request reset password      | 1x per 60 detik |
| Percobaan verify OTP        | Maks 5x per OTP |

### Token Expiry
| Token             | Masa Berlaku |
|-------------------|--------------|
| JWT (login)       | 24 jam       |
| OTP verifikasi    | 5 menit      |
| Reset password    | 15 menit     |

### Preferred Days (Order)
Format string angka dipisah koma mengikuti `time.Weekday` Go:

| Angka | Hari   |
|-------|--------|
| 0     | Minggu |
| 1     | Senin  |
| 2     | Selasa |
| 3     | Rabu   |
| 4     | Kamis  |
| 5     | Jumat  |
| 6     | Sabtu  |

Contoh: `"1,3,5"` = Senin, Rabu, Jumat