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

1. **Pagination default = `ORDER BY kode DESC`** (terbaru dulu). Kosongkan seluruh `tgl_entri` dukungan bisection & index terarah.
2. **Bisection** (`kode_cut`) memanfaatkan `idx_*_kode_tgl (kode DESC INCLUDE tgl_entri)` + slack 50k.
3. **Partial jawaban/perintah** duduk untuk filter triase (`requestFromReseller` / `replyToReseller` + `is_jawaban=0` / `is_perintah=0`); versi `tgl_entri DESC` digunakan saat filter tanggal + triase.
4. Sort kolom di luar `kode DESC` **belum** didukung (butuh keputusan & index per kolom — lihat catatan).

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