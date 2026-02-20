# 🚀 Potensia Backend API

![Go Version](https://img.shields.io/badge/Go-1.20+-00ADD8?logo=go)
![Framework](https://img.shields.io/badge/Framework-Gin-blue)
![ORM](https://img.shields.io/badge/ORM-GORM-green)
![Database](https://img.shields.io/badge/Database-MySQL-orange?logo=mysql)
![Auth](https://img.shields.io/badge/Auth-JWT-black)
![License](https://img.shields.io/badge/License-MIT-yellow)

Potensia Backend adalah RESTful API yang digunakan sebagai server-side untuk aplikasi mobile **Potensia**.  
Backend ini menangani autentikasi, manajemen user, serta komunikasi data antara mobile app dan database.

---

## 🛠️ Tech Stack

- Golang (Go)
- Gin Gonic
- GORM
- MySQL
- JWT (JSON Web Token)

---

## 📂 Project Structure

```
├── controllers
├── models
├── routes
├── middleware
├── config
├── utils
└── main.go
```

---

## ⚙️ Installation & Setup

### 1. Clone Repository
```
git clone https://github.com/username/potensia-backend.git
cd potensia-backend
```

### 2. Install Dependencies
```
go mod tidy
```

### 3. Setup Environment
Create `.env` file:
```
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASS=
DB_NAME=potensia

JWT_SECRET=your_secret_key
```

### 4. Run Server
```
go run main.go
```

Server runs at:
```
http://localhost:8080
```

---

## 🔐 API Endpoints

### Register
POST /api/register

### Login
POST /api/login

---

## 🧠 Features

- JWT Authentication
- Role-based login validation
- Password hashing (bcrypt)
- Clean architecture

---

## 🔒 Security

- Hashed password (bcrypt)
- Token expiration
- Protected routes with middleware

---

## 📄 License

MIT License
