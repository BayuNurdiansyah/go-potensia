# Go Potensia — Backend API

Backend lengkap untuk aplikasi bimbel **Potensia** (Golang + Gin + GORM + PostgreSQL).

---

## 🗂️ Struktur Project

```
go-potensia/
├── config/
│   └── database.go         # Koneksi & auto-migrate
├── controllers/
│   ├── auth.go             # Register, Login, OTP, ForgotPassword, dll
│   ├── mentor.go           # Dashboard, profile, students, schedule, reviews
│   ├── course.go           # CRUD kursus & search mentor publik
│   ├── parent.go           # Dashboard, children, orders, payments, progress
│   └── notification.go     # Notifikasi in-app
├── middlewares/
│   └── auth.go             # JWT auth + role guard
├── models/
│   └── models.go           # Semua struct GORM
├── routes/
│   └── routes.go           # Routing lengkap
├── utils/
│   ├── jwt.go
│   ├── otp.go
│   ├── email.go            # Brevo API
│   ├── validator.go
│   └── response.go         # Helper response standar
├── main.go
└── .env.example
```

---

## 📊 Skema Database (17 Tabel)

| Tabel                  | Fungsi                                                    |
|------------------------|-----------------------------------------------------------|
| `users`                | Akun login (mentor / parent / admin)                      |
| `mentor_profiles`      | Data lengkap mentor (1:1 dengan users)                    |
| `mentor_certificates`  | Sertifikat mentor                                         |
| `mentor_achievements`  | Prestasi mentor                                           |
| `mentor_galleries`     | Foto galeri mentor                                        |
| `mentor_educations`    | Riwayat pendidikan mentor                                 |
| `parent_profiles`      | Data tambahan orang tua (1:1 dengan users)                |
| `children`             | Data anak dari parent                                     |
| `courses`              | Kursus yang dibuat mentor                                 |
| `course_competencies`  | Kompetensi/materi per kursus                              |
| `course_packages`      | Paket harga (Starter/Reguler/Intensif)                    |
| `orders`               | Transaksi pembelian paket kursus                          |
| `sessions`             | Satu pertemuan belajar                                    |
| `invoices`             | Tagihan pembayaran per order                              |
| `reviews`              | Ulasan parent terhadap mentor/kursus                      |
| `notifications`        | Notifikasi in-app                                         |
| `skill_progresses`     | Progress per skill per anak per kursus                    |

---

## ⚙️ Setup

```bash
cp .env.example .env
# isi .env dengan konfigurasi kamu

go mod tidy
go run main.go
```

---

## 🔌 API Endpoints Lengkap

### Auth (Public)
| Method | Endpoint                              | Keterangan                      |
|--------|---------------------------------------|---------------------------------|
| POST   | `/api/auth/register`               | Daftar akun (mentor/parent)     |
| POST   | `/api/auth/login`                  | Login                           |
| POST   | `/api/auth/verify-otp`             | Verifikasi OTP register         |
| POST   | `/api/auth/resend-otp`             | Kirim ulang OTP                 |
| POST   | `/api/auth/forgot-password`        | Request link reset password     |
| GET    | `/api/auth/verify-reset-token`     | Validasi token reset            |
| POST   | `/api/auth/reset-password`         | Set password baru               |

### Public
| Method | Endpoint                              | Keterangan                      |
|--------|---------------------------------------|---------------------------------|
| GET    | `/api/mentors`                     | Cari mentor (filter + search)   |
| GET    | `/api/mentors/:mentor_id`          | Detail profil publik mentor     |

### Protected (All Roles) — Bearer Token required
| Method | Endpoint                              | Keterangan                      |
|--------|---------------------------------------|---------------------------------|
| POST   | `/api/auth/change-password`        | Ubah password                   |
| POST   | `/api/auth/delete-account`         | Hapus akun                      |
| GET    | `/api/notifications`               | Daftar notifikasi               |
| PUT    | `/api/notifications/:id/read`      | Tandai dibaca (`all` = semua)   |
| GET    | `/api/notifications/preferences`   | Preferensi notifikasi           |
| PUT    | `/api/notifications/preferences`   | Update preferensi notifikasi    |

### Mentor — Role: mentor
| Method | Endpoint                              | Keterangan                      |
|--------|---------------------------------------|---------------------------------|
| GET    | `/api/mentor/dashboard`            | Dashboard statistik             |
| GET    | `/api/mentor/profile`              | Profil mentor                   |
| PUT    | `/api/mentor/profile`              | Update profil mentor            |
| GET    | `/api/mentor/students`             | Daftar siswa                    |
| GET    | `/api/mentor/students/:order_id`   | Detail siswa + progress + sesi  |
| GET    | `/api/mentor/schedule`             | Jadwal mengajar                 |
| PUT    | `/api/mentor/sessions/:id`         | Update sesi (topik/catatan/bintang/status) |
| GET    | `/api/mentor/courses`              | Daftar kursus                   |
| POST   | `/api/mentor/courses`              | Buat kursus baru                |
| GET    | `/api/mentor/courses/:id`          | Detail kursus                   |
| PUT    | `/api/mentor/courses/:id`          | Update kursus                   |
| DELETE | `/api/mentor/courses/:id`          | Hapus kursus                    |
| GET    | `/api/mentor/reviews`              | Ulasan masuk                    |

### Parent — Role: parent
| Method | Endpoint                                    | Keterangan                    |
|--------|---------------------------------------------|-------------------------------|
| GET    | `/api/parent/dashboard`                  | Dashboard                     |
| GET    | `/api/parent/profile`                    | Profil orang tua              |
| PUT    | `/api/parent/profile`                    | Update profil                 |
| GET    | `/api/parent/children`                   | Daftar anak                   |
| POST   | `/api/parent/children`                   | Tambah anak                   |
| PUT    | `/api/parent/children/:id`               | Update data anak              |
| DELETE | `/api/parent/children/:id`               | Hapus data anak               |
| GET    | `/api/parent/children/:id/progress`      | Progress belajar anak         |
| GET    | `/api/parent/orders`                     | Daftar order                  |
| POST   | `/api/parent/orders`                     | Beli paket kursus             |
| GET    | `/api/parent/payments`                   | Daftar invoice/tagihan        |
| POST   | `/api/parent/payments/:invoice_id`       | Konfirmasi pembayaran         |
| GET    | `/api/parent/schedule`                   | Jadwal sesi anak              |
| POST   | `/api/parent/reviews`                    | Kirim ulasan                  |
| GET    | `/api/parent/reviews`                    | Riwayat ulasan                |

---

## 🔐 Auth Flow

```
Register → OTP (email) → Verify OTP → auto Login (JWT)

Forgot Password:
  POST /forgot-password { email }
  → email: APP_URL/reset-password?token=<token> (15 menit, sekali pakai)
  GET  /verify-reset-token?token=<token>
  POST /reset-password { token, new_password, confirm_password }
```

---

## 📋 Query Params

**GET /api/mentors**
- `province` — filter provinsi
- `regency` — filter kabupaten/kota
- `district` — filter kecamatan
- `category` — filter kategori kursus (MATEMATIKA, dll)
- `search` — cari nama mentor
- `page` (default: 1)
- `limit` (default: 20)

**GET /api/mentor/students**
- `status` — `active` | `completed`
- `search` — nama anak / kursus

**GET /api/mentor/schedule**
- `date` — filter tanggal (`YYYY-MM-DD`)

**GET /api/mentor/reviews**
- `course_id` — filter per kursus