# Deploy

Buat yang mau deploy hari ini. Kalau butuh alasan di balik tiap konfigurasi, ada di `deployment-notes.md`.

Butuh: Go 1.27.1 di mesin build, satu binary 27 MB, port 8080, dan 6 koneksi database yang harus hidup (3 Postgres + 3 MSSQL, satu per tenant).

Binary jalan di Windows dan di Linux, keduanya tanpa dependensi tambahan. Pilih sesuai server kamu.

## 1. Build

Di mesin yang ada Go-nya, bukan di server.

Windows (PowerShell):

```powershell
go build -trimpath -ldflags "-s -w" -o bin\api.exe .\cmd\api
```

Linux:

```bash
go build -trimpath -ldflags "-s -w" -o bin/api ./cmd/api
```

`make build` juga sudah memilih sistem operasi sendiri, jadi di Windows menghasilkan `bin\api.exe`.

## 2. Siapkan folder di server

Windows:

```
C:\apps\business-data-api\
├── api.exe
├── docs\
│   └── swagger.yaml
├── .env
├── api_keys.json
└── logs\                 # dibuat sendiri oleh proses
```

Linux:

```
/opt/business-data-api/
├── api
├── docs/
│   └── swagger.yaml
├── .env
├── api_keys.json
└── logs/                 # dibuat sendiri oleh proses
```

`docs/swagger.yaml` disalin dari repo, boleh dilewati kalau `/swagger` tidak dipakai.

## 3. Buat JWT_SECRET dan API key

Jalankan di laptop, **jangan di server**. Dua perintah ini ditolak kalau `APP_ENV=production`.

```bash
APP_ENV=development go run cmd/jwtgen/main.go
APP_ENV=development go run cmd/keygen/main.go "Nama Developer"
```

Perintah pertama mencetak `JWT_SECRET`. Perintah kedua menambah kunci ke `api_keys.json` di akar repo. Salin `api_keys.json` itu ke server.

## 4. Isi .env

Salin `.env.example` ke folder deploy, lalu isi yang wajib saja. Sisanya biarkan sesuai bawaan.

| Variabel | Isi |
|---|---|
| `JWT_SECRET` | Hasil langkah 3, minimal 32 byte |
| `POSTGRES_MAXTOP_URL`, `MSSQL_MAXTOP_URL` | URL database tenant maxtop |
| `POSTGRES_PANDORA_URL`, `MSSQL_PANDORA_URL` | URL database tenant pandora |
| `POSTGRES_TOPLINK_URL`, `MSSQL_TOPLINK_URL` | URL database tenant toplink |
| `API_KEYS_PATH` | Lokasi absolut `api_keys.json` |
| `APP_ENV` | `production` |
| `COOKIE_SECURE` | `false` hanya kalau diakses lewat HTTP polos |
| `TRUSTED_PROXIES` | IP/CIDR proxy, isi `::1` juga kalau proxy lokal |

`API_KEYS_PATH` di Windows diisi dengan `C:\apps\business-data-api\api_keys.json`.

Kalau `APP_ENV=production` dengan `TRUSTED_PROXIES` kosong dan `ALLOW_DIRECT_CLIENTS` masih `false`, server menolak berjalan. Kalau klien memang connect langsung tanpa proxy, set `ALLOW_DIRECT_CLIENTS=true` sebagai gantinya.

`POSTGRES_WRITE_ENABLED` biarkan `false` kalau tidak butuh endpoint tulis.

## 5. Jalankan

Dari dalam folder deploy. Menjalankan dari direktori lain membuat `.env` tidak terbaca, log ditulis ke tempat lain, dan `/swagger` membalas 404.

Windows:

```powershell
cd C:\apps\business-data-api
.\api.exe
```

Linux:

```bash
cd /opt/business-data-api
./api
```

## 6. Cek berhasil

```bash
curl -H "X-API-KEY: <key>" http://127.0.0.1:8080/health
```

Harus membalas 200 dengan `{"status":"sukses","data":{"status":"API berjalan dengan normal!"}}`. Tanpa header `X-API-KEY` jawabannya 401, itu bukan tanda gagal.

## 7. Kalau gagal

| Gejala di layar | Penyebab | Solusi |
|---|---|---|
| `JWT_SECRET terlalu pendek` | kurang dari 32 byte | ulang langkah 3 |
| pesan yang menyebut nama variabel | salah satu dari 6 URL database kosong | cek langkah 4 |
| `Gagal koneksi Postgres/MSSQL <tenant>` | database tidak terjangkau | cek URL, jaringan, firewall |
| `Konfigurasi TRUSTED_PROXIES tidak valid` | ada entri yang bukan IP atau CIDR | perbaiki formatnya |
| Semua endpoint membalas 401 | `api_keys.json` tidak terbaca | cek `API_KEYS_PATH` dan izin bacanya |

## 8. Menghentikan server

**Hentikan dengan Ctrl+C di jendela konsol.** itu satu-satunya cara yang menjalankan shutdown rapi di kedua sistem operasi.

Di Windows, jangan pakai `Stop-Service` atau `sc stop`. Proses ini tidak terdaftar sebagai Windows Service dan tidak punya handler Service Control Manager, jadi perintah itu akan mengakhiri proses secara paksa: `Shutdown` tidak dipanggil, 6 koneksi database tidak ditutup, request yang sedang berjalan terpotus.

Kalau server Windows harus jalan tanpa jendela konsol, pakai NSSM atau WinSW, tapi aksi berhentinya **wajib** disetel agar mengirim Ctrl+C ke process, bukan memanggil TerminateProcess. Di NSSM itu berarti `AppStopMethodConsole`. Kalau dibiarkan bawaan, setiap deploy akan memutus koneksi database dan log.

Di Linux, `systemctl stop` aman karena systemd mengirim `SIGTERM`, yang ditangani `main.go:107-108`.