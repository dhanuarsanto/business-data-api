# Index — Inbox & Outbox (Status Aktual)

Dokumen ini mencerminkan **kondisi index aktual yang diverifikasi langsung** ke `pg_indexes` (`pandora_dw.staging`) untuk PostgreSQL dan `sys.indexes` (`PandoraData`) untuk MSSQL. Definisi, nama, dan ukuran di bawah adalah hasil pembacaan katalog, bukan perkiraan.

Semua DDL adalah keputusan manusia pemilik database. Kode aplikasi dan test tidak pernah membuat, mengubah, atau menghapus index.

## Kondisi aktual — PostgreSQL `inbox`

| Index                                       | Definisi aktual                                                                                | Ukuran  | Dipakai untuk                                                     |
| ------------------------------------------- | ---------------------------------------------------------------------------------------------- | ------- | ----------------------------------------------------------------- |
| `pk_inbox`                                  | UNIQUE btree (kode)                                                                            | 215 MB  | `ORDER BY kode DESC LIMIT n` (top-N terbaru)                      |
| `idx_inbox_kode_tgl`                        | btree (kode DESC) INCLUDE (tgl_entri)                                                          | 302 MB  | `cutStartPG` / `cutEndPG` (Index Only Scan) — jangan di-drop      |
| `idx_inbox_tgl_entri`                       | btree (tgl_entri DESC, kode DESC)                                                              | 302 MB  | jalur tgl-leading (subquery `ORDER BY tgl_entri DESC, kode DESC`) |
| `idx_inbox_tgl_status`                      | btree (tgl_status)                                                                             | 215 MB  | lookup tgl_status                                                 |
| `idx_inbox_pesan_trgm`                      | GIN (pesan gin_trgm_ops)                                                                       | 2170 MB | ILIKE pesan — index terbesar di tabel ini                         |
| `idx_inbox_pengirim_trgm`                   | GIN (pengirim gin_trgm_ops)                                                                    | 173 MB  | ILIKE pengirim                                                    |
| `idx_inbox_tgl_entri_jawaban`               | partial btree (tgl_entri DESC, kode DESC) WHERE `is_jawaban = 1`                               | 209 MB  | tanggal + `jawabanFromProvider`                                   |
| `idx_inbox_jawaban`                         | partial btree (kode DESC) WHERE `is_jawaban = 1`                                               | 149 MB  | `jawabanFromProvider` tanpa tanggal                               |
| `idx_inbox_tgl_entri_kode_reseller_jawaban` | partial btree (tgl_entri DESC, kode DESC) WHERE `kode_reseller IS NOT NULL AND is_jawaban = 0` | 93 MB   | tanggal + `requestFromReseller`                                   |
| `idx_inbox_kode_reseller_jawaban`           | partial btree (kode DESC) WHERE `kode_reseller IS NOT NULL AND is_jawaban = 0`                 | 67 MB   | `requestFromReseller` tanpa tanggal                               |
| `idx_inbox_kode_reseller`                   | btree (kode_reseller)                                                                          | 67 MB   | equality reseller (dropdown)                                      |
| `idx_inbox_status`                          | btree (status) polos, tanpa predicate                                                          | 66 MB   | equality status                                                   |
| `idx_inbox_status_tipe`                     | btree (status, tipe_pengirim)                                                                  | 66 MB   | kombinasi dropdown status + tipe                                  |
| `idx_inbox_tipe`                            | btree (tipe_pengirim) WHERE IS NOT NULL                                                        | 66 MB   | dropdown tipe                                                     |
| `idx_inbox_terminal`                        | btree (kode_terminal) WHERE IS NOT NULL                                                        | 48 kB   | dropdown terminal                                                 |

> Trigram `kode_reseller` (`idx_inbox_reseller_trgm`) sudah **DIDROP** — diganti btree equality untuk dropdown.
> `idx_inbox_pesan_trgm` 2170 MB adalah index terbesar di tabel, diimbangi upkeep write yang mahal. Belum di-drop karena keputusan eviction ada di pemilik database.

## Kondisi aktual — PostgreSQL `outbox`

| Index                                         | Definisi aktual                                                                                 | Ukuran     | Dipakai untuk                                                                                  |
| --------------------------------------------- | ----------------------------------------------------------------------------------------------- | ---------- | ---------------------------------------------------------------------------------------------- |
| `pk_outbox`                                   | UNIQUE btree (kode)                                                                             | 22 MB      | `ORDER BY kode DESC LIMIT n`                                                                   |
| `idx_outbox_kode_tgl`                         | btree (kode DESC) INCLUDE (tgl_entri)                                                           | 31 MB      | `cutStartPG` / `cutEndPG` — jangan di-drop                                                     |
| `idx_outbox_tgl_entri`                        | btree (tgl_entri DESC, kode DESC)                                                               | 31 MB      | jalur tgl-leading                                                                              |
| `idx_outbox_pesan_trgm`                       | GIN (pesan gin_trgm_ops)                                                                        | 172 MB     | ILIKE pesan                                                                                    |
| `idx_outbox_penerima_trgm`                    | GIN (penerima gin_trgm_ops)                                                                     | 50 MB      | ILIKE penerima                                                                                 |
| `idx_outbox_tgl_entri_perintah`               | partial btree (tgl_entri DESC, kode DESC) WHERE `is_perintah = 1`                               | 20 MB      | tanggal + `perintahProvider`                                                                   |
| `idx_outbox_perintah`                         | partial btree (kode DESC) WHERE `is_perintah = 1`                                               | 14 MB      | `perintahProvider` tanpa tanggal                                                               |
| `idx_outbox_tgl_entri_kode_reseller_perintah` | partial btree (tgl_entri DESC, kode DESC) WHERE `kode_reseller IS NOT NULL AND is_perintah = 0` | 11 MB      | tanggal + `replyToReseller`                                                                    |
| `idx_outbox_kode_reseller_perintah`           | partial btree (kode DESC) WHERE `kode_reseller IS NOT NULL AND is_perintah = 0`                 | 8040 kB    | `replyToReseller` tanpa tanggal                                                                |
| `idx_outbox_kode_reseller`                    | btree (kode_reseller)                                                                           | 6976 kB    | equality reseller                                                                              |
| `idx_outbox_status`                           | btree (status) polos                                                                            | 6968 kB    | equality status                                                                                |
| `idx_outbox_tipe`                             | btree (tipe_penerima) WHERE IS NOT NULL                                                         | 6968 kB    | dropdown tipe                                                                                  |
| `ix_kode_transaksi`                           | btree (kode_transaksi) WHERE IS NOT NULL                                                        | 16 MB      | lookup relasional                                                                              |
| `ix_kode_inbox`                               | btree (kode_inbox) WHERE IS NOT NULL                                                            | 80 kB      | lookup relasional                                                                              |
| `ix_outbox_tgl_status`                        | btree (tgl_status) WHERE `tgl_status >= '2026-09-01 17:48:43.982'`                              | 8192 bytes | praktis tidak terpakai — predicate tanggal beku, nyaris tak pernah cocok dengan query aplikasi |

> Nama `idx_inbox_*_perintah` di tabel `outbox` terdengar aneh (hasil kreasi manual), tapi menunjuk tabel outbox dengan benar. Penggantian nama = DDL opsional.

## Perilaku yang bergantung pada index ini

1. **Pengurutan hasil = `ORDER BY kode DESC`** (terbaru dulu) di semua jalur.
2. **Batas tanggal via bisection** memakai `idx_*_kode_tgl (kode DESC INCLUDE tgl_entri)` + slack 50k. `cutEndPG` memetakan `tgl_entri <= end` menjadi `kode <= cutEnd + 50000`; `cutStartPG` memetakan `tgl_entri < start` menjadi `kode > cutStart`. Hasil identik secara logika, planner selalu lewat index kode tanpa Seq/Bitmap scan.
3. **Partial triase** melayani `requestFromReseller` (`kode_reseller IS NOT NULL AND is_jawaban = 0`), `jawabanFromProvider` (`is_jawaban = 1`), `replyToReseller` (`kode_reseller IS NOT NULL AND is_perintah = 0`), `perintahProvider` (`is_perintah = 1`). Varian `tgl_entri DESC` dipakai saat filter tanggal ikut menyala. Kedua flag triase saling lepas: menyalakan keduanya berarti tanpa syarat triase, sama seperti tidak menyalakan apa pun.
4. **Jalur tgl-leading** menyubquery `n` baris terbaru dengan `ORDER BY tgl_entri DESC, kode DESC`, lalu query luar mengurutkan `i.kode DESC`. Urutan luar wajib `kode DESC` supaya baris yang sama tidak pernah muncul di dua permintaan berbeda.

## Angka terukur (hasil `EXPLAIN ANALYZE`, 10.051.416 baris inbox / 1.025.931 baris outbox)

| Skenario                                            | Index yang dipilih                    | Execution time      |
| --------------------------------------------------- | ------------------------------------- | ------------------- |
| `inbox` `LIMIT 10000`, tanpa filter                 | `pk_inbox` Index Scan Backward        | 12,7 ms             |
| `inbox` `status = 20`                               | `pk_inbox`                            | 15,4 ms             |
| `inbox` `tipe_pengirim`                             | `idx_inbox_tipe`                      | 0,6 ms              |
| `inbox` `kode_reseller`                             | `pk_inbox`                            | 162 ms              |
| `inbox` `pengirim ILIKE '%52.74%'`                  | `pk_inbox`                            | 256 ms              |
| `inbox` `pesan ILIKE '%TRX Normal%'`                | `pk_inbox`                            | 123 ms              |
| `inbox` batas tanggal (jalur `kode <= cut + slack`) | `pk_inbox`                            | 33,9 ms             |
| `inbox` `requestFromReseller`                       | `idx_inbox_kode_reseller_jawaban`     | 14,4 ms             |
| `inbox` `jawabanFromProvider`                       | `idx_inbox_jawaban`                   | 13,3 ms             |
| `cutEndPG` inbox (`tgl_entri <= ...`)               | `idx_inbox_kode_tgl` Index Only Scan  | 80 ms               |
| `cutStartPG` inbox, `startDate` lama                | `idx_inbox_kode_tgl` Index Only Scan  | 1178 ms             |
| `cutStartPG` / `cutEndPG` outbox                    | `idx_outbox_kode_tgl` Index Only Scan | 0,07 sampai 0,13 ms |

Catatan: untuk `pengirim`/`pesan` dengan `ILIKE '%...%'`, planner memilih top-N lewat `pk_inbox` daripada GIN trigram. Hasil tetap di bawah 300 ms pada `LIMIT 10000`.

**Titik lambat yang diketahui — `cutStartPG` dengan `startDate` lama.** Memindai 634.278 entri index (Heap Fetches 634.279) untuk menemukan satu titik potong, sekitar 1,2 detik. Bottleneck ini bawaan desain bisection. Kalau FE rutin meminta data historis jauh ke belakang, ini yang akan terasa lebih dulu.

Ukuran payload JSON di `LIMIT 10000`: inbox 4,09 MB, outbox 3,26 MB (rata-rata `pesan` 178 byte, maksimum 8.000 byte). Rujukan batas: `MaxLimit = 10000`.

## MSSQL — kondisi aktual dan DDL yang BELUM dieksekusi

FE tidak pernah mengirim `X-DB-Source: mssql`, jadi jalur MSSQL tidak dipakai produksi. Kalau suatu saat dipakai, perlu index berikut lebih dulu.

Index yang benar-benar ada di `PandoraData` (hasil `sys.indexes`) hanya 7:

| Tabel    | Index                                | Definisi                                                     | Dipakai query list?                         |
| -------- | ------------------------------------ | ------------------------------------------------------------ | ------------------------------------------- |
| `inbox`  | `PK__inbox`                          | CLUSTERED (kode)                                             | ya                                          |
| `inbox`  | `IX_status`                          | `status` WHERE `status < 20`                                 | tidak — query tidak membatasi `status < 20` |
| `inbox`  | `IX_tgl_status`                      | `tgl_status` WHERE `tgl_status >= '2026-09-25 16:36:41.442'` | tidak                                       |
| `outbox` | `PK__outbox`                         | CLUSTERED (kode)                                             | ya                                          |
| `outbox` | `IX_status`                          | `status` WHERE `status < 20`                                 | tidak                                       |
| `outbox` | `IX_tgl_status`                      | `tgl_status` WHERE `tgl_status >= '2026-09-25 16:36:41.442'` | tidak                                       |
| `outbox` | `IX_kode_inbox`, `IX_kode_transaksi` | partial, `IS NOT NULL`                                       | hanya lookup relasional                     |

Tidak ada satu pun index `tgl_entri`, trigram, atau partial triase di MSSQL. Konsekuensinya jelas: `TOP(n) ORDER BY kode DESC` sering menjadi clustered scan, filter tanggal menyapu seluruh 10 juta baris, `LIKE '%...%'` menyapu seluruh tabel, dan kedua flag triase juga menyapu seluruh tabel. Inilah alasan suite integrasi `internal/repository` secara historis membutuhkan sekitar 534 detik, hampir semuanya habis di query MSSQL yang menyapu tabel.

DDL referensi (referensi saja, **tidak dieksekusi oleh kode atau test**):

```sql
CREATE NONCLUSTERED INDEX IX_inbox_tgl      ON dbo.inbox  (kode DESC, tgl_entri DESC) INCLUDE (tipe_pengirim, status);
CREATE NONCLUSTERED INDEX IX_outbox_tgl     ON dbo.outbox (kode DESC, tgl_entri DESC) INCLUDE (tipe_penerima, status);
CREATE NONCLUSTERED INDEX IX_inbox_pesan    ON dbo.inbox  (kode DESC) INCLUDE (pesan);
CREATE NONCLUSTERED INDEX IX_inbox_pengirim ON dbo.inbox  (pengirim, kode DESC);
CREATE NONCLUSTERED INDEX IX_outbox_pesan   ON dbo.outbox (kode DESC) INCLUDE (pesan);
CREATE NONCLUSTERED INDEX IX_outbox_penerima ON dbo.outbox (penerima, kode DESC);
CREATE NONCLUSTERED INDEX IX_inbox_jawaban  ON dbo.inbox  (kode DESC) WHERE is_jawaban = 1;
CREATE NONCLUSTERED INDEX IX_inbox_request  ON dbo.inbox  (kode DESC) WHERE kode_reseller IS NOT NULL AND is_jawaban = 0;
CREATE NONCLUSTERED INDEX IX_outbox_perintah ON dbo.outbox (kode DESC) WHERE is_perintah = 1;
CREATE NONCLUSTERED INDEX IX_outbox_reply   ON dbo.outbox (kode DESC) WHERE kode_reseller IS NOT NULL AND is_perintah = 0;
```

Setelah DDL dijalankan: `UPDATE STATISTICS dbo.inbox; UPDATE STATISTICS dbo.outbox;`

## Catatan

- Jangan drop index yang ditandai dipakai bisection (`idx_*_kode_tgl`) tanpa menyesuaikan logika repo.
- `ANALYZE` di PostgreSQL dan `UPDATE STATISTICS` di MSSQL sebaiknya dijalankan setelah perubahan data massal agar perencana segar.
- Sort di luar `kode` dan `tgl_entri` belum didukung — butuh keputusan dan index per kolom.
- Sort selalu murni di sisi Go; tidak ada index yang dibuat atau diubah oleh kode maupun test.
- Batas `MaxLimit = 10000` dipilih karena top-N lewat `pk_inbox` hanya 12,7 ms dan payload 10.000 baris sekitar 4,1 MB. Menaikkan batas menaikkan payload secara linier, sementara waktu query hampir tetap.
