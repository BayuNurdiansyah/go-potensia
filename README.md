# Go Potensia — Backend API

Backend untuk aplikasi bimbel **Potensia**, dibangun dengan Go + Gin + GORM + PostgreSQL.

---

## 🗂️ Struktur Project

```
go-potensia/
├── config/           # Koneksi database
├── controllers/      # Handler request (auth, profile)
├── middlewares/      # JWT auth middleware
├── models/           # GORM models
├── routes/           # Setup routing
├── utils/            # JWT, OTP, Email, Validator
├── main.go
├── .env.example
└── go.mod
```

---

## ⚙️ Setup

1. Copy `.env.example` ke `.env` dan isi nilainya:
```bash
cp .env.example .env
```

2. Install dependencies:
```bash
go mod tidy
```

3. Run server:
```bash
go run main.go
```

---

## 📬 Environment Variables

| Variable             | Keterangan                                      |
|----------------------|-------------------------------------------------|
| `DATABASE_URL`       | PostgreSQL connection string                    |
| `JWT_SECRET`         | Secret key untuk JWT (min 32 karakter)          |
| `PORT`               | Port server (default: 8080)                     |
| `APP_NAME`           | Nama aplikasi (tampil di email)                 |
| `APP_URL`            | Base URL frontend (untuk link reset password)   |
| `BREVO_API_KEY`      | API key dari [Brevo](https://brevo.com)         |
| `BREVO_SENDER_EMAIL` | Email pengirim yang terverifikasi di Brevo      |
| `BREVO_SENDER_NAME`  | Nama pengirim (default: APP_NAME)               |

---

## 🔌 API Endpoints

### Auth

| Method | Endpoint                          | Auth | Keterangan                            |
|--------|-----------------------------------|------|---------------------------------------|
| POST   | `/api/v1/auth/register`           | ❌   | Daftar akun baru                      |
| POST   | `/api/v1/auth/login`              | ❌   | Login                                 |
| POST   | `/api/v1/auth/verify-otp`         | ❌   | Verifikasi OTP setelah register       |
| POST   | `/api/v1/auth/resend-otp`         | ❌   | Kirim ulang OTP                       |
| POST   | `/api/v1/auth/forgot-password`    | ❌   | Request link reset password via email |
| GET    | `/api/v1/auth/verify-reset-token` | ❌   | Validasi token reset (untuk frontend) |
| POST   | `/api/v1/auth/reset-password`     | ❌   | Set password baru                     |

### Protected

| Method | Endpoint            | Auth | Keterangan           |
|--------|---------------------|------|----------------------|
| GET    | `/api/v1/profile`   | ✅   | Lihat profil sendiri |

---

## 🔐 Forgot Password Flow

```
1. POST /auth/forgot-password     { "email": "user@example.com" }
      → Server kirim email dengan link: APP_URL/reset-password?token=<token>
      → Token berlaku 15 menit, sekali pakai

2. GET /auth/verify-reset-token?token=<token>
      → Frontend panggil ini untuk validasi token sebelum tampilkan form

3. POST /auth/reset-password      { "token": "...", "new_password": "...", "confirm_password": "..." }
      → Password diupdate, token di-invalidate
```

---

## 🧪 Contoh Request

### Register
```json
POST /api/v1/auth/register
{
  "name": "Budi Santoso",
  "email": "budi@example.com",
  "password": "budi1234",
  "role": "student"
}
```

### Login
```json
POST /api/v1/auth/login
{
  "email": "budi@example.com",
  "password": "budi1234"
}
```

### Forgot Password
```json
POST /api/v1/auth/forgot-password
{ "email": "budi@example.com" }
```

### Reset Password
```json
POST /api/v1/auth/reset-password
{
  "token": "abc123...",
  "new_password": "newpass99",
  "confirm_password": "newpass99"
}
```