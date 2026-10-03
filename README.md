# Business Data API

API Gateway multi-tenant (Maxtop, Pandora, Toplink) dengan dual database (PostgreSQL & MSSQL).

---

## Prasyarat

- Go 1.23+
- PostgreSQL 14+ (ekstensi `pg_trgm`)
- MSSQL (opsional)

---

## 1. Clone & Install

```bash
git clone <repo-url>
cd business-data-api
go mod download
```

---

## 2. Setup Environment

Salin template lalu isi nilai development:

```bash
cp .env.example .env
# edit .env
# WAJIB: APP_ENV=development (supaya keygen/jwtgen jalan)
```

---

## 3. Generate Kunci (Setup)

```bash
# API key → simpan ke api_keys.json
go run ./cmd/keygen

# JWT secret (64 hex chars)
go run ./cmd/jwtgen
```

> Kedua command **tolak jalan** kalau `APP_ENV=production`.

---

## 4. Development

Jalankan (hot reload pakai air):

```bash
go install github.com/air-verse/air@latest
air
# atau
make dev
```

Server: `http://localhost:8080`

Swagger UI (hanya development): `http://localhost:8080/swagger/`

---

## 5. Build

```bash
# Binary ke bin/
make build
# Windows: bin/api.exe
# Linux: bin/api
```

---

## 6. Deploy Production (Linux + PM2)

### 1. Siapkan Server

Copy binary & config ke server:

```bash
scp bin/api user@server:/opt/business-data-api/
scp api_keys.json user@server:/opt/business-data-api/
scp ecosystem.config.cjs user@server:/opt/business-data-api/
```

Atau build di server (kalau clone di server):

```bash
cd business-data-api
make build
```

### 2. Konfigurasi Production

Salin `.env.example` ke `.env` lalu isi nilai production:

```bash
cp .env.example .env
# edit .env
```

**Wajib diubah untuk production:**
- `APP_ENV=production`
- `POSTGRES_*_URL` → `sslmode=require`
- `MSSQL_*_URL` → `encrypt=true&trustServerCertificate=true`
- `JWT_SECRET` → generate via `go run ./cmd/jwtgen`
- `API_KEYS_PATH` → path absolut (contoh: `/opt/business-data-api/api_keys.json`)

Generate `api_keys.json` di dev, copy ke server:

```bash
go run ./cmd/keygen  # di dev
scp api_keys.json user@server:/opt/business-data-api/
```

Set permission:

```bash
chmod 600 .env api_keys.json
```

### 3. PM2

Konfigurasi `ecosystem.config.cjs` yang sudah ada di repo (sesuaikan path kalau perlu), lalu:

```bash
npm install -g pm2
pm2 start ecosystem.config.cjs
pm2 save
pm2 startup  # jalankan perintah yang tercetak
```

### 4. Verifikasi

```bash
pm2 logs business-data-api
# Cek shutdown log: "Server berhasil dimatikan dengan aman"
curl -H "X-API-KEY: <key>" http://localhost:8080/health
```

---

## Integrasi FE (Pandora Server)

| Variabel FE | Nilai |
|---|---|
| `PRIVATE_API_BASE_URL` | `http://<host-backend>:8080` |
| `PRIVATE_API_KEY` | Sama dengan salah satu key di `api_keys.json` backend |
| `COOKIE_SECURE` | `false` (akses HTTP) |

**Middleware FE wajib:**
1. Baca IP client → kirim header `X-Forwarded-For` ke backend
2. Forward `Set-Cookie` dari respons backend ke browser

Tanpa itu: log IP backend selalu `::1`, login cookie tidak jalan.

---

## Struktur Project

```
├── cmd/
│   ├── api/          # Entry point server
│   ├── keygen/       # Generate API key
│   └── jwtgen/       # Generate JWT secret
├── internal/
│   ├── config/       # Load & validasi .env
│   ├── handler/      # HTTP handler
│   ├── middleware/   # Auth, rate limit, network guard
│   ├── repository/   # DB query (Postgres & MSSQL)
│   └── usecase/      # Business logic
├── pkg/
│   ├── database/     # Connection pool
│   ├── jwt/          # JWT token
│   ├── logger/       # Structured logging
│   └── response/     # Standard response
├── docs/
│   ├── swagger.yaml  # OpenAPI spec
│   └── optimize_indexes.md  # DDL index database
├── bin/              # Build output (gitignore)
├── tmp/              # Air temp (gitignore)
├── .env              # Config (gitignore)
├── api_keys.json     # API keys (gitignore)
├── ecosystem.config.cjs  # PM2 config
├── go.mod
├── go.sum
└── Makefile
```

---

## Testing

```bash
go test ./...
```

---

## Index Database

Lihat `docs/optimize_indexes.md` untuk DDL lengkap.

Index bisection (`idx_*_kode_tgl` / `IX_*_kode_tgl`) **wajib** — dipakai `cutStartPG` / `cutEndPG`. Jangan drop tanpa ubah kode repository.

---

## Catatan Penting

- `APP_ENV=production` → Swagger UI **tidak terdaftar** (404)
- `GLOBAL_LOCAL_ONLY=true` + `IsLocal()` hanya loopback → endpoint data hanya bisa diakses dari server yang sama (atau via SSH tunnel)
- `/health` butuh header `X-API-KEY`, rate limit 50 req/detik
- MSSQL butuh `encrypt=true&trustServerCertificate=true` di connection string
- Postgres butuh `sslmode=require` + ekstensi `pg_trgm` untuk ILIKE
- `COOKIE_SECURE=false` wajib karena akses via HTTP (bukan HTTPS)