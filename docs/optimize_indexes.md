# Index PostgreSQL — Inbox & Outbox (Status Aktual)

Dokumen ini mencerminkan **kondisi index aktual di PostgreSQL** (`pandora_dw.staging`) dan daftar index MSSQL yang harus dibuat manual (bagian bawah).

Semua DDL sudah DIEKSEKUSI manual di database. Jangan membuat ulang tanpa alasan.

## Kondisi aktual — `inbox`

| Index | Definisi (perkiraan) | Dipakai untuk |
|---|---|---|
| `pk_inbox` | UNIQUE btree (kode) | PK / pagination |
| `idx_inbox_tgl_entri` | btree (tgl_entri DESC, kode DESC) | filter tanggal |
| `idx_inbox_kode_tgl` | btree (kode DESC) INCLUDE (tgl_entri) | **bisection pagination** (`kode_cut`) — jangan di-drop |
| `idx_inbox_status` | btree (status) WHERE status<20 | dropdown status |
| `idx_inbox_status_tipe` | btree (status, tipe_pengirim) | kombinasi dropdown |
| `idx_inbox_terminal` | btree (kode_terminal) WHERE IS NOT NULL | dropdown terminal |
| `idx_inbox_tipe` | btree (tipe_pengirim) WHERE IS NOT NULL | dropdown tipe |
| `idx_inbox_kode_reseller` | btree (kode_reseller) | dropdown equality reseller |
| `idx_inbox_tgl_status` | btree (tgl_status) | lookup tgl_status |
| `idx_inbox_pesan_trgm` | GIN (pesan gin_trgm_ops) | ILIKE pesan |
| `idx_inbox_pengirim_trgm` | GIN (pengirim gin_trgm_ops) | ILIKE pengirim |
| `idx_inbox_kode_reseller_jawaban` | partial btree (kode DESC) WHERE kode_reseller IS NOT NULL AND is_jawaban = 0 | filter triage `requestFromReseller` |
| `idx_inbox_tgl_entri_kode_reseller_jawaban` | partial btree (tgl_entri DESC, kode DESC) WHERE ... | kombinasi tanggal + triage |

> Trigram **kode_reseller** (`idx_*_reseller_trgm`) sudah **DIDROP** — diganti btree equality (dropdown). Selesai.

## Kondisi aktual — `outbox`

| Index | Dipakai untuk |
|---|---|
| `pk_outbox` | PK / pagination |
| `idx_outbox_tgl_entri` | filter tanggal |
| `idx_outbox_kode_tgl` | **bisection pagination** — jangan di-drop |
| `idx_outbox_kode_reseller` | dropdown equality reseller |
| `idx_outbox_tipe` | dropdown tipe |
| `idx_outbox_penerima_trgm` / `idx_outbox_pesan_trgm` | ILIKE penerima / pesan |
| `ix_kode_inbox` / `ix_kode_transaksi` | lookup relasional |
| `ix_outbox_status` / `ix_outbox_tgl_status` | dropdown status |
| `idx_inbox_kode_reseller_perintah` | partial (kode DESC) WHERE kode_reseller IS NOT NULL AND is_perintah=0 — triage `replyToReseller` |
| `idx_inbox_tgl_entri_kode_reseller_perintah` | partial (tgl_entri DESC, kode DESC) WHERE ... — triage + tanggal |

> Nama `idx_inbox_*_perintah` di tabel `outbox` — nama memang terdengar aneh (hasil kreasi manual), tapi menunjuk tabel outbox dengan benar. Perbaikan nama = DDL opsional, bukan keharusan.

## Perilaku yang bergantung pada index ini

1. **Pengurutan hasil = `ORDER BY kode DESC`** (terbaru dulu). Kosongkan seluruh `tgl_entri` dukungan bisection & index terarah.
2. **Bisection** (`kode_cut`) memanfaatkan `idx_*_kode_tgl (kode DESC INCLUDE tgl_entri)` + slack 50k. Sewaktu `StartDate` diset, batas bawah juga dibisect (`kode > cutStart`) sehingga filter tanggal `tgl_entri >= start` tidak lagi dipakai query utama — hasil identik secara logika, tapi planner selalu lewat index kode (cepat deterministik, tanpa Bitmap/Seq scan).
3. **Partial jawaban/perintah** duduk untuk filter triase (`requestFromReseller` / `replyToReseller` + `is_jawaban=0` / `is_perintah=0`); versi `tgl_entri DESC` digunakan saat filter tanggal + triase. Kedua flag triase saling lepas: menyalakan keduanya berarti tidak ada syarat triase sama sekali, sama seperti tidak menyalakan apa pun.
4. Pada jalur tgl-leading, subquery memilih `n` baris terbaru dengan `ORDER BY tgl_entri DESC, kode DESC`, lalu query luar mengurutkan `i.kode DESC`. Urutan luar wajib `kode DESC` supaya klien selalu menerima kode terbesar lebih dulu dan baris yang sama tidak pernah muncul di dua permintaan berbeda. Index tgl-leading di tabel tetap berguna karena melayani pemilihan `n` baris terbaru di dalam subquery.
5. Sort kolom di luar `kode` dan `tgl_entri` **belum** didukung (butuh keputusan & index per kolom — lihat catatan).

## MSSQL — index yang HARUS dibuat manual (untuk pola varian tgl+flag)

Query varian di API (PG) memakai index tgl-leading; untuk MSSQL jalur yang sama cepat,
bikin index berikut (DBeaver / SQL, manual):

```sql
-- range tanggal umum
CREATE NONCLUSTERED INDEX IX_inbox_tgl ON dbo.inbox (tgl_entri DESC, kode DESC);
CREATE NONCLUSTERED INDEX IX_outbox_tgl ON dbo.outbox (tgl_entri DESC, kode DESC);

-- jawaban provider (is_jawaban = 1)
CREATE NONCLUSTERED INDEX IX_inbox_jawaban     ON dbo.inbox (kode DESC) WHERE is_jawaban = 1;
CREATE NONCLUSTERED INDEX IX_inbox_tgl_jawaban ON dbo.inbox (tgl_entri DESC, kode DESC) WHERE is_jawaban = 1;

-- request dari reseller (is_jawaban = 0)
CREATE NONCLUSTERED INDEX IX_inbox_request     ON dbo.inbox (kode DESC) WHERE kode_reseller IS NOT NULL AND is_jawaban = 0;
CREATE NONCLUSTERED INDEX IX_inbox_tgl_request ON dbo.inbox (tgl_entri DESC, kode DESC) WHERE kode_reseller IS NOT NULL AND is_jawaban = 0;

-- outbox: perintah provider (is_perintah = 1)
CREATE NONCLUSTERED INDEX IX_outbox_perintah     ON dbo.outbox (kode DESC) WHERE is_perintah = 1;
CREATE NONCLUSTERED INDEX IX_outbox_tgl_perintah ON dbo.outbox (tgl_entri DESC, kode DESC) WHERE is_perintah = 1;

-- outbox: reply ke reseller (is_perintah = 0)
CREATE NONCLUSTERED INDEX IX_outbox_reply     ON dbo.outbox (kode DESC) WHERE kode_reseller IS NOT NULL AND is_perintah = 0;
CREATE NONCLUSTERED INDEX IX_outbox_tgl_reply ON dbo.outbox (tgl_entri DESC, kode DESC) WHERE kode_reseller IS NOT NULL AND is_perintah = 0;
```

Setelah dibuat, jalankan update statistik: `UPDATE STATISTICS dbo.inbox; UPDATE STATISTICS dbo.outbox;`

> Tanpa index tersebut, query varian/flag di MSSQL akan tetap scan (tidak di-fast-path).

## Catatan

- Jangan drop oleh index yang ditandai "dipakai bisection" tanpa menyesuaikan logika repo.
- `ANALYZE` sebaiknya dijalankan setelah mass-change data agar perencana segar.
- Fitur sort non-trivial di FE masih *open* — bila dibutuhkan, inventaris index per kolom sort (keyset) diputuskan dulu, bukan otomatis buat.

## Kebijakan hak tulis atas database

- Kode aplikasi **boleh** melakukan `INSERT`/`UPDATE` — itu memang tugasnya, dan endpoint-nya dilindungi `PostgresWriteGuard` yang dikendalikan `POSTGRES_WRITE_ENABLED` (default `false`).
- Siapa pun yang mengerjakan perubahan kode dan tes **hanya boleh membaca**. Tidak pernah menjalankan `CREATE`, `ALTER`, `DROP`, `TRUNCATE`, `GRANT`, `REVOKE`, migrasi, seed, backfill, atau pembuatan index.
- **Semua index adalah tanggung jawab manusia pemilik database.** Blok SQL di bagian MSSQL di atas adalah referensi, bukan sesuatu yang dieksekusi otomatis. Pengurutan di dalam kode murni Go; tidak ada index yang dibuat atau diubah oleh kode maupun tes.
- Tes di repo ini tidak boleh menulis ke database. Aturan itu dijaga secara statis oleh `internal/repository/db_readonly_guard_test.go`: seluruh `*_test.go` dipindai dan gagal bila memuat SQL tulis, DDL, pemanggilan `Exec`/`Begin`, string koneksi nyata, pemanggilan tulis repository, atau berkas `.sql`. Pengecualian hanya lewat daftar putih yang wajib menyertakan alasan, dan entri yang sudah tidak relevan akan menggagalkan tes.
- Penutupan koneksi (`Close`, `CloseAll`) aman: itu hanya melepas handle sisi klien milik proses ini dan tidak mengubah data, skema, atau sesi aplikasi lain.
- Test integrasi adalah satu-satunya tes yang menyentuh server, dijaga `INTEGRATION_DB=1`, dan hanya menjalankan pembacaan. Jalankan dari akar repo setelah memuat `.env` ke lingkungan proses.