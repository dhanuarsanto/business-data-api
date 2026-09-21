# Rekomendasi Index — Inbox & Outbox

Dibuat setelah audit langsung ke `pg_indexes`/`pg_stats` (PostgreSQL) dan `sys.indexes` (MSSQL).
Semua DDL di bawah dieksekusi MANUAL di database masing-masing.

## Kondisi saat ini

| DB | Tabel | Index yang sudah ada |
|---|---|---|
| PostgreSQL | inbox | `pk_inbox (kode)`, `idx_inbox_status (partial status<20)`, `idx_inbox_tgl_status` |
| PostgreSQL | outbox | `pk_outbox (kode)`, `ix_kode_inbox`, `ix_kode_transaksi`, `ix_outbox_status`, `ix_outbox_tgl_status` |
| MSSQL | inbox | PK `kode`, `IX_status`, `IX_tgl_status` |
| MSSQL | outbox | PK `kode`, `IX_status`, `IX_tgl_status`, `IX_kode_inbox`, `IX_kode_transaksi` |

Kolom filter yang BELUM di-index: `tgl_entri`, `kode_terminal`, `tipe_pengirim`/`tipe_penerima`, `pengirim`/`penerima`, `pesan`, `kode_reseller`.

## Selektivitas (dari `pg_stats`)

| Kolom | Selektivitas | Prioritas |
|---|---|---|
| `tgl_entri` | sangat unik (±80% baris unik) | P0 |
| `pesan` | sangat unik | P1 (teks) |
| `pengirim`/`penerima` | 134 / 136 nilai | P2 |
| `kode_reseller` | 24–30 nilai | P2 |
| `status` | 10 nilai | sudah partial |
| `kode_terminal`/`tipe` | 1–4 nilai (tidak selektif) | P3 |

## PostgreSQL

### P0 — filter tanggal + urutan `kode DESC`
```sql
CREATE INDEX idx_inbox_tgl  ON inbox  (tgl_entri DESC, kode DESC);
CREATE INDEX idx_outbox_tgl ON outbox (tgl_entri DESC, kode DESC);
```

### P1 — pencarian teks (pesan, pengirim, penerima) via trigram
Kolom teks masih memakai `ILIKE '%..%'`; index btree tidak membantu — wajib `pg_trgm`.
`kode_reseller` TIDAK lagi di sini (sudah jadi dropdown equality, lihat P2).
```sql
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_inbox_pesan_trgm  ON inbox  USING gin (pesan gin_trgm_ops);
CREATE INDEX idx_inbox_pengirim_trgm ON inbox USING gin (pengirim gin_trgm_ops);
CREATE INDEX idx_outbox_pesan_trgm  ON outbox USING gin (pesan gin_trgm_ops);
CREATE INDEX idx_outbox_penerima_trgm ON outbox USING gin (penerima gin_trgm_ops);

-- MIGRASI: reseller kini equality dropdown — trigram lama tidak terpakai, ganti btree
-- DROP INDEX IF EXISTS idx_inbox_reseller_trgm;
-- DROP INDEX IF EXISTS idx_outbox_reseller_trgm;
```

### P2 — filter equality/range dropdown (terminal, tipe, reseller)
```sql
CREATE INDEX idx_inbox_terminal ON inbox  (kode_terminal) WHERE kode_terminal IS NOT NULL;
CREATE INDEX idx_inbox_tipe     ON inbox  (tipe_pengirim) WHERE tipe_pengirim IS NOT NULL;
CREATE INDEX idx_outbox_tipe    ON outbox (tipe_penerima) WHERE tipe_penerima IS NOT NULL;
CREATE INDEX idx_inbox_reseller ON inbox  (kode_reseller);
CREATE INDEX idx_outbox_reseller ON outbox (kode_reseller);
```

## MSSQL

### P0 — filter tanggal + urutan `kode DESC`
```sql
CREATE NONCLUSTERED INDEX IX_inbox_tgl  ON inbox  (tgl_entri DESC, kode DESC);
CREATE NONCLUSTERED INDEX IX_outbox_tgl ON outbox (tgl_entri DESC, kode DESC);
```

### P2 — filter equality dropdown (terminal, tipe, reseller)
```sql
CREATE NONCLUSTERED INDEX IX_inbox_terminal ON inbox (kode_terminal) WHERE kode_terminal IS NOT NULL;
CREATE NONCLUSTERED INDEX IX_inbox_tipe     ON inbox (tipe_pengirim) WHERE tipe_pengirim IS NOT NULL;
CREATE NONCLUSTERED INDEX IX_outbox_tipe    ON outbox (tipe_penerima) WHERE tipe_penerima IS NOT NULL;
CREATE NONCLUSTERED INDEX IX_inbox_reseller ON inbox (kode_reseller);
CREATE NONCLUSTERED INDEX IX_outbox_reseller ON outbox (kode_reseller);
```

## Catatan

- **P0 adalah prioritas utama** — filter TGL paling sering dipakai dan paling selektif; dampak terbesar.
- **Kolom teks yang masih `ILIKE '%..%'`: `pesan`, `pengirim`, `penerima`** — tidak bisa memakai index biasa. PostgreSQL: `pg_trgm`. MSSQL: scan saja, atau Full-Text Search bila nanti terlalu lambat.
- **`kode_reseller` sekarang equality (`=`) / dropdown** (dari tabel `reseller` via `GET /master/reseller`) — butuh **index btree biasa** (P2), bukan trigram. Jika trigram reseller lama (`idx_*_reseller_trgm`) sudah terpasang, drop & ganti seperti di P1.
- **`kode_terminal` / `tipe`** (1–4 nilai, tidak selektif): bantu hanya saat digabung dengan filter `tgl_entri` (bitmap combine); kerjakan terakhir.
- Index menambah overhead kecil pada INSERT/UPDATE — tetap aman untuk volume inbox ±10jt / outbox ±7jt baris.
- Sebelum eksekusi, pastikan `ANALYZE` (PG) berjalan agar statistik segar.