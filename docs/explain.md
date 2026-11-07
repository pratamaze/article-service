# Bukti EXPLAIN `ListArticles`

Verifikasi dilakukan pada 8 November 2026 dengan `make psql`, menggunakan
`EXPLAIN (ANALYZE, BUFFERS)`. Keempat statement di bawah memakai bentuk SQL
yang sama dengan statement yang dirakit `Repository.ListArticles`: kolom,
filter, keyset cursor, urutan, dan `LIMIT`. Statement diprepare agar placeholder
`$n` tetap sama seperti pada aplikasi. Nilai limit yang dieksekusi adalah 21
(limit klien 20 ditambah satu untuk `has_more`). Tidak ada perubahan data dan
tidak menggunakan `SET enable_seqscan = off`.

## Pengukuran awal (dengan cursor)

### Tanpa filter

```sql
SELECT id, author_id, title, left(body, 200), created_at FROM articles
WHERE (created_at, id) < ($1, $2)
ORDER BY created_at DESC, id DESC LIMIT $3;
```

Dengan cursor `2026-11-08 00:00:00+00` dan id `ZZZZZZZZZZZZZZZZZZZZZZZZZZ`,
plan aktual memakai `Index Scan using idx_articles_created`, mengembalikan 21
baris dalam 0.132 ms (7 shared-buffer hits).

### Filter satu author

```sql
SELECT id, author_id, title, left(body, 200), created_at FROM articles
WHERE author_id = $1 AND (created_at, id) < ($2, $3)
ORDER BY created_at DESC, id DESC LIMIT $4;
```

Dengan `author-014536` dan cursor yang sama, plan aktual memakai `Index Scan
using idx_articles_author_created`, mengembalikan 21 baris dalam 0.269 ms
(7 hits, 17 reads).

### Filter pencarian

```sql
SELECT id, author_id, title, left(body, 200), created_at FROM articles
WHERE search_vector @@ websearch_to_tsquery('simple', $1)
  AND (created_at, id) < ($2, $3)
ORDER BY created_at DESC, id DESC LIMIT $4;
```

Dengan query `hidrokarbon` dan cursor yang sama, plan aktual memakai `Bitmap
Index Scan on idx_articles_search`, lalu `Bitmap Heap Scan` dan sort untuk
urutan keyset. Hasilnya 10 baris, 0.151 ms, dengan 20 shared-buffer hits.

### Filter author dan pencarian

```sql
SELECT id, author_id, title, left(body, 200), created_at FROM articles
WHERE author_id = $1
  AND search_vector @@ websearch_to_tsquery('simple', $2)
  AND (created_at, id) < ($3, $4)
ORDER BY created_at DESC, id DESC LIMIT $5;
```

Dengan `author-giant`, query `hidrokarbon`, dan cursor yang sama, plan aktual
memakai `Bitmap Index Scan on idx_articles_search`; filter author dan cursor
diterapkan pada heap scan, lalu hasil diurutkan. Hasilnya 4 baris, 0.088 ms,
dan 14 shared-buffer hits. Untuk kombinasi ini PostgreSQL memilih index
full-text, bukan `idx_articles_author_created`, berdasarkan parameter aktual.

## Matriks ukur (halaman pertama)

Setiap statement memakai SQL `ListArticles` halaman pertama dengan nilai literal
dan `LIMIT 21`. Waktu dingin adalah eksekusi pertama dan waktu hangat adalah
eksekusi kedua dari pengukuran yang sama.

| Kasus | Node utama | Index yang dipakai | Jumlah baris | Waktu dingin (ms) | Waktu hangat (ms) |
|---|---|---|---:|---:|---:|
| author-giant, tanpa query | Index Scan | `idx_articles_created` | 21 | 0.147 | 0.092 |
| author-000100, tanpa query | Index Scan | `idx_articles_author_created` | 21 | 8.660 | 0.119 |
| author-giant, query `kebijakan` | Index Scan | `idx_articles_created` | 21 | 2.519 | 2.644 |
| author-000100, query `kebijakan` | Sort → Bitmap Heap Scan | `idx_articles_author_created`, `idx_articles_search` | 0 | 2.837 | 1.965 |
| author-giant, query `berita` | Index Scan | `idx_articles_created` | 21 | 0.136 | 0.079 |
| tanpa author, query `berita` | Index Scan | `idx_articles_created` | 21 | 0.051 | 0.046 |

Kasus 4 menghasilkan 0 baris pada kedua eksekusi dan memakai dua bitmap index
scan. Lima kasus lain memakai `idx_articles_created` atau
`idx_articles_author_created`; tidak ada `Seq Scan` pada enam plan ini.

## Generic plan (opsional)

Untuk statement kasus 3 yang diprepare dengan parameter `$1` untuk author,
node utama eksekusi ke-1 adalah `Index Scan using idx_articles_created`.
Pada eksekusi ke-7 node utama berubah menjadi `Sort` di atas `Bitmap Heap Scan`
dengan `BitmapAnd`; karena itu node utama berubah.

## Temuan

Tidak ada `Seq Scan`, node `Sort` dengan lebih dari 10.000 baris, atau
`Execution Time` di atas 300 ms pada enam kasus matriks halaman pertama.

## Catatan

`EXPLAIN` hanya membuktikan plan pada data, statistik, dan parameter saat
dijalankan. Angka waktu dan buffer di atas bukan jaminan performa tetap.
