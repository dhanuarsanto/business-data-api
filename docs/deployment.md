# Deployment & Operasional

Catatan khusus yang perlu diketahui tim saat deploy/operasional. Dokumen ini terpisah dari index docs (`optimize_indexes.md`) yang khusus membahas index.

## TRUSTED_PROXIES (rate limit per-IP)

Server perlu tahu proxy mana yang jujur supaya rate-limiter memakai IP user asli, bukan IP proxy.

- `TRUSTED_PROXIES` di `.env`: daftar IP/CIDR proxy (nginx/IIS/LB), dipisah koma. Contoh: `TRUSTED_PROXIES=127.0.0.1,10.0.0.0/8`.
- **Kosong (default)** = aman anti-spoof: header `X-Forwarded-For`/`X-Real-IP` dari IP tak dikenal diabaikan. `RemoteAddr` dipakai langsung.
- **Peringatan saat kosong + API di belakang proxy**: semua client terlihat dari IP proxy → rate-limit jadi satu bucket bersama → bisa menyebabkan 429 massal. Startup akan log warning bila kosong.
- Header hanya dihormati dari proxy yang terdaftar (`internal/middleware/ratelimit.go`, `network_guard.go`).

## COOKIE_SECURE

- `true` (default) → cookie login hanya dikirim lewat HTTPS.
- Set `false` di `.env` bila API diakses via plain HTTP (LAN/dev). Bila diakses via HTTPS, biarkan `true`.
- Catatan: login juga mengembalikan token di body response, jadi akses HTTP tidak menghalangi auth selama klien memakai token body (bukan cookie).

## Model Autentikasi (keputusan arsitektur)

- **JWT stateless** — tidak terikat IP/device/browser. Siapa pun yang memegang token (dari cookie atau body) = otentik penuh sampai `exp`.
- Claim wajib dan tervalidasi di `pkg/jwt`: `iss`, `iat`, `nbf`, `exp` (leeway 30 detik). Token tanpa `exp` atau dengan `iat` di masa depan ditolak.
- **Jalur upgrade bila nanti perlu revoke/cabut device**: refresh-token-rotate atau session server-side. Hindari IP binding (sensitif terhadap NAT/proxy).