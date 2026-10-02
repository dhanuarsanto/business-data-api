# Deployment & Operasional

Catatan khusus yang perlu diketahui tim saat deploy/operasional. Dokumentasi index ada terpisah di `optimize_indexes.md`.

Semua nilai di bawah diambil dari kode (`internal/config/config.go`, `cmd/api/main.go`, `cmd/api/modules.go`, `internal/handler/http/router.go`, `internal/handler/http/auth_handler.go`, `internal/middleware/`, `pkg/jwt/jwt.go`, `pkg/logger/logger.go`, `pkg/response/response.go`) dan `.env.example`.

## Prasyarat start

Server **menolak start** (`os.Exit(1)`) kalau salah satu hal berikut tidak terpenuhi.

| Prasyarat | Sumber | Gejala kalau gagal |
|---|---|---|
| `JWT_SECRET` diisi dan panjangnya minimal 32 byte | `config.go:61` (`required`), `jwt.MinSecretLength` | exit + pesan "JWT_SECRET terlalu pendek" |
| `POSTGRES_MAXTOP_URL` | `config.go:28` | exit, pesan menyebut nama variabel |
| `MSSQL_MAXTOP_URL` | `config.go:29` | idem |
| `POSTGRES_PANDORA_URL` | `config.go:31` | idem |
| `MSSQL_PANDORA_URL` | `config.go:32` | idem |
| `POSTGRES_TOPLINK_URL` | `config.go:34` | idem |
| `MSSQL_TOPLINK_URL` | `config.go:35` | idem |
| `api_keys.json` terbaca di `API_KEYS_PATH` | `apikey.go:31` | **tidak exit**, hanya warning, tapi semua endpoint membalas 401 |
| keenam koneksi database berhasil dibuat | `main.go:38-72` | exit, menyebut tenant yang gagal |

Tiga tenant: `maxtop`, `pandora`, `toplink`. Tiap tenant punya pasangan pool Postgres dan MSSQL, jadi ada enam koneksi yang dibuka saat start. Satu gagal, server mati.

`API_KEYS_PATH` default-nya `api_keys.json` dan sifatnya **relatif terhadap working directory**. Kalau service dijalankan dari `/`, berkas tidak ditemukan. Pakai path absolut untuk deployment non-dev.

`api_keys.json` dibaca ulang tiap 60 detik (`apikey.go:23`), jadi menambah atau mencabut API key tidak perlu restart. Kalau reload gagal, daftar lama dipertahankan.

Generate daftar key:

```bash
go run cmd/keygen/main.go "Nama Developer"
```

## Gate produksi: TRUSTED_PROXIES wajib diisi atau ALLOW_DIRECT_CLIENTS

`cmd/api/main.go:79-85`. Bila `TRUSTED_PROXIES` kosong **dan** ketiga kondisi ini terpenuhi:

- `APP_ENV=production`
- `GLOBAL_LOCAL_ONLY=true`
- `ALLOW_DIRECT_CLIENTS=false`

maka server **`os.Exit(1)` dengan pesan error**, bukan sekadar warning. Ini sengaja: konfigurasi itu berarti API tidak tahu apakah kliennya nyambung langsung atau lewat proxy, sehingga semua klien akan terlihat dari satu IP dan rate limit jadi satu ember bersama.

Isi `.env` sesuai deploy kamu:

| Situasi | Yang diisi |
|---|---|
| API di belakang reverse-proxy (nginx/IIS/LB) | `TRUSTED_PROXIES` dengan IP/CIDR proxy, `ALLOW_DIRECT_CLIENTS` boleh `false` |
| Klien connect langsung ke API | `ALLOW_DIRECT_CLIENTS=true` |

Di luar produksi, atau saat `GLOBAL_LOCAL_ONLY=false`, atau `ALLOW_DIRECT_CLIENTS=true`, `TRUSTED_PROXIES` kosong hanya memunculkan warning (`main.go:84`).

## TRUSTED_PROXIES dan cara header dibaca

`TRUSTED_PROXIES` berisi daftar IP/CIDR proxy, dipisah koma. Contoh: `TRUSTED_PROXIES=127.0.0.1,10.0.0.0/8`. Nilai tidak valid membuat server exit (`main.go:74-78`).

Kosong berarti anti-spoof: `RemoteAddr` dipakai langsung, `X-Forwarded-For` dan `X-Real-IP` diabaikan. Rate limit dan `NetworkRoleGuard` berbagi resolver yang sama (`internal/middleware/network_guard.go`), jadi konsekuensi `TRUSTED_PROXIES` bukan cuma rate limit — dia juga menentukan `IsLocal` yang dipakai untuk pembatasan jaringan.

**Presedensi XFF** (`network_guard.go:39-54`) — urutan bacaannya:

1. Kalau `RemoteAddr` tidak ada di daftar proxy, langsung dipakai. Header diabaikan.
2. Kalau `RemoteAddr` ada di daftar, `X-Forwarded-For` dipindai **dari kanan ke kiri**, dan berhenti di IP pertama yang **tidak** ada di daftar proxy. IP itu yang dipakai.
3. Kalau seluruh entri XFF ada di daftar proxy (atau header kosong), baru dipakai `X-Real-IP`.
4. Kalau `X-Real-IP` juga kosong atau tak valid, jatuh ke `RemoteAddr`.

Aturan ini yang membuat aman dari spoofing: penyerang tak bisa menyisipkan IP palsu di ujung kiri `X-Forwarded-For`, karena pemindaian selalu dimulai dari ujung kanan. Kalau urutan ini dibalik, siapa pun bisa mengaku berasal dari jaringan internal.

### IPv6 harus ditulis eksplisit

`netip.Prefix.Contains()` **tidak pernah mencocokkan antar keluarga alamat**: prefix IPv4 tidak akan cocok dengan alamat IPv6, dan sebaliknya. Jadi `TRUSTED_PROXIES=127.0.0.1` saja **tidak** menutup loopback IPv6.

Gejalanya mudah terlihat di log: browser yang membuka `http://localhost:8080` sering memilih IPv6 dulu, jadi `Audit Log` mencatat `ip=::1`. Itu IP klien yang benar, bukan bug. Tapi kalau ada proxy di host yang sama, `::1` yang tidak terdaftar berarti `X-Forwarded-For` diabaikan, `clientAddr` mengembalikan IP proxy, lalu `IsLocal()` bernilai `true` karena loopback. Akibatnya semua klien dari luar jaringan lolos pembatasan jaringan, termasuk peran `sa` yang seharusnya ditolak.

Tuliskan keduanya:

```
TRUSTED_PROXIES=127.0.0.1,::1,10.0.0.0/8,fe80::/10
```

`::1` tanpa CIDR otomatis jadi `::1/128` (`config.go:99`). Alamat IPv4-mapped seperti `::ffff:10.0.0.1` otomatis dilepas ke bentuk IPv4-nya (`network_guard.go:71`), jadi cukup tulis `10.0.0.0/8`.

### ALLOW_DIRECT_CLIENTS hanya sah tanpa proxy di host yang sama

`ALLOW_DIRECT_CLIENTS=true` berarti: "klien connect langsung, tidak ada proxy". Kalau ternyata ada reverse-proxy di host yang sama dan `TRUSTED_PROXIES` tetap kosong, setiap request masuk dengan IP loopback, sehingga `IsLocal()` selalu `true` dan `GLOBAL_LOCAL_ONLY` beserta aturan `sa`-wajib-lokal kehilangan makna.

Kalau ada proxy di host yang sama, **isi `TRUSTED_PROXIES`** dan biarkan `ALLOW_DIRECT_CLIENTS=false`.

## Pembatasan jaringan: GLOBAL_LOCAL_ONLY

- `GLOBAL_LOCAL_ONLY=true` (default): semua request harus datang dari loopback atau jaringan privat, apa pun perannya. Non-lokal dapat 403.
- `GLOBAL_LOCAL_ONLY=false`: pembatasan turun ke level peran lewat `networkMatrix` (`cmd/api/modules.go:24-27`). Nilai `true` berarti peran itu **wajib lokal**, `false` berarti bebas.

Matriks peran saat ini:

| Peran | Wajib lokal |
|---|---|
| `sa` | ya |
| `*` (peran lain) | tidak |

Dampaknya di dua tempat: `NetworkRoleGuard` pada setiap route (`router.go:85, 90`) dan pemeriksaan login di `auth_handler.go:49`. Pengguna `sa` yang login dari luar jaringan tetap melewati verifikasi password, lalu mendapat 403.

## POSTGRES_WRITE_ENABLED

`POSTGRES_WRITE_ENABLED` default `false`. Selama `false`, `PostgresWriteGuard` (`internal/middleware/postgres_write_guard.go`) memblokir `POST`, `PUT`, dan `DELETE` ke Postgres dengan 403.

Jadi di environment bawaan, seluruh endpoint tulis mati. Endpointnya: `POST/PUT /inbox`, `POST/PUT /outbox`, `POST/PUT /auth/users`. Endpoint baca tidak terpengaruh.

Perhatikan selectivity header: guard hanya menolak kalau `X-DB-Source` kosong (default Postgres) atau bernilai `postgres`. Request dengan `X-DB-Source: mssql` tidak lewat guard ini, jadi proteksi write Postgres tidak berlaku untuk jalur MSSQL.

Set `true` hanya di environment yang memang butuh endpoint tulis, dan tetap jaga `GLOBAL_LOCAL_ONLY` — dua-duanya lapisan berbeda.

## Autentikasi berlapis

Setiap endpoint, termasuk `/health`, melewati dua lapis sebelum handler:

**Lapis 1 — API key.** Header `X-API-KEY` wajib, nilai harus ada di `api_keys.json` (`apikey.go:81-101`). Kosong → 401 "Header X-API-KEY kosong", tidak terdaftar → 401 "API Key tidak terdaftar". Nama developer dari key tersebut ikut masuk ke setiap baris log sebagai `developer`.

**Lapis 2 — token dan peran.** Untuk route terproteksi, `Authorization: Bearer <token>` wajib dan harus valid (`jwt_rbac.go:20`). Klaim `tenant` pada token harus sama dengan `{tenant}` di path, kalau tidak 403.

RBAC per endpoint (`cmd/api/modules.go:29-36`):

| Izin | Peran yang boleh |
|---|---|
| `ManageUsers` | `sa` |
| `ReadResellerDropdown` | `sa`, `op`, `opout` |
| `ReadInbox` | `sa`, `op`, `opout` |
| `WriteInbox` | `sa` |
| `ReadOutbox` | `sa`, `op`, `opout` |
| `WriteOutbox` | `sa` |

**Catatan `/health` untuk load balancer.** Endpoint `/health` ada di `protectedApiKey` (`router.go:71-84, 93`), artinya tetap butuh `X-API-KEY`. Probe tanpa header dapat 401 dan akan menandai service sebagai unhealthy. Konfigurasikan probe dengan header key, atau pakai pemeriksa lain.

## COOKIE_SECURE

- `true` (default): cookie `access_token` hanya dikirim lewat HTTPS.
- Set `false` bila API diakses via plain HTTP (LAN/dev).
- Kalau diakses via HTTPS, biarkan `true`.

Login juga mengembalikan token di body response (`auth_handler.go:69-73`), jadi akses HTTP tidak menghalangi auth selama klien memakai token dari body, bukan cookie. Cookie-nya sendiri `HttpOnly` dan `SameSite=Strict`.

Umur cookie mengikuti `JWT_TOKEN_DURATION` (`auth_handler.go:128`), jadi satu env itu mengatur dua hal sekaligus.

## Model autentikasi (keputusan arsitektur)

JWT stateless, tidak terikat IP, device, atau browser. Siapa pun yang memegang token punya akses penuh sampai `exp`.

Pola token (`pkg/jwt/jwt.go:53-62`):

| Klaim | Status |
|---|---|
| `exp` | **wajib**, ditolak kalau tidak ada (`WithExpirationRequired`) |
| `iss` | divalidasi manual, harus sama dengan `JWT_ISSUER` (`jwt.go:87-95`) |
| `iat` | divalidasi (`WithIssuedAt`), `iat` di masa depan ditolak |
| `nbf` | **dikirim tapi tidak divalidasi** — tidak ada `WithNotBefore()` dan `jwt.go` tidak pernah membaca `nbf` |
| `user_id`, `username`, `rules`, `tenant` | dipakai untuk RBAC dan pencocokan tenant |

Leeway 30 detik (`tokenLeeway`, `jwt.go:13`) untuk semua pemeriksaan waktu.

Token ditolak juga bila metode signature bukan HMAC (`jwt.go:73-75`), yang menutup celah algoritma confusion.

**Jalur upgrade bila nanti perlu revoke atau pencabutan perangkat:** refresh-token-rotate atau session server-side. Hindari IP binding karena sensitif terhadap NAT dan proxy.

## Timeout dan shutdown

| Env | Default | Fungsi |
|---|---|---|
| `SERVER_READ_HEADER_TIMEOUT` | 10s | batas waktu baca header |
| `SERVER_READ_TIMEOUT` | 10s | batas waktu baca body |
| `SERVER_WRITE_TIMEOUT` | 60s | batas waktu tulis respons |
| `SERVER_IDLE_TIMEOUT` | 120s | batas waktu keep-alive |
| `SERVER_SHUTDOWN_TIMEOUT` | 10s | tenggat graceful shutdown |

`SIGTERM`/`SIGINT` memicu `srv.Shutdown` dengan tenggat `SERVER_SHUTDOWN_TIMEOUT` (`main.go:107-117`). Sesudah itu semua rate limiter dan key manager dihentikan, lalu keenam koneksi database ditutup (`main.go:119-124`).

Konsekuensi yang perlu diingat: shutdown hanya 10 detik, sedangkan `SERVER_WRITE_TIMEOUT` 60 detik. Request yang masih berjalan lebih dari 10 detik akan terputus paksa saat deploy. Naikkan `SERVER_SHUTDOWN_TIMEOUT` bila ada request lambat yang perlu selesai.

Perhatikan juga `SERVER_READ_TIMEOUT=10s`: request dengan body besar yang butuh lebih dari 10 detik akan ditolak.

## Logging

- `logger.SetupLogger` menulis ke `logs/api.log` relatif terhadap working directory, dan sekaligus ke stdout.
- Rotasi lumberjack: maks 10 MB per berkas, 30 backup, umur 30 hari, dikompres.
- Level: `Info` saat `APP_ENV=production`, `Debug` untuk sisanya (`logger.go:51-55`).
- Setiap request menghasilkan satu baris `Audit Log` berisi `trace_id`, `method`, `path`, `ip`, `developer`, `status`, dan `latency_ms` (`tracer.go:41-49`). `trace_id` juga masuk ke baris respons dan pesan error, jadi bisa dipakai buat korelasi saat menelusuri insiden.
- Percobaan login gagal dan penolakan RBAC dicatat sebagai `Security Alert` (`auth_handler.go:44`, `jwt_rbac.go:64, 72, 79`).
- Saat `APP_ENV` selain `development`, pesan error 4xx/5xx dimasking jadi teks generik di respons, detail aslinya hanya di log server (`pkg/response/response.go:47-68`).

Direktori `logs/` harus writable oleh user service. Kalau tidak, penulisan file gagal tanpa menghentikan server.

## Rate limit

Dua limiter, keduanya token bucket per IP dengan IP dari resolver proxy.

| Limiter | Env | Default | Berlaku untuk |
|---|---|---|---|
| Global | `RATE_LIMIT_GLOBAL_RATE` / `_CAPACITY` | 50/detik, burst 100 | semua endpoint berlapis API key |
| Login | `RATE_LIMIT_LOGIN_RATE` / `_CAPACITY` | 0,2/detik, burst 5 | `POST /auth/login` saja |

`RATE_LIMIT_CLEANUP_INTERVAL` default 3 menit: entri IP yang tidak aktif lebih dari itu dihapus dari memori.

**Peringatan: `capacity = 0` berarti semua request diblokir.** Token bucket memberi `tokens = capacity` untuk IP baru (`ratelimit.go:33`), jadi nol berarti nol token dan langsung 429. Jangan set 0. Ini fail-closed, bukan fail-open.

 Kalau `TRUSTED_PROXIES` kosong dan API ada di belakang proxy, semua klien terlihat dari satu IP, sehingga rate limit global jadi satu ember bersama dan memicu 429 massal. Isi `TRUSTED_PROXIES`.

## CORS dan batas badan

- `ALLOWED_ORIGINS` default `http://localhost:3000,http://localhost:5173`, dipisah koma (`config.go:21`). origins yang tidak dikenal tidak diizinkan.
- Header yang diizinkan: `Accept`, `Authorization`, `Content-Type`, `X-CSRF-Token`, `X-API-KEY`, `X-DB-Source`. Kalau FE mengirim header lain di luar daftar itu, preflight gagal.
- `CORS_MAX_AGE` default 300 detik.
- `MAX_BODY_BYTES` default 1048576 (1 MB). Melebihi itu ditolak 413.

## Daftar periksa sebelum deploy

1. `JWT_SECRET` minimal 32 byte, unik per environment, tidak pernah masuk git.
2. Tujuh variabel wajib terisi. `API_KEYS_PATH` pakai path absolut.
3. `api_keys.json` ada dan berisi key untuk tiap developer yang perlu akses.
4. Kalau produksi: isi `TRUSTED_PROXIES` atau set `ALLOW_DIRECT_CLIENTS=true`, sesuai topologi. Kalau salah, server exit sendiri.
5. `GLOBAL_LOCAL_ONLY` dan `networkMatrix` sesuai kebijakan akses jaringan tim.
6. `POSTGRES_WRITE_ENABLED` diset sesuai apakah endpoint tulis perlu hidup. Default `false` mematikan semuanya.
7. `COOKIE_SECURE=false` hanya bila memang diakses via plain HTTP.
8. Timeout disesuaikan dengan beban, khususnya `SERVER_SHUTDOWN_TIMEOUT` bila ada request lambat.
9. Probe `/health` menyertakan `X-API-KEY`.
10. `logs/` writable oleh user service, dan rotasi log sudah ditangani (lumberjack bawaan, atau mechanism pengarsipan lain).

## Test

Unit test tidak menyentuh database dan berjalan dalam hitungan detik:

```bash
go test ./...
```

Test integrasi membaca database sungguhan dan hanya menjalankan `SELECT`. Durasi sekitar 9 menit, hampir seluruhnya habis di query MSSQL yang menyapu tabel. Jalankan dari akar repo:

```bash
go test -tags integration -count=1 -timeout 1800s ./...
```

Dua syarat: `INTEGRATION_DB=1` di lingkungan proses, dan `.env` termuat ke lingkungan proses. `LoadConfig()` memakai `godotenv.Load()` yang path-nya relatif ke working directory, sementara working directory test ada di dalam paket, jadi variabelnya perlu di-preload dari akar repo sebelum menjalankan `go test`. Tanpa `.env`, test dilewati dengan `Skip`, bukan gagal.

Test read-only dijaga secara statis oleh `internal/repository/db_readonly_guard_test.go`: seluruh `*_test.go` dipindai dan gagal bila memuat SQL tulis, DDL, pemanggilan `Exec`/`Begin`, string koneksi nyata, pemanggilan tulis repository, atau berkas `.sql`.
