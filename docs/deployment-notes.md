# Catatan Operasional

Penjelasan lengkap mengapa setiap konfigurasi produksi dipilih. Untuk langkah eksekusi, baca `deployment.md` dulu. Dokumentasi index ada terpisah di `optimize_indexes.md`.

Semua nilai di bawah diambil dari kode (`internal/config/config.go`, `cmd/api/main.go`, `cmd/api/modules.go`, `internal/handler/http/router.go`, `internal/handler/http/auth_handler.go`, `internal/middleware/`, `pkg/jwt/jwt.go`, `pkg/logger/logger.go`, `pkg/response/response.go`) dan `.env.example`.

## Kebutuhan server

### Perangkat lunak

| Kebutuhan | Keterangan |
|---|---|
| Go 1.27.1 atau lebih baru | **Hanya di mesin build.** Server tidak butuh Go (`go.mod:3`) |
| Windows Server atau Linux | Binary tunggal, tanpa dependensi runtime |
| CGO tidak aktif | `CGO_ENABLED=0`, semua driver murni Go. Tidak ada libc, DLL runtime, atau ODBC di server |

Driver database yang dipakai: `pgx/v5` v5.10.0 untuk Postgres dan `go-mssqldb` v1.11.0 untuk MSSQL. Keduanya murni Go, jadi tidak perlu memasang driver basis data apa pun di server. Windows juga tidak butuh ODBC Driver Manager untuk MSSQL.

Ukuran binary sekitar **27 MB** bila dibangun dengan `-trimpath -ldflags "-s -w"`.

Repo ini condong ke Windows dan buktinya ada di dalam: `generate_folders_windows.bat` membuat seluruh pohon direktori, `.air.toml` punya blok `[build.windows]` dengan target `tmp\main.exe`, dan `Makefile` punya cabang `ifeq ($(OS),Windows_NT)` yang menghasilkan `bin/api.exe`.

Satu perbedaan penting yang tidak terlihat di kode: **perilaku berhenti server berbeda di kedua sistem operasi.** Bagian shutdown di bawah menjelaskannya.

### Jaringan

| Kebutuhan | Keterangan |
|---|---|
| Satu port masuk | `PORT`, bawaan 8080, **HTTP polos** |
| Enam koneksi keluar ke database | 3 Postgres + 3 MSSQL, satu per tenant. Semuanya harus terjangkau sejak proses mulai |

`main.go:101` memakai `ListenAndServe` tanpa TLS, jadi **server tidak memegang sertifikat**. Kalau klien memakai HTTPS, TLS wajib dituntaskan di reverse proxy di depannya. Kalau API dibuka langsung ke internet tanpa proxy, seluruh trafik berjalan tanpa enkripsi — itu bukan konfigurasi yang layak.

Butir lain: `?limit=10000` memindahkan 4 MB per permintaan, jadi bandwidth antarmuka uplink harus diingat. Lihat `optimize_indexes.md` untuk angka pengukurannya.

### Database

Tidak ada migrasi skema sama sekali. Repo ini tidak memuat berkas `.sql`, dan tidak ada `CREATE`, `ALTER`, `DROP`, `TRUNCATE`, `GRANT`, atau `REVOKE` di seluruh kode.

Perlu dibedakan: server **tidak pernah mengubah objek basis data** (tidak ada DDL), tapi server **pernah menulis baris** kalau endpoint tulis diizinkan. Rinciannya di bagian `POSTGRES_WRITE_ENABLED`.

| Aspek | Kebutuhan |
|---|---|
| Postgres | Terverifikasi jalan di versi 14.24 |
| `max_connections` Postgres | API ini saja bisa membuka 3 × `POSTGRES_MAX_CONNS` = **30** koneksi. Nilai bawaan server yang dipakai sekarang 100, cukup |
| MSSQL | Driver `go-mssqldb`, tanpa batasan versi khusus |

Port yang perlu dibuka dari host API: Postgres 5432, MSSQL 8170 (port tidak standar, ikut dikonfigurasi lewat URL).

### Sumber daya

| Aspek | Minimum | Rekomendasi |
|---|---|---|
| RAM | 512 MB | 1 GB |
| CPU | 1 vCPU | 1 vCPU |
| Disk | 100 MB | 200 MB |

Proses API sendiri ringan. Yang paling besar adalah payload respons, yang ukurannya `?limit` × sekitar 200 byte, dan dihitung dua kali karena JSON ditahan di memori sebelum dikirim. `limit=10000` berarti sekitar 4 MB per permintaan besar.

Disk dipakai untuk binary 27 MB, `docs/swagger.yaml`, dan log. Log maksimum 10 MB per berkas dengan 30 berkas cadangan, jadi batasnya sekitar 310 MB sebelum rotasi memampatkan yang lama.

### Berkas dan izin

| Lokasi | Izin | Alasan |
|---|---|---|
| Direktori kerja | tulis | `SetupLogger` membuat `logs/` sendiri (`logger.go:40`), tapi butuh izin tulis di direktori kerja |
| `logs/` | tulis | `logs/api.log` plus rotasi lumberjack |
| `api_keys.json` | baca saja | Server hanya membacanya, lalu memuat ulang tiap 60 detik (`apikey.go:39, 60-73`). Tidak pernah ditulis server |
| `.env` | baca saja | Dimuat sekali saat proses mulai |
| `docs/swagger.yaml` | baca saja | Tanpa berkas ini, `/swagger` dan `/docs/swagger.yaml` membalas 404 (`router.go:17-35`) |

Jalankan sebagai user khusus, bukan akun administrator. Jangan memakai user database yang sama sebagai user proses.

Di Windows, `api_keys.json` yang dihasilkan `cmd/keygen` ditulis dengan mode `0644` (`cmd/keygen/main.go:41`), yang di NTFS berarti semua user bisa membaca. Folder deploy sebaiknya tidak berada di lokasi yang bisa ditulis user lain.

## Prasyarat start

Server **menolak start** (`os.Exit(1)`) kalau salah satu hal berikut tidak terpenuhi.

| Prasyarat | Sumber | Gejala kalau gagal |
|---|---|---|
| `JWT_SECRET` diisi dan panjangnya minimal 32 byte | `config.go:61` (`required`), `jwt.MinSecretLength` | keluar + pesan "JWT_SECRET terlalu pendek" |
| `POSTGRES_MAXTOP_URL` | `config.go:28` | keluar, pesan menyebut nama variabel |
| `MSSQL_MAXTOP_URL` | `config.go:29` | idem |
| `POSTGRES_PANDORA_URL` | `config.go:31` | idem |
| `MSSQL_PANDORA_URL` | `config.go:32` | idem |
| `POSTGRES_TOPLINK_URL` | `config.go:34` | idem |
| `MSSQL_TOPLINK_URL` | `config.go:35` | idem |
| `api_keys.json` terbaca di `API_KEYS_PATH` | `apikey.go:31` | **tidak keluar**, hanya peringatan, tapi semua endpoint membalas 401 |
| keenam koneksi database berhasil dibuat | `main.go:38-72` | keluar, menyebut tenant yang gagal |

Tiga tenant: `maxtop`, `pandora`, `toplink` (`cmd/api/modules.go:14-16`). Tiap tenant punya pasangan pool Postgres dan MSSQL, jadi ada enam koneksi yang dibuka saat start. Satu gagal, server mati.

Keduanya di-ping sebelum dipakai: Postgres di `postgres.go:86`, MSSQL di `mssql.go:71`. Kalau pool gagal dibuat, pool ditutup dulu sebelum galat dikembalikan, jadi tidak ada koneksi yang menggantung saat server mati.

`API_KEYS_PATH` bawaannya `api_keys.json` dan sifatnya **relatif terhadap direktori kerja**. Kalau service dijalankan dari `/`, berkas tidak ditemukan. Pakai path absolut untuk deployment non-dev.

`api_keys.json` dibaca ulang tiap 60 detik (`apikey.go:23`), jadi menambah atau mencabut API key tidak perlu memulai ulang. Kalau pemuatan ulang gagal, daftar lama dipertahankan.

## Gate produksi: TRUSTED_PROXIES wajib diisi atau ALLOW_DIRECT_CLIENTS

`cmd/api/main.go:79-85`. Bila `TRUSTED_PROXIES` kosong **dan** ketiga kondisi ini terpenuhi:

- `APP_ENV=production`
- `GLOBAL_LOCAL_ONLY=true`
- `ALLOW_DIRECT_CLIENTS=false`

maka server **`os.Exit(1)` dengan pesan galat**, bukan sekadar peringatan. Ini sengaja: konfigurasi itu berarti API tidak tahu apakah kliennya nyambung langsung atau lewat proxy, sehingga semua klien akan terlihat dari satu IP dan rate limit jadi satu ember bersama.

Isi `.env` sesuai deploy kamu:

| Situasi | Yang diisi |
|---|---|
| API di belakang reverse-proxy (nginx/IIS/LB) | `TRUSTED_PROXIES` dengan IP/CIDR proxy, `ALLOW_DIRECT_CLIENTS` boleh `false` |
| Klien connect langsung ke API | `ALLOW_DIRECT_CLIENTS=true` |

Di luar produksi, atau saat `GLOBAL_LOCAL_ONLY=false`, atau `ALLOW_DIRECT_CLIENTS=true`, `TRUSTED_PROXIES` kosong hanya memunculkan peringatan (`main.go:84`).

## TRUSTED_PROXIES dan cara header dibaca

`TRUSTED_PROXIES` berisi daftar IP/CIDR proxy, dipisah koma. Contoh: `TRUSTED_PROXIES=127.0.0.1,10.0.0.0/8`. Nilai tidak valid membuat server keluar (`main.go:74-78`). Entri tanpa CIDR otomatis jadi satu alamat: IP IPv4 jadi `/32`, IPv6 jadi `/128` (`config.go:92-100`). Entri kosong diabaikan, jadi koma berlebih tidak apa-apa.

Kosong berarti anti-spoof: `RemoteAddr` dipakai langsung, `X-Forwarded-For` dan `X-Real-IP` diabaikan. Rate limit dan `NetworkRoleGuard` berbagi resolver yang sama (`internal/middleware/network_guard.go`), jadi konsekuensi `TRUSTED_PROXIES` bukan cuma rate limit — dia juga menentukan `IsLocal` yang dipakai untuk pembatasan jaringan.

**Presedensi XFF** (`network_guard.go:39-54`) — urutan bacaannya:

1. Kalau `RemoteAddr` tidak ada di daftar proxy, langsung dipakai. Header diabaikan.
2. Kalau `RemoteAddr` ada di daftar, `X-Forwarded-For` dipindai **dari kanan ke kiri**, dan berhenti di IP pertama yang **tidak** ada di daftar proxy. IP itu yang dipakai.
3. Kalau seluruh entri XFF ada di daftar proxy (atau header kosong), baru dipakai `X-Real-IP`.
4. Kalau `X-Real-IP` juga kosong atau tidak valid, jatuh ke `RemoteAddr`.

Aturan ini yang membuat aman dari spoofing: penyerang tak bisa menyisipkan IP palsu di ujung kiri `X-Forwarded-For`, karena pemindaian selalu dimulai dari ujung kanan. Kalau urutan ini dibalik, siapa pun bisa mengaku berasal dari jaringan internal.

### IPv6 harus ditulis eksplisit

`netip.Prefix.Contains()` **tidak pernah mencocokkan antar keluarga alamat**: prefix IPv4 tidak akan cocok dengan alamat IPv6, dan sebaliknya. Jadi `TRUSTED_PROXIES=127.0.0.1` saja **tidak** menutup loopback IPv6.

Gejalanya mudah terlihat di log: browser yang membuka `http://localhost:8080` sering memilih IPv6 dulu, jadi `Audit Log` mencatat `ip=::1`. Itu IP klien yang benar, bukan cacat. Tapi kalau ada proxy di host yang sama, `::1` yang tidak terdaftar berarti `X-Forwarded-For` diabaikan, `clientAddr` mengembalikan IP proxy, lalu `IsLocal()` bernilai `true` karena loopback. Akibatnya semua klien dari luar jaringan lolos pembatasan jaringan, termasuk peran `sa` yang seharusnya ditolak.

Tuliskan keduanya:

```
TRUSTED_PROXIES=127.0.0.1,::1,10.0.0.0/8,fe80::/10
```

`::1` tanpa CIDR otomatis jadi `::1/128` (`config.go:99`). Alamat IPv4-mapped seperti `::ffff:10.0.0.1` otomatis dilepas ke bentuk IPv4-nya (`network_guard.go:71`), jadi cukup tulis `10.0.0.0/8`.

### ALLOW_DIRECT_CLIENTS hanya sah tanpa proxy di host yang sama

`ALLOW_DIRECT_CLIENTS=true` berarti: "klien connect langsung, tidak ada proxy". Kalau ternyata ada reverse-proxy di host yang sama dan `TRUSTED_PROXIES` tetap kosong, setiap request masuk dengan IP loopback, sehingga `IsLocal()` selalu `true` dan `GLOBAL_LOCAL_ONLY` beserta aturan `sa`-wajib-lokal kehilangan makna.

Kalau ada proxy di host yang sama, **isi `TRUSTED_PROXIES`** dan biarkan `ALLOW_DIRECT_CLIENTS=false`.

## Pembatasan jaringan: GLOBAL_LOCAL_ONLY

- `GLOBAL_LOCAL_ONLY=true` (bawaan): semua request harus datang dari loopback atau jaringan privat, apa pun perannya. Non-lokal dapat 403.
- `GLOBAL_LOCAL_ONLY=false`: pembatasan turun ke level peran lewat `networkMatrix` (`cmd/api/modules.go:24-27`). Nilai `true` berarti peran itu **wajib lokal**, `false` berarti bebas dari luar jaringan.

Matriks peran saat ini (`cmd/api/modules.go:24-27`):

```go
networkMatrix := map[string]bool{
    "*":    true,   // peran lain WAJIB lokal
    roleSA: false,  // sa bebas dari luar jaringan
}
```

| Peran | Nilai matriks | Akses dari luar jaringan |
|---|---|---|
| `sa` | `false` | **boleh** |
| `op` | jatuh ke `"*"` = `true` | **ditolak, 403** |
| `opout` | jatuh ke `"*"` = `true` | **ditolak, 403** |

Perhatikan arahnya: peran **`sa` justru satu-satunya yang bebas** dari luar jaringan. `op` dan `opout` tidak punya entri sendiri, jadi keduanya memakai nilai `"*"`, yaitu wajib lokal.

Konsekuensi yang tidak obvious: `GLOBAL_LOCAL_ONLY=false` bukan berarti melonggarkan. Dengan `true` semua orang terkunci, dengan `false` yang terkunci justru `op` dan `opout`, sementara `sa` dilepas ke luar jaringan.

Fallback-nya dua lapis (`network_guard.go:100-103`, `network_guard.go:116-119`): kalau nama peran tidak ada di matriks, dipakai entri `"*"`. Kalau `"*"` juga tidak ada, hasilnya `false`, artinya semua peran bebas dari luar jaringan. Matriks kosong berarti tidak ada pembatasan peran sama sekali.

### Apa yang sebenarnya diperiksa

`IsLocal` (`network_guard.go:74-77`) hanya menerima `IsLoopback() || IsPrivate()`. Menurut `go doc`, `IsPrivate` berarti persis `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, atau `fc00::/7`. Jadi `169.254.0.0/16`, `fe80::/10`, dan `100.64.0.0/10` **tidak** dihitung lokal, danKlien dari alamat seperti itu tetap dapat 403 saat `GLOBAL_LOCAL_ONLY=true`.

Menaruh `fe80::/10` di `TRUSTED_PROXIES` hanya membuat proxy tersebut dipercaya untuk dibaca XFF-nya. Itu tidak membuat klien yang datang dari `fe80::` dihitung lokal.

### Di route mana guard ini benar-benar aktif

`NetworkRoleGuard` dipasang di dua grup (`router.go:84-91`):

```go
public    := protectedApiKey.With(NetworkRoleGuard(...))
protected := protectedApiKey.With(RequireToken(), NetworkRoleGuard(...))
```

Cabang role di dalam guard membaca `claimsKey` (`network_guard.go:92`). Klaim itu hanya ada kalau `RequireToken` sudah berjalan. Route `public` tidak memasang `RequireToken`, jadi `claims` selalu kosong dan cabang role **selalu lolos** (`network_guard.go:92-96`).

Artinya: pembatasan peran hanya benar-benar berlaku di route `protected` — inbox, outbox, master, auth/users. Route `public` hanya `/health` dan `POST /auth/login`.

Karena `POST /auth/login` ada di `public`, dia butuh pemeriksaan sendiri di `auth_handler.go:49` (`IsLocal` + `IsRoleAllowedFromOutside`). Tanpa baris itu, login dari luar jaringan tidak akan pernah diblokir.

Pengguna `op` atau `opout` yang login dari luar jaringan tetap melewati verifikasi sandi lebih dulu (`auth_handler.go:42-47`), baru mendapat 403 di baris 49-52. `sa` lolos dan mendapat token.

## POSTGRES_WRITE_ENABLED

`POSTGRES_WRITE_ENABLED` bawaan `false`. Selama `false`, `PostgresWriteGuard` (`internal/middleware/postgres_write_guard.go:9-25`) memblokir `POST`, `PUT`, dan `DELETE` ke Postgres dengan 403.

Jadi di environment bawaan, seluruh endpoint tulis ke Postgres mati. Endpointnya: `POST/PUT /inbox`, `POST/PUT /outbox`, `POST/PUT /auth/users`. Endpoint baca tidak terpengaruh.

Perhatikan header pemilih: guard hanya menolak kalau `X-DB-Source` kosong (bawaan Postgres) atau bernilai persis `postgres` (`postgres_write_guard.go:13-17`). Request dengan `X-DB-Source: mssql` **tidak** lewat guard ini.

Artinya proteksi write hanya berlaku untuk Postgres. Jalur MSSQL tidak punya guard write apa pun, dan repository-nya memang berisi `INSERT` dan `UPDATE` sungguhan:

| Tabel | Postgres | MSSQL |
|---|---|---|
| `users` | `user_repository.go:36, 45` | `user_repository.go:69, 78` |
| `inbox` | `inbox_repository.go:108, 138` | `inbox_repository.go:185` |
| `inbox` pembaruan | `inbox_repository.go:122` | `inbox_repository.go:199` |
| `outbox` sisip | `outbox_repository.go:86` | `outbox_repository.go:167` |
| `outbox` pembaruan | `outbox_repository.go:101` | `outbox_repository.go:182` |

Tidak ada `DELETE` di mana pun. Yang tidak ada di repository adalah DDL — tidak ada `CREATE`, `ALTER`, `DROP`, `TRUNCATE`, `GRANT`, atau `REVOKE`.

Set `POSTGRES_WRITE_ENABLED=true` hanya di environment yang memang butuh endpoint tulis, dan tetap jaga `GLOBAL_LOCAL_ONLY` — dua-duanya lapisan berbeda.

## Autentikasi berlapis

Setiap endpoint melewati lapis API key dan rate limit global. Pengecualiannya `/docs/swagger.yaml` dan `/swagger/*`, keduanya dipasang langsung di router tanpa `protectedApiKey` (`router.go:53-59`), jadi tidak butuh API key dan tidak dibatasi jaringan.

**Lapis 1 — API key.** Header `X-API-KEY` wajib, nilai harus ada di `api_keys.json` (`apikey.go:81-101`). Kosong → 401 "Header X-API-KEY kosong", tidak terdaftar → 401 "API Key tidak terdaftar". Nama developer dari key tersebut ikut masuk ke setiap baris log sebagai `developer`.

Kedua lapis ini dipasang di grup `protectedApiKey` (`router.go:71-74`). Route di luar grup itu dijawab `404` atau `405` oleh handler `NotFound`/`MethodNotAllowed` milik grup tersebut (`router.go:76-82`) — bukan oleh router utama. Terverifikasi: `GET /tidak-ada` dengan key yang benar membalas 404.

**Lapis 2 — token dan peran.** Hanya di route `protected`. `Authorization: Bearer <token>` wajib dan harus valid (`jwt_rbac.go:20-35`). Tanpa header → 401 "Token otorisasi diperlukan", format salah → 401, token tidak valid → 401. Setelah itu klaim `tenant` pada token harus sama dengan `{tenant}` di path, kalau tidak 403 "Token tidak berlaku untuk tenant ini" (`jwt_rbac.go:46-50`). Terverifikasi: login kosong membalas 401.

RBAC per endpoint (`cmd/api/modules.go:29-36`), dipetakan ke route di `inbox_handler.go:184`, `outbox_handler.go:165`, `master_handler.go:44`, `auth_handler.go:127`:

| Izin | Route | Peran yang boleh |
|---|---|---|
| `ManageUsers` | `POST/PUT /api/v1/{tenant}/auth/users` | `sa` |
| `ReadResellerDropdown` | `GET /api/v1/{tenant}/master/reseller-dropdown` | `sa`, `op`, `opout` |
| `ReadInbox` | `GET /api/v1/{tenant}/inbox` | `sa`, `op`, `opout` |
| `WriteInbox` | `POST/PUT /api/v1/{tenant}/inbox` | `sa` |
| `ReadOutbox` | `GET /api/v1/{tenant}/outbox` | `sa`, `op`, `opout` |
| `WriteOutbox` | `POST/PUT /api/v1/{tenant}/outbox` | `sa` |

Perhatikan `POSTGRES_WRITE_ENABLED` hanya dipasang di grup tulis (`auth_handler.go:135`, `inbox_handler.go:193`, `outbox_handler.go:174`). Route baca dan `master/reseller-dropdown` tidak punya guard sama sekali.

Role yang tidak dikenal atau tidak punya entri di matriks RBAC selalu mendapat 403 (`jwt_rbac.go:77-82`), jadi peran baru harus ditambah ke `roleMatrix` dulu sebelum bisa dipakai.

**Catatan `/health` untuk load balancer.** Endpoint `/health` dipasang di grup `public` (`router.go:93`), yang berarti dilewati tiga lapis: `keyManager.Middleware()`, `globalLimiter.Middleware()` (`router.go:71-74`), lalu `NetworkRoleGuard` (`router.go:84-86`).

Konsekuensinya ada dua, bukan satu:

1. Tanpa header `X-API-KEY` jawabannya 401 (`apikey.go:88`).
2. Dengan key yang benar tapi dari alamat non-lokal, jawabannya **403** selama `GLOBAL_LOCAL_ONLY=true`.

Probe load balancer dari luar jaringan akan selalu gagal. Beri header key **dan** jalankan dari alamat loopback atau jaringan privat, atau arahkan probe ke pemeriksaan lain.

## COOKIE_SECURE

- `true` (bawaan): cookie `access_token` hanya dikirim lewat HTTPS.
- Set `false` bila API diakses lewat HTTP polos (LAN/dev).
- Kalau diakses lewat HTTPS, biarkan `true`.

Login juga mengembalikan token di body respons (`auth_handler.go:69-73`), jadi akses HTTP tidak menghalangi auth selama klien memakai token dari body, bukan cookie. Cookie-nya sendiri `HttpOnly` dan `SameSite=Strict`.

Umur cookie mengikuti `JWT_TOKEN_DURATION` (`auth_handler.go:128`), jadi satu variabel itu mengatur dua hal sekaligus.

## Model autentikasi (keputusan arsitektur)

JWT stateless, tidak terikat IP, perangkat, atau peramban. Siapa pun yang memegang token punya akses penuh sampai `exp`.

Pola token (`pkg/jwt/jwt.go:53-62`):

| Klaim | Status |
|---|---|
| `exp` | **wajib**, ditolak kalau tidak ada (`WithExpirationRequired`) |
| `iss` | divalidasi manual, harus sama dengan `JWT_ISSUER` (`jwt.go:87-95`) — **tapi hanya selama `JWT_ISSUER` tidak kosong** (`jwt.go:87`). Kalau `JWT_ISSUER=` dikosongkan, klaim `iss` dilewati sepenuhnya |
| `iat` | divalidasi (`WithIssuedAt`), `iat` di masa depan ditolak |
| `nbf` | **dikirim tapi tidak divalidasi** — tidak ada `WithNotBefore()` dan `jwt.go` tidak pernah membaca `nbf` |
| `user_id`, `username`, `rules`, `tenant` | dipakai untuk RBAC dan pencocokan tenant |

Kelonggaran waktu 30 detik (`tokenLeeway`, `jwt.go:13`) untuk semua pemeriksaan waktu.

Token ditolak juga bila metode tanda tangan bukan HMAC (`jwt.go:73-75`), yang menutup celah algoritma confusion.

**Jalur peningkatan bila nanti perlu pencabutan atau pencabutan perangkat:** refresh-token-rotate atau session server-side. Hindari pengikatan IP karena sensitif terhadap NAT dan proxy.

## Timeout dan shutdown

| Env | Bawaan | Fungsi |
|---|---|---|
| `SERVER_READ_HEADER_TIMEOUT` | 10s | batas waktu baca header |
| `SERVER_READ_TIMEOUT` | 10s | batas waktu baca body |
| `SERVER_WRITE_TIMEOUT` | 60s | batas waktu tulis respons |
| `SERVER_IDLE_TIMEOUT` | 120s | batas waktu keep-alive |
| `SERVER_SHUTDOWN_TIMEOUT` | 10s | tenggat shutdown rapi |

Kode shutdounya sama di kedua sistem operasi (`main.go:107-124`): tunggu sinyal, `srv.Shutdown` dengan tenggat `SERVER_SHUTDOWN_TIMEOUT`, hentikan semua rate limiter dan key manager, lalu tutup keenam koneksi database lewat `CloseAll` (`registry.go:57-68`).

**Tapi sinyal yang bisa sampai ke proses berbeda.**

| Cara menghentikan | Linux | Windows |
|---|---|---|
| Ctrl+C di konsol | `SIGINT` | `os.Interrupt` |
| Init sistem / service manager | `SIGTERM` | **tidak ada** |
| Jendela konsol ditutup | `SIGTERM` | `SIGTERM` via `CTRL_CLOSE_EVENT` |
| Logoff atau shutdown Windows | tidak relevan | `SIGTERM` via `CTRL_LOGOFF_EVENT`/`CTRL_SHUTDOWN_EVENT` |
| `Process.Kill` | `SIGKILL` | `TerminateProcess` |

Jadi di Windows, `os.Interrupt` dan `SIGTERM` sama-sama bisa sampai, tapi hanya lewat jalur interaktif. `go doc os.Process.Signal` menyatakan tegas: "Sending Interrupt on Windows is not implemented" — tidak ada proses lain yang bisa mengirim sinyal hangat ke proses ini.

Akibatnya di Windows:

- `Stop-Service` dan `sc stop` **tidak menjalankan shutdown rapi**. Keduanya memanggil TerminateProcess, jadi `srv.Shutdown` tidak pernah dipanggil, keenam koneksi database tidak ditutup, dan request yang sedang berjalan terpotus.
- Aplikasi ini juga **tidak bisa didaftarkan sebagai Windows Service** dengan benar, karena tidak ada handler Service Control Manager. `go.mod` tidak punya `golang.org/x/sys/windows/svc`, dan pencarian di seluruh kode tidak menemukan impor `svc`. Proses yang dipegang SCM akan dianggap tidak merespons.
- Kalau harus jalan tanpa jendela konsol, bungkus dengan NSSM atau WinSW, tapi aksi berhentinya wajib disetel agar mengirim Ctrl+C ke process. Di NSSM itu `AppStopMethodConsole`. Dibiarkan bawaan, setiap deploy akan memutus koneksi database dan log.

Di Linux tidak ada masalah ini: systemd mengirim `SIGTERM`, dan `main.go:108` mendaftarkannya.

Konsekuensi lain yang perlu diingat di kedua sistem: tenggat shutdown hanya 10 detik, sedangkan `SERVER_WRITE_TIMEOUT` 60 detik. Request yang masih berjalan lebih dari 10 detik akan terputus paksa saat deploy. Naikkan `SERVER_SHUTDOWN_TIMEOUT` bila ada request lambat yang perlu selesai.

Perhatikan juga `SERVER_READ_TIMEOUT=10s`: request dengan body besar yang butuh lebih dari 10 detik akan ditolak.

## Logging

- `logger.SetupLogger` menulis ke `logs/api.log` relatif terhadap direktori kerja, dan sekaligus ke stdout.
- Rotasi lumberjack: maks 10 MB per berkas, 30 cadangan, umur 30 hari, dikompres (`logger.go:42-48`).
- Tingkat: `Info` saat `APP_ENV=production`, `Debug` untuk sisanya (`logger.go:51-55`).
- Setiap request menghasilkan satu baris `Audit Log` berisi `trace_id`, `method`, `path`, `ip`, `developer`, `status`, dan `latency_ms` (`tracer.go:41-49`). `trace_id` juga masuk ke baris respons dan pesan galat, jadi bisa dipakai buat korelasi saat menelusuri insiden.
- Percobaan login gagal dan penolakan RBAC dicatat sebagai `Security Alert` (`auth_handler.go:44`, `jwt_rbac.go:64, 72, 79`).
- Setiap query SQL juga dicatat sebagai `SQL Query Executed` dengan `trace_id`, `db`, `sql`, dan `duration_ms`. Query yang lebih lambat dari **500 ms** dicatat sebagai `Slow SQL Query` dengan tingkat `WARN` (`pkg/database/slow_query.go:5`, `postgres.go:38-39`, `mssql.go:24-25`). Ini yang dipakai untuk mengukur query lambat di produksi.
- Panic dicatat sebagai `System Panic / Crash` lengkap dengan stack (`recoverer.go:20`).
- Saat `APP_ENV` selain `development`, pesan galat 4xx/5xx dimasking jadi teks generik di respons, detail aslinya hanya di log server (`response.go:14`, `response.go:47-68`).

Direktori `logs/` harus bisa ditulis oleh user service. Kalau tidak, penulisan berkas gagal tanpa menghentikan server.

Catatan: `developer` di baris log berasal dari nama yang dipetakan dari API key. Kalau pemuatan `api_keys.json` gagal, nilainya `"Tidak Diketahui"` (`tracer.go:33`).

## Rate limit

Dua limiter, keduanya token bucket per IP dengan IP dari resolver proxy.

| Limiter | Env | Bawaan | Berlaku untuk |
|---|---|---|---|
| Global | `RATE_LIMIT_GLOBAL_RATE` / `_CAPACITY` | 50/detik, burst 100 | semua endpoint berlapis API key |
| Login | `RATE_LIMIT_LOGIN_RATE` / `_CAPACITY` | 0,2/detik, burst 5 | `POST /auth/login` saja |

`RATE_LIMIT_CLEANUP_INTERVAL` bawaan 3 menit: entri IP yang tidak aktif lebih dari itu dihapus dari memori (`ratelimit.go:52-73`, `ratelimit.go:101`). Kalau nilainya 0 atau negatif, `NewRateLimiter` memakai 3 menit sebagai bawaan (`ratelimit.go:104-106`).

Rate limiter, `SecurityTracer`, dan `NetworkRoleGuard` memakai `clientAddr` yang sama, jadi IP yang tercatat di `Audit Log`, IP yang dipakai rate limit, dan IP yang dipakai penentuan `IsLocal` selalu konsisten satu sumber.

**Peringatan: `capacity = 0` berarti semua request diblokir.** Token bucket memberi `tokens = capacity` untuk IP baru (`ratelimit.go:33`), jadi nol berarti nol token dan langsung 429. Jangan set 0. Ini fail-closed, bukan fail-open.

Kalau `TRUSTED_PROXIES` kosong dan API ada di belakang proxy, semua klien terlihat dari satu IP, sehingga rate limit global jadi satu ember bersama dan memicu 429 massal. Isi `TRUSTED_PROXIES`.

## CORS dan batas badan

- `ALLOWED_ORIGINS` bawaan `http://localhost:3000,http://localhost:5173`, dipisah koma (`config.go:21`, `config.go:66-78`). Origin yang tidak dikenal tidak diizinkan.
- Metode yang diizinkan: `GET`, `POST`, `PUT`, `DELETE`, `OPTIONS` (`router.go:42`).
- Header yang diizinkan: `Accept`, `Authorization`, `Content-Type`, `X-CSRF-Token`, `X-API-KEY`, `X-DB-Source` (`router.go:43`). Kalau FE mengirim header lain di luar daftar itu, pemeriksaan awal gagal.
- Header yang diekspos ke browser: `Link`, `Set-Cookie` (`router.go:44`).
- `CORS_MAX_AGE` bawaan 300 detik.
- `MAX_BODY_BYTES` bawaan 1048576 (1 MB). Melebihi itu ditolak 413, dicek dua kali: dari `Content-Length` (`bodylimit.go:12`) dan dari pembacaan body lewat `MaxBytesReader` (`bodylimit.go:16-18`).

## Mendaftarkan sebagai service

### Linux

`WorkingDirectory` **wajib** diisi, bukan opsional, karena `.env`, `logs/`, `api_keys.json`, dan `docs/swagger.yaml` semuanya relatif terhadap direktori kerja.

```ini
[Unit]
Description=Business Data API
After=network-online.target
StartLimitIntervalSec=60
StartLimitBurst=5

[Service]
Type=simple
User=apisvc
WorkingDirectory=/opt/business-data-api
ExecStart=/opt/business-data-api/api
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/opt/business-data-api/logs

[Install]
WantedBy=multi-user.target
```

`ProtectSystem=strict` bersama `ReadWritePaths` membuat seluruh direktori deploy tidak bisa ditulis kecuali `logs/`.

`StartLimitIntervalSec` dan `StartLimitBurst` sengaja dipasang. Kesalahan konfigurasi memicu `os.Exit(1)`, jadi dengan `Restart=on-failure` tanpa batas, satu `.env` yang salah akan menyebabkan proses mulai ulang terus-menerus tanpa henti. Batas ini membuatnya berhenti setelah lima kegagalan dalam 60 detik, sehingga penyebabnya bisa dibaca dari log.

```bash
systemctl daemon-reload
systemctl enable --now business-data-api
systemctl status business-data-api
journalctl -u business-data-api -f
```

Log systemd tidak sama dengan `logs/api.log`. Di bawah systemd, `slog` menulis ke stdout yang ditangkap systemd, sedangkan lumberjack tetap menulis ke `logs/api.log`. Isi kedua sumber itu tidak sama.

### Windows

Aplikasi ini **tidak bisa didaftarkan sebagai Windows Service** lewat Service Control Manager, karena tidak ada handler SCM di kodenya. `sc create` akan membuat entri yang tidak merespons. Yang harus dipakai adalah pembungkus yang menjalankan proses sebagai anak dan mengirim Ctrl+C saat dihentikan.

Opsi paling sederhana: Scheduled Task dengan setting "Run whether user is logged on or not". Kalau butuh impersonal, pakai NSSM dengan konfigurasi berikut.

```cmd
nssm install BusinessDataApi C:\apps\business-data-api\api.exe
nssm set BusinessDataApi AppDirectory C:\apps\business-data-api
nssm set BusinessDataApi AppParameters ""
nssm set BusinessDataApi AppStopMethodConsole 0
nssm set BusinessDataApi AppStdout C:\apps\business-data-api\logs\nssql-out.log
nssm set BusinessDataApi AppStderr C:\apps\business-data-api\logs\nssql-err.log
nssm set BusinessDataApi AppRotateFiles 1
nssm start BusinessDataApi
```

`AppDirectory` wajib diisi karena `.env` dan `logs/` relatif terhadap direktori kerja. `AppStopMethodConsole 0` adalah bagian penting: tanpa itu, `nssm stop` memanggil TerminateProcess dan shutdown rapi tidak pernah jalan.

Tidak perlu `AppExit` untuk `api.exe` di sini. Kalau butuh mulai ulang otomatis saat konfigurasi salah, pakai Scheduled Task yang selalu dijalankan ulang, tapi beri jeda antar percobaan — tanpa batas, satu `.env` yang salah akan membuat proses berputar tanpa henti.

### Perbedaan penting

| | Linux | Windows |
|---|---|---|
| Path | `/opt/business-data-api` | `C:\apps\business-data-api` |
| Binary | `api` | `api.exe` |
| Perintah build | `go build -o bin/api ./cmd/api` | `go build -o bin\api.exe .\cmd\api` |
| Penataan | `systemctl` | Scheduled Task atau NSSM |
| Cara berhenti | `systemctl stop` mengirim `SIGTERM` | Ctrl+C, atau NSSM dengan `AppStopMethodConsole` |
| Perintah yang **tidak boleh** dipakai | tidak ada | `Stop-Service`, `sc stop`, Task Manager "End task" |

## Mengembalikan versi lama

Tidak ada migrasi skema, jadi tidak ada langkah basis data saat mengembalikan versi. Cukup: hentikan proses, ganti berkas binary, jalankan lagi.

Linux:

```bash
systemctl stop business-data-api
cp /opt/business-data-api/api.previous /opt/business-data-api/api
systemctl start business-data-api
```

Windows (dari jendela konsol yang menjalankan server, tekan Ctrl+C lebih dulu):

```powershell
Copy-Item .\api.exe .\api.previous
Copy-Item .\api.exe.previous .\api.exe
```

Kalau server dijalankan lewat NSSM: `nssm stop BusinessDataApi`, ganti berkas, `nssm start BusinessDataApi`.

Satu catatan: mengembalikan versi kode **tidak** membalikkan data yang sudah berubah. Kalau endpoint tulis pernah aktif (`POSTGRES_WRITE_ENABLED=true` atau lewat jalur MSSQL yang tidak dikunci), perubahan data di database tetap ada setelah kode dikembalikan.

## Test

Unit test tidak menyentuh database dan berjalan dalam hitungan detik:

```bash
go test ./...
```

Test integrasi membaca database sungguhan dan hanya menjalankan `SELECT`. Durasi sekitar 9 menit, hampir seluruhnya habis di query MSSQL yang menyapu tabel. Jalankan dari akar repo:

```bash
go test -tags integration -count=1 -timeout 1800s ./...
```

Dua syarat: `INTEGRATION_DB=1` di lingkungan proses, dan `.env` termuat ke lingkungan proses. `LoadConfig()` memakai `godotenv.Load()` yang path-nya relatif ke direktori kerja, sementara direktori kerja test ada di dalam paket, jadi variabelnya perlu di-preload dari akar repo sebelum menjalankan `go test`. Tanpa `.env`, test dilewati dengan `Skip`, bukan gagal.

Test read-only dijaga secara statis oleh `internal/repository/db_readonly_guard_test.go`. Guard memindai seluruh `*_test.go` di repo dan gagal bila menemukan pola berikut:

| Pola yang dilarang | Contoh |
|---|---|
| penyisipan baris | `INSERT INTO` |
| pembaruan baris | `UPDATE <nama> SET` |
| penghapusan baris | `DELETE FROM` |
| penggabungan baris | `MERGE INTO` |
| pengosongan tabel | `TRUNCATE` |
| penghapusan objek | `DROP TABLE/INDEX/DATABASE/SCHEMA/VIEW` |
| perubahan skema | `ALTER TABLE` |
| pembuatan objek | `CREATE TABLE/INDEX/DATABASE/SCHEMA/UNIQUE` |
| pengaturan hak akses | `GRANT`, `REVOKE` |
| eksekusi perintah | `.Exec(`, `.ExecContext(`, `.Begin(`, `.BeginTx(` |
| string koneksi nyata | `postgres://`, `postgresql://`, `sqlserver://`, `mysql://` |
| pemanggilan tulis repository | `.Insert(`, `.Update(`, `.Delete(` di dalam `internal/repository/` |
| berkas `.sql` | keberadaannya di repo |

Ada enam berkas test yang dikecualikan, masing-masing wajib menyertakan alasannya di `allowances` (`db_readonly_guard_test.go:19-44`). Kalau sebuah pengecualian tidak lagi cocok dengan pola mana pun, guard justru gagal dan meminta pengecualian itu dihapus — jadi daftar tidak bisa mengendap jadi sia-sia.