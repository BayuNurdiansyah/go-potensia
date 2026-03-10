# Go Potensia — Backend API

REST API for the **Potensia** tutoring platform, connecting parents with mentors for children's learning.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.21 |
| Framework | Gin |
| ORM | GORM |
| Database | PostgreSQL |
| Auth | JWT (HS256) |
| Password | bcrypt |
| Email | Brevo |
| Config | godotenv |

---

## Installation

**1. Clone & install dependencies**
```bash
git clone https://github.com/your-org/go-potensia.git
cd go-potensia
go mod tidy
```

**2. Configure environment**
```bash
cp .env.example .env
# Edit .env with your values
```

Required `.env` variables:
```env
DATABASE_URL=postgres://user:password@localhost:5432/potensia?sslmode=disable
JWT_SECRET=your-super-secret-key
PORT=8080

SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your@email.com
SMTP_PASS=your-app-password
SMTP_FROM=Potensia <your@email.com>
```

**3. Run the server**
```bash
go run main.go
```

**4. Seed the database** *(optional — fills all tables with linked sample data)*
```bash
go run main.go --seed
```

---

## Quick API Example

**Register**
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Siti Aminah",
    "email": "siti@email.com",
    "password": "Password123!",
    "role": "parent"
  }'
```

**Login**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "siti@email.com", "password": "Password123!"}'
```

**Use the token** — add `Authorization: Bearer <token>` to protected routes:
```bash
curl http://localhost:8080/api/v1/parent/dashboard \
  -H "Authorization: Bearer eyJhbGci..."
```

**Search mentors** *(public)*
```bash
curl "http://localhost:8080/api/v1/mentors?q=matematika&category=Matematika"
```

---

## Seeded Accounts

After running `--seed`, these accounts are ready to use (password: `Password123!`):

| Role | Email |
|---|---|
| Mentor | budi.mentor@potensia.id |
| Mentor | aisyah.mentor@potensia.id |
| Mentor | rizky.mentor@potensia.id |
| Parent | siti.parent@potensia.id |
| Parent | ahmad.parent@potensia.id |
| Parent | dewi.parent@potensia.id |

---

## Health Check

```bash
curl http://localhost:8080/health
# {"status":"ok","service":"go-potensia"}
```