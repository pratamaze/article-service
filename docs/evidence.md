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

## Nama kembar dan resolusi nama

Pengukuran berikut memakai SQL `ResolveAuthorIDs` dan `ListArticles` halaman
pertama dengan literal serta `LIMIT 21`, lalu menjalankan setiap statement dua
kali. Untuk resolusi nama, setiap pasangan angka pada sel menyatakan
`syndication` lalu `budi santoso`.

| Kasus | Node utama | Index | Jumlah baris | Waktu pertama (ms) | Waktu kedua (ms) |
|---|---|---|---:|---:|---:|
| Resolusi nama `syndication` / `budi santoso` | Sort → Index Scan | `idx_authors_name_norm` | 2 / 3 | 0.085 / 0.021 | 0.028 / 0.015 |
| `Syndication` (giant + kecil), tanpa query | Index Scan | `idx_articles_created` | 21 | 3280.261 | 534.102 |
| `Syndication` (giant + kecil), query `kebijakan` | Index Scan | `idx_articles_created` | 21 | 865.649 | 506.931 |
| `Budi Santoso` (3 author kecil), tanpa query | Sort → Index Scan | `idx_articles_author_created` | 21 | 1.960 | 0.239 |
| `author-giant`, cursor baris ke-10000 | Index Scan | `idx_articles_author_created` | 21 | 0.339 | 0.062 |

Resolusi `syndication` mengembalikan `author-000001` dan `author-giant`,
sedangkan `budi santoso` mengembalikan `author-000002`, `author-000003`, dan
`author-000004`. Kedua resolusi memakai `idx_authors_name_norm`. Pada dua list
`Syndication`, `idx_articles_created` menyaring 434464 dan 701443 baris.

### Pemeriksaan prepared statement kasus `Syndication` tanpa query

Statement dengan parameter array author dijalankan tujuh kali dengan nilai
`['author-000001','author-giant']`; kolom waktu adalah `Execution Time`.

| Pengaturan | Eksekusi 1: node utama | Waktu 1 (ms) | Eksekusi 7: node utama | Waktu 7 (ms) |
|---|---|---:|---|---:|
| `force_custom_plan` | Index Scan `idx_articles_created` | 282.466 | Index Scan `idx_articles_created` | 263.624 |
| tanpa pengaturan (`RESET plan_cache_mode`) | Index Scan `idx_articles_created` | 262.786 | Index Scan `idx_articles_created` | 252.015 |

Dalam pengukuran ini, node utama pada eksekusi pertama dan ketujuh sama pada
kedua pengaturan.

## Generic plan (opsional)

Untuk statement kasus 3 yang diprepare dengan parameter `$1` untuk author,
node utama eksekusi ke-1 adalah `Index Scan using idx_articles_created`.
Pada eksekusi ke-7 node utama berubah menjadi `Sort` di atas `Bitmap Heap Scan`
dengan `BitmapAnd`; karena itu node utama berubah.

## Temuan

### Pengukuran API tambahan (8 November 2026)

Pengukuran berikut dijalankan dengan `make psql`, memakai literal, halaman
pertama, dan `LIMIT 21`. Masing-masing kasus dijalankan dua kali.

| Kasus | Node utama dan index | Baris | Eksekusi 1 (ms) | Eksekusi 2 (ms) |
|---|---|---:|---:|---:|
| author-giant, `zzxqv` | Sort → Bitmap Heap Scan; `idx_articles_search` | 0 | 0.678 | 0.018 |
| author-giant, `hidrokarbon` | Sort → Bitmap Heap Scan; `idx_articles_search` | 4 | 0.062 | 0.030 |

Prepared statement `author-giant` + `kebijakan` memakai `Index Scan using
idx_articles_created` pada eksekusi pertama (2.768 ms). Tanpa pengaturan plan
cache, eksekusi ke-6 dan ke-7 berubah ke `Sort → Bitmap Heap Scan` dengan
`BitmapAnd`; waktu eksekusi ke-7 adalah 77.426 ms (ke-6 94.358 ms). Setelah
`SET plan_cache_mode = force_custom_plan`, eksekusi pertama (1.840 ms) dan
ke-7 (2.223 ms) sama-sama memakai `Index Scan using idx_articles_created`.

Tidak ada `Seq Scan`, node `Sort` dengan lebih dari 10.000 baris, atau
`Execution Time` di atas 300 ms pada enam kasus matriks halaman pertama.

Pengukuran nama kembar menambahkan dua kasus di atas 300 ms: `Syndication`
tanpa query sebesar 3280.261 ms lalu 534.102 ms, serta `Syndication` dengan
`kebijakan` sebesar 865.649 ms lalu 506.931 ms. Keduanya memakai `Index Scan
idx_articles_created`; tidak ada `Seq Scan`. Sort untuk tiga author `Budi
Santoso` menerima 83 baris sebelum `LIMIT`, bukan lebih dari 10.000 baris.

Pengukuran prepared statement di atas berubah plan pada eksekusi ke-6 dan
ke-7. Karena itu API mengatur `plan_cache_mode=force_custom_plan`; kesimpulan
ini hanya berlaku untuk data, statistik, dan parameter yang diukur.

## Catatan

`EXPLAIN` hanya membuktikan plan pada data, statistik, dan parameter saat
dijalankan. Angka waktu dan buffer di atas bukan jaminan performa tetap.

## Raw output

````text
# Output mentah EXPLAIN halaman pertama

Diambil melalui `make psql` pada 8 November 2026. Tiap blok memuat run 1 lalu
run 2, dengan SQL literal yang sama dengan cabang `ListArticles` tanpa cursor
dan `LIMIT 21`.

## 1. author-giant, tanpa query

```text
run 1
Limit  (cost=0.42..25.20 rows=21 width=117) (actual time=0.031..0.118 rows=21 loops=1)
  Buffers: shared hit=13
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..352599.77 rows=298924 width=117) (actual time=0.029..0.115 rows=21 loops=1)
        Filter: (author_id = 'author-giant'::text)
        Rows Removed by Filter: 43
        Buffers: shared hit=13
Planning:
  Buffers: shared hit=181
Planning Time: 0.667 ms
Execution Time: 0.147 ms

run 2
Limit  (cost=0.42..25.20 rows=21 width=117) (actual time=0.011..0.076 rows=21 loops=1)
  Buffers: shared hit=13
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..352599.77 rows=298924 width=117) (actual time=0.011..0.074 rows=21 loops=1)
        Filter: (author_id = 'author-giant'::text)
        Rows Removed by Filter: 43
        Buffers: shared hit=13
Planning Time: 0.068 ms
Execution Time: 0.092 ms
```

## 2. author-000100, tanpa query

```text
run 1
Limit  (cost=0.42..86.72 rows=21 width=117) (actual time=0.083..8.643 rows=21 loops=1)
  Buffers: shared hit=2 read=22
  ->  Index Scan using idx_articles_author_created on articles  (cost=0.42..152.47 rows=37 width=117) (actual time=0.082..8.636 rows=21 loops=1)
        Index Cond: (author_id = 'author-000100'::text)
        Buffers: shared hit=2 read=22
Planning Time: 0.044 ms
Execution Time: 8.660 ms

run 2
Limit  (cost=0.42..86.72 rows=21 width=117) (actual time=0.033..0.099 rows=21 loops=1)
  Buffers: shared hit=24
  ->  Index Scan using idx_articles_author_created on articles  (cost=0.42..152.47 rows=37 width=117) (actual time=0.032..0.097 rows=21 loops=1)
        Index Cond: (author_id = 'author-000100'::text)
        Buffers: shared hit=24
Planning Time: 0.131 ms
Execution Time: 0.119 ms
```

## 3. author-giant, query kebijakan

```text
run 1
Limit  (cost=0.42..2386.32 rows=21 width=117) (actual time=0.127..2.497 rows=21 loops=1)
  Buffers: shared hit=937
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..354362.41 rows=3119 width=117) (actual time=0.126..2.494 rows=21 loops=1)
        Filter: ((search_vector @@ '''kebijakan'''::tsquery) AND (author_id = 'author-giant'::text))
        Rows Removed by Filter: 6166
        Buffers: shared hit=937
Planning:
  Buffers: shared hit=30
Planning Time: 0.682 ms
Execution Time: 2.519 ms

run 2
Limit  (cost=0.42..2386.32 rows=21 width=117) (actual time=0.104..2.623 rows=21 loops=1)
  Buffers: shared hit=937
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..354362.41 rows=3119 width=117) (actual time=0.103..2.619 rows=21 loops=1)
        Filter: ((search_vector @@ '''kebijakan'''::tsquery) AND (author_id = 'author-giant'::text))
        Rows Removed by Filter: 6166
        Buffers: shared hit=937
Planning:
  Buffers: shared hit=1
Planning Time: 0.149 ms
Execution Time: 2.644 ms
```

## 4. author-000100, query kebijakan

```text
run 1
Limit  (cost=90.64..90.65 rows=1 width=117) (actual time=2.774..2.776 rows=0 loops=1)
  Buffers: shared hit=13 read=5
  ->  Sort  (cost=90.64..90.65 rows=1 width=117) (actual time=2.773..2.774 rows=0 loops=1)
        Sort Key: created_at DESC, id DESC
        Sort Method: quicksort  Memory: 25kB
        Buffers: shared hit=13 read=5
        ->  Bitmap Heap Scan on articles  (cost=86.62..90.63 rows=1 width=117) (actual time=2.733..2.734 rows=0 loops=1)
              Recheck Cond: ((author_id = 'author-000100'::text) AND (search_vector @@ '''kebijakan'''::tsquery))
              Buffers: shared hit=7 read=5
              ->  BitmapAnd  (cost=86.62..86.62 rows=1 width=0) (actual time=2.730..2.731 rows=0 loops=1)
                    Buffers: shared hit=7 read=5
                    ->  Bitmap Index Scan on idx_articles_author_created  (cost=0.00..4.70 rows=37 width=0) (actual time=0.026..0.026 rows=35 loops=1)
                          Index Cond: (author_id = 'author-000100'::text)
                          Buffers: shared hit=3
                    ->  Bitmap Index Scan on idx_articles_search  (cost=0.00..81.66 rows=10442 width=0) (actual time=2.674..2.674 rows=10000 loops=1)
                          Index Cond: (search_vector @@ '''kebijakan'''::tsquery)
                          Buffers: shared hit=4 read=5
Planning:
  Buffers: shared hit=1
Planning Time: 0.158 ms
Execution Time: 2.837 ms

run 2
Limit  (cost=90.64..90.65 rows=1 width=117) (actual time=1.940..1.941 rows=0 loops=1)
  Buffers: shared hit=12
  ->  Sort  (cost=90.64..90.65 rows=1 width=117) (actual time=1.938..1.940 rows=0 loops=1)
        Sort Key: created_at DESC, id DESC
        Sort Method: quicksort  Memory: 25kB
        Buffers: shared hit=12
        ->  Bitmap Heap Scan on articles  (cost=86.62..90.63 rows=1 width=117) (actual time=1.933..1.934 rows=0 loops=1)
              Recheck Cond: ((author_id = 'author-000100'::text) AND (search_vector @@ '''kebijakan'''::tsquery))
              Buffers: shared hit=12
              ->  BitmapAnd  (cost=86.62..86.62 rows=1 width=0) (actual time=1.931..1.932 rows=0 loops=1)
                    Buffers: shared hit=12
                    ->  Bitmap Index Scan on idx_articles_author_created  (cost=0.00..4.70 rows=37 width=0) (actual time=0.020..0.021 rows=35 loops=1)
                          Index Cond: (author_id = 'author-000100'::text)
                          Buffers: shared hit=3
                    ->  Bitmap Index Scan on idx_articles_search  (cost=0.00..81.66 rows=10442 width=0) (actual time=1.833..1.834 rows=10000 loops=1)
                          Index Cond: (search_vector @@ '''kebijakan'''::tsquery)
                          Buffers: shared hit=9
Planning:
  Buffers: shared hit=1
Planning Time: 0.178 ms
Execution Time: 1.965 ms
```

## 5. author-giant, query berita

```text
run 1
Limit  (cost=0.42..62.81 rows=21 width=117) (actual time=0.025..0.125 rows=21 loops=1)
  Buffers: shared hit=28
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..354653.07 rows=119380 width=117) (actual time=0.024..0.122 rows=21 loops=1)
        Filter: ((search_vector @@ '''berita'''::tsquery) AND (author_id = 'author-giant'::text))
        Rows Removed by Filter: 153
        Buffers: shared hit=28
Planning:
  Buffers: shared hit=1
Planning Time: 0.126 ms
Execution Time: 0.136 ms

run 2
Limit  (cost=0.42..62.81 rows=21 width=117) (actual time=0.013..0.072 rows=21 loops=1)
  Buffers: shared hit=28
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..354653.07 rows=119380 width=117) (actual time=0.012..0.070 rows=21 loops=1)
        Filter: ((search_vector @@ '''berita'''::tsquery) AND (author_id = 'author-giant'::text))
        Rows Removed by Filter: 153
        Buffers: shared hit=28
Planning:
  Buffers: shared hit=1
Planning Time: 0.067 ms
Execution Time: 0.079 ms
```

## 6. tanpa author, query berita

```text
run 1
Limit  (cost=0.42..18.96 rows=21 width=117) (actual time=0.007..0.044 rows=21 loops=1)
  Buffers: shared hit=11
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..352851.74 rows=399711 width=117) (actual time=0.007..0.043 rows=21 loops=1)
        Filter: (search_vector @@ '''berita'''::tsquery)
        Rows Removed by Filter: 33
        Buffers: shared hit=11
Planning:
  Buffers: shared hit=1
Planning Time: 0.071 ms
Execution Time: 0.051 ms

run 2
Limit  (cost=0.42..18.96 rows=21 width=117) (actual time=0.007..0.040 rows=21 loops=1)
  Buffers: shared hit=11
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..352851.74 rows=399711 width=117) (actual time=0.006..0.038 rows=21 loops=1)
        Filter: (search_vector @@ '''berita'''::tsquery)
        Rows Removed by Filter: 33
        Buffers: shared hit=11
Planning:
  Buffers: shared hit=1
Planning Time: 0.048 ms
Execution Time: 0.046 ms
```
````

## 2. Concurrency

TODO: bukti belum dibuat.

## 3. Search & Filtering

Kecepatan pencarian sudah terlihat pada pengukuran EXPLAIN: kombinasi
`author-giant` dan `kebijakan` pada matriks awal memakai `Index Scan
idx_articles_created` dan selesai dalam 2.519 ms lalu 2.644 ms; pada nama
`Syndication` kombinasi yang sama selesai dalam 865.649 ms lalu 506.931 ms.

Akurasi berikut diverifikasi oleh `TestIntegrationI7MixedLanguageSearch` pada
`make test`. Tiga artikel ID, EN, dan campuran memakai satu `author_id` khusus,
sehingga setiap query hanya menguji tiga artikel tersebut.

| Query | Hasil yang ditemukan |
|---|---|
| `banjir` | ID |
| `flood` | EN |
| `startup` | Campuran |
| `program` | ID, EN, Campuran |
| `jakarta`, `JAKARTA` | ID, EN |
| `meluncurkan` | ID, Campuran |
| `minister` | Campuran |
| `banjir jakarta` | ID |
| `meluncur`, `launch`, `flooded` | Tidak ada |
| tiga spasi | ID, EN, Campuran |

Konfigurasi `simple` tidak melakukan stemming: `meluncur` tidak cocok dengan
`meluncurkan`, `launch` tidak cocok dengan `launches`, dan `flooded` tidak
cocok dengan `flood` atau `flooding`. Ini adalah trade-off yang disengaja dan
dibuktikan oleh I7, bukan kegagalan test.

## 4. Extensibility & Maintainability

TODO: bukti belum dibuat.
