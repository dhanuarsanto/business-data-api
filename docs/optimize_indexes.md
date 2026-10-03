# Index — Inbox & Outbox

DDL lengkap untuk membuat SEMUA index dari nol. Jalankan manual di database.

## PostgreSQL — Inbox

```sql
-- PRIMARY KEY
CREATE UNIQUE INDEX pk_inbox ON staging.inbox USING btree (kode);

-- Bisection (cutStartPG / cutEndPG) — WAJIB
CREATE INDEX idx_inbox_kode_tgl ON staging.inbox (kode DESC) INCLUDE (tgl_entri);

-- Jalur tgl-leading
CREATE INDEX idx_inbox_tgl_entri ON staging.inbox (tgl_entri DESC, kode DESC);

-- Filter equality / dropdown
CREATE INDEX idx_inbox_status ON staging.inbox (status);
CREATE INDEX idx_inbox_tipe ON staging.inbox (tipe_pengirim) WHERE tipe_pengirim IS NOT NULL;
CREATE INDEX idx_inbox_terminal ON staging.inbox (kode_terminal) WHERE kode_terminal IS NOT NULL;
CREATE INDEX idx_inbox_kode_reseller ON staging.inbox (kode_reseller);
CREATE INDEX idx_inbox_status_tipe ON staging.inbox (status, tipe_pengirim);

-- ILIKE pencarian teks (butuh pg_trgm)
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_inbox_pesan_trgm ON staging.inbox USING GIN (pesan gin_trgm_ops);
CREATE INDEX idx_inbox_pengirim_trgm ON staging.inbox USING GIN (pengirim gin_trgm_ops);

-- Triase: jawaban provider (is_jawaban = 1)
CREATE INDEX idx_inbox_tgl_entri_jawaban ON staging.inbox (tgl_entri DESC, kode DESC) WHERE is_jawaban = 1;
CREATE INDEX idx_inbox_jawaban ON staging.inbox (kode DESC) WHERE is_jawaban = 1;

-- Triase: request reseller (kode_reseller IS NOT NULL AND is_jawaban = 0)
CREATE INDEX idx_inbox_tgl_entri_kode_reseller_jawaban ON staging.inbox (tgl_entri DESC, kode DESC)
  WHERE kode_reseller IS NOT NULL AND is_jawaban = 0;
CREATE INDEX idx_inbox_kode_reseller_jawaban ON staging.inbox (kode DESC)
  WHERE kode_reseller IS NOT NULL AND is_jawaban = 0;

-- lookup tgl_status
CREATE INDEX idx_inbox_tgl_status ON staging.inbox (tgl_status);
```

## PostgreSQL — Outbox

```sql
-- PRIMARY KEY
CREATE UNIQUE INDEX pk_outbox ON staging.outbox USING btree (kode);

-- Bisection (cutStartPG / cutEndPG) — WAJIB
CREATE INDEX idx_outbox_kode_tgl ON staging.outbox (kode DESC) INCLUDE (tgl_entri);

-- Jalur tgl-leading
CREATE INDEX idx_outbox_tgl_entri ON staging.outbox (tgl_entri DESC, kode DESC);

-- Filter equality / dropdown
CREATE INDEX idx_outbox_status ON staging.outbox (status);
CREATE INDEX idx_outbox_tipe ON staging.outbox (tipe_penerima) WHERE tipe_penerima IS NOT NULL;
CREATE INDEX idx_outbox_kode_reseller ON staging.outbox (kode_reseller);
CREATE INDEX ix_kode_transaksi ON staging.outbox (kode_transaksi) WHERE kode_transaksi IS NOT NULL;
CREATE INDEX ix_kode_inbox ON staging.outbox (kode_inbox) WHERE kode_inbox IS NOT NULL;

-- ILIKE pencarian teks
CREATE INDEX idx_outbox_pesan_trgm ON staging.outbox USING GIN (pesan gin_trgm_ops);
CREATE INDEX idx_outbox_penerima_trgm ON staging.outbox USING GIN (penerima gin_trgm_ops);

-- Triase: perintah provider (is_perintah = 1)
CREATE INDEX idx_outbox_tgl_entri_perintah ON staging.outbox (tgl_entri DESC, kode DESC) WHERE is_perintah = 1;
CREATE INDEX idx_outbox_perintah ON staging.outbox (kode DESC) WHERE is_perintah = 1;

-- Triase: reply reseller (kode_reseller IS NOT NULL AND is_perintah = 0)
CREATE INDEX idx_outbox_tgl_entri_kode_reseller_perintah ON staging.outbox (tgl_entri DESC, kode DESC)
  WHERE kode_reseller IS NOT NULL AND is_perintah = 0;
CREATE INDEX idx_outbox_kode_reseller_perintah ON staging.outbox (kode DESC)
  WHERE kode_reseller IS NOT NULL AND is_perintah = 0;

-- lookup tgl_status
CREATE INDEX idx_outbox_tgl_status ON staging.outbox (tgl_status);
```

## MSSQL — Inbox

```sql
-- PRIMARY KEY (CLUSTERED)
CREATE UNIQUE CLUSTERED INDEX PK__inbox ON dbo.inbox (kode);

-- Bisection (cutStartPG / cutEndPG) — WAJIB
CREATE NONCLUSTERED INDEX IX_inbox_kode_tgl   ON dbo.inbox  (kode DESC) INCLUDE (tgl_entri);

-- tgl-leading
CREATE NONCLUSTERED INDEX IX_inbox_tgl        ON dbo.inbox  (kode DESC, tgl_entri DESC) INCLUDE (tipe_pengirim, status);

-- Filter equality / dropdown
CREATE NONCLUSTERED INDEX IX_inbox_status     ON dbo.inbox  (status);
CREATE NONCLUSTERED INDEX IX_inbox_tipe       ON dbo.inbox  (tipe_pengirim) WHERE tipe_pengirim IS NOT NULL;
CREATE NONCLUSTERED INDEX IX_inbox_terminal   ON dbo.inbox  (kode_terminal) WHERE kode_terminal IS NOT NULL;
CREATE NONCLUSTERED INDEX IX_inbox_kode_reseller ON dbo.inbox (kode_reseller);
CREATE NONCLUSTERED INDEX IX_inbox_status_tipe ON dbo.inbox (status, tipe_pengirim);

-- Pencarian teks (full-text search; trigram tidak ada)
-- CREATE FULLTEXT CATALOG ftCatalog AS DEFAULT;
-- CREATE FULLTEXT INDEX ON dbo.inbox (pesan, pengirim) KEY INDEX PK__inbox;

-- Triase: jawaban provider (is_jawaban = 1)
CREATE NONCLUSTERED INDEX IX_inbox_jawaban ON dbo.inbox (kode DESC) WHERE is_jawaban = 1;

-- Triase: request reseller (kode_reseller IS NOT NULL AND is_jawaban = 0)
CREATE NONCLUSTERED INDEX IX_inbox_request ON dbo.inbox (kode DESC)
  WHERE kode_reseller IS NOT NULL AND is_jawaban = 0;

-- Lookup relasional
CREATE NONCLUSTERED INDEX IX_inbox_kode_transaksi ON dbo.inbox (kode_transaksi) WHERE kode_transaksi IS NOT NULL;
```

## MSSQL — Outbox

```sql
-- PRIMARY KEY (CLUSTERED)
CREATE UNIQUE CLUSTERED INDEX PK__outbox ON dbo.outbox (kode);

-- Bisection (cutStartPG / cutEndPG) — WAJIB
CREATE NONCLUSTERED INDEX IX_outbox_kode_tgl  ON dbo.outbox (kode DESC) INCLUDE (tgl_entri);

-- tgl-leading
CREATE NONCLUSTERED INDEX IX_outbox_tgl       ON dbo.outbox (kode DESC, tgl_entri DESC) INCLUDE (tipe_penerima, status);

-- Filter equality / dropdown
CREATE NONCLUSTERED INDEX IX_outbox_status    ON dbo.outbox (status);
CREATE NONCLUSTERED INDEX IX_outbox_tipe      ON dbo.outbox (tipe_penerima) WHERE tipe_penerima IS NOT NULL;
CREATE NONCLUSTERED INDEX IX_outbox_kode_reseller ON dbo.outbox (kode_reseller);

-- Pencarian teks (full-text)
-- CREATE FULLTEXT INDEX ON dbo.outbox (pesan, penerima) KEY INDEX PK__outbox;

-- Triase: perintah provider (is_perintah = 1)
CREATE NONCLUSTERED INDEX IX_outbox_perintah ON dbo.outbox (kode DESC) WHERE is_perintah = 1;

-- Triase: reply reseller (kode_reseller IS NOT NULL AND is_perintah = 0)
CREATE NONCLUSTERED INDEX IX_outbox_reply ON dbo.outbox (kode DESC)
  WHERE kode_reseller IS NOT NULL AND is_perintah = 0;

-- Lookup relasional
CREATE NONCLUSTERED INDEX IX_outbox_kode_transaksi ON dbo.outbox (kode_transaksi) WHERE kode_transaksi IS NOT NULL;
CREATE NONCLUSTERED INDEX IX_outbox_kode_inbox ON dbo.outbox (kode_inbox) WHERE kode_inbox IS NOT NULL;
```

## Catatan Penting

- Index bisection (`idx_*_kode_tgl` / `IX_*_kode_tgl`) dipakai logika repo — **jangan drop** tanpa ubah kode
- PostgreSQL: butuh `CREATE EXTENSION pg_trgm` untuk GIN trigram
- MSSQL: tidak ada trigram; gunakan full-text search untuk `LIKE '%...%'`
- Setelah pasang semua: `ANALYZE` (PG) / `UPDATE STATISTICS` (MSSQL)
- Sort selalu `ORDER BY kode DESC` di sisi Go
