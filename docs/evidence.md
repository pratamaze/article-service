# Bukti EXPLAIN `ListArticles`

Verifikasi dilakukan pada 8 November 2026 dengan `make psql`, menggunakan
`EXPLAIN (ANALYZE, BUFFERS)`. Keempat statement di bawah memakai bentuk SQL
yang sama dengan statement yang dirakit `Repository.ListArticles`: kolom,
filter, keyset cursor, urutan, dan `LIMIT`. Statement diprepare agar placeholder
`$n` tetap sama seperti pada aplikasi. Nilai limit yang dieksekusi adalah 21
(limit klien 20 ditambah satu untuk `has_more`). Tidak ada perubahan data dan
tidak menggunakan `SET enable_seqscan = off`.

## Bentuk data uji saat ini

Pengukuran bagian ini dilakukan pada 8 November 2026. Semua query terhadap database `articles` adalah `SELECT`; jumlah artikel sebelum dan setelah pengukuran serta seed terisolasi adalah 1.000.000. `min(created_at)` adalah `2022-03-02 18:43:20.37845+00` dan `max(created_at)` adalah `2026-11-07 18:42:24.621674+00`.

Tabel berikut berasal dari `width_bucket(extract(epoch FROM created_at), lo, hi, 10)`, dengan `lo` dan `hi` adalah minimum serta maksimum waktu pada `articles`. Bucket 11 berisi tepat satu baris yang sama dengan batas atas; ini adalah perilaku `width_bucket`, bukan kelompok waktu tambahan yang dirancang oleh seed.

| Bucket | Semua artikel | `author-giant` |
|---:|---:|---:|
| 1 | 205583 | 205583 |
| 2 | 94858 | 94858 |
| 3--8 | 0 | 0 |
| 9 | 220936 | 0 |
| 10 | 478622 | 0 |
| 11 (batas atas) | 1 | 0 |

Distribusi per bulan yang benar-benar ada pada hasil query adalah berikut. Nilai nol `author-giant` pada bulan 2026 ditulis eksplisit agar bentuk data tidak tersamarkan oleh `GROUP BY`.

| Bulan | Semua artikel | `author-giant` |
|---|---:|---:|
| 2022-03 | 35073 | 35073 |
| 2022-04 | 36067 | 36067 |
| 2022-05 | 37052 | 37052 |
| 2022-06 | 36261 | 36261 |
| 2022-07 | 37235 | 37235 |
| 2022-08 | 37116 | 37116 |
| 2022-09 | 36219 | 36219 |
| 2022-10 | 37214 | 37214 |
| 2022-11 | 8204 | 8204 |
| 2026-03 | 81808 | 0 |
| 2026-04 | 83933 | 0 |
| 2026-05 | 86949 | 0 |
| 2026-06 | 83738 | 0 |
| 2026-07 | 86765 | 0 |
| 2026-08 | 86884 | 0 |
| 2026-09 | 83781 | 0 |
| 2026-10 | 86786 | 0 |
| 2026-11 | 18915 | 0 |

Pemeriksaan `git diff b0763ae 923e1ba -M --stat -- cmd/seed scripts/seed` menunjukkan perpindahan `scripts/seed/main.go` ke `cmd/seed/main.go` tanpa perubahan baris. Kode seed membuat `created_at` menaik dari waktu mulai hingga rentang 250 hari, dengan jitter dan 5% kemungkinan memakai waktu baris sebelumnya. Untuk setiap artikel, `author-giant` dipilih secara acak independen dengan peluang 30%; selain itu salah satu author ekor dipilih acak.

Seed dijalankan hanya pada `articles_test` yang dibuat ulang melalui `make test-db`, dengan `DATABASE_URL` database tersebut dan `go run ./cmd/seed -n 20000 -seed 42` (tanpa `-reset`). Hasilnya 20.000 artikel dan distribusi giant berikut; bucket 11 kembali hanya baris batas atas.

| Bucket | Semua artikel test | `author-giant` test |
|---:|---:|---:|
| 1 | 2000 | 616 |
| 2 | 2001 | 578 |
| 3 | 2000 | 602 |
| 4 | 1999 | 580 |
| 5 | 2001 | 592 |
| 6 | 1999 | 578 |
| 7 | 2000 | 623 |
| 8 | 2000 | 633 |
| 9 | 2000 | 616 |
| 10 | 1999 | 615 |
| 11 (batas atas) | 1 | 0 |

Jadi, seed saat ini menghasilkan `author-giant` yang tersebar sepanjang waktu (pada seed ini ada di bucket 1--10), sedangkan data `articles` saat ini terkonsentrasi pada waktu paling lama (bucket 1--2). Kapan atau mengapa bentuk data `articles` berubah: **TIDAK DIKETAHUI**.

## Matriks ukur pada data saat ini

SQL setiap kasus persis cabang halaman pertama `Repository.ListArticles`: `SELECT id, author_id, title, left(body, 200), created_at FROM articles`, filter yang sesuai, `ORDER BY created_at DESC, id DESC`, dan `LIMIT 21`. Setiap literal dijalankan dua kali memakai `EXPLAIN (ANALYZE, BUFFERS)`.

| Kasus | Node utama | Index | Jumlah baris | Waktu pertama (ms) | Waktu kedua (ms) |
|---|---|---|---:|---:|---:|
| `author-giant`, tanpa query | Index Scan | `idx_articles_author_created` | 21 | 0.161 | 0.055 |
| `author-giant`, `kebijakan` | Index Scan | `idx_articles_author_created` | 21 | 4.219 | 0.913 |
| `author-giant`, `berita` | Index Scan | `idx_articles_author_created` | 21 | 0.077 | 0.061 |
| `author-giant`, `zzxqv` | Sort → Bitmap Heap Scan | `idx_articles_search` | 0 | 6.266 | 0.975 |
| `author-000100`, tanpa query | Index Scan | `idx_articles_author_created` | 21 | 0.623 | 0.060 |
| `author-000100`, `kebijakan` | Index Scan | `idx_articles_author_created` | 0 | 0.099 | 0.036 |
| tanpa author, `berita` | Index Scan | `idx_articles_created` | 21 | 0.122 | 0.052 |
| tanpa author, tanpa query | Index Scan | `idx_articles_created` | 21 | 0.041 | 0.036 |

### Prepared statement kasus `author-giant` + `kebijakan` pada data saat ini

Statement berparameter untuk kasus kedua dijalankan tujuh kali tanpa pengaturan khusus (`RESET plan_cache_mode`), kemudian tujuh kali lagi dengan `SET plan_cache_mode = force_custom_plan`. Kedua bentuk memakai `LIMIT 21`.

| Pengaturan | Eksekusi 1: node / index / baris | Waktu 1 (ms) | Eksekusi 7: node / index / baris | Waktu 7 (ms) |
|---|---|---:|---|---:|
| default | Index Scan / `idx_articles_author_created` / 21 | 1.551 | Index Scan / `idx_articles_author_created` / 21 | 2.490 |
| `force_custom_plan` | Index Scan / `idx_articles_author_created` / 21 | 0.531 | Index Scan / `idx_articles_author_created` / 21 | 0.509 |

Pada default, estimate pada node `Index Scan` berubah dari 3496 baris pada eksekusi 1 menjadi 1 baris pada eksekusi 6 dan 7; node dan index tidak berubah. Dengan `force_custom_plan`, estimate tetap 3496 pada eksekusi 1--7. Tidak ada `Seq Scan` maupun waktu di atas 300 ms dalam delapan kasus dan pemeriksaan ini.

## Arsip pengukuran pada distribusi data sebelumnya

Seluruh tabel dan raw output setelah heading ini diukur pada distribusi data sebelumnya; **tidak berlaku untuk data saat ini**. Angka dipertahankan sebagai arsip dan tidak dipakai sebagai pembanding dengan matriks aktual di atas.

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

## Matriks ukur lama (halaman pertama; distribusi data sebelumnya)

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

## Nama kembar dan resolusi nama — sebelum perbaikan

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

### Diagnosa dan sesudah perbaikan (langkah 3b)

Pada pemeriksaan ulang 8 November 2026, resolusi `syndication` mengembalikan
`author-000001` (35 artikel) dan `author-giant` (300441 artikel). Jadi
`author-giant` memang termasuk dalam daftar id; tidak ada koreksi terhadap
temuan sebelumnya.

Sebelum kode diubah, SQL repository `author_id = ANY($1::text[])`, halaman
pertama, dan `LIMIT 21` memakai `Index Scan` pada `idx_articles_created`.
Estimate scan adalah 323783 baris, aktual hasil akhir 21 baris, dan 434464
baris dibuang oleh filter. Waktu dua eksekusi adalah 1523.932 ms dan 250.481
ms. Alternatif LATERAL diuji terlebih dahulu tanpa mengubah kode: ia memakai
`idx_articles_author_created` untuk dua scan per-author, lalu sort 42 baris.
Waktunya 6.263 ms dan 0.126 ms tanpa query; dengan `kebijakan`, 18.595 ms dan
1.485 ms (960 baris dibuang oleh filter). Karena waktu hangat kedua alternatif
di bawah 50 ms dan lebih cepat daripada bentuk lama, bentuk LATERAL diterapkan
hanya untuk lebih dari satu author id.

SQL baru membatasi setiap author sampai `limit+1` di subquery LATERAL, lalu
menggabungkan dan membatasi lagi dengan urutan `created_at DESC, id DESC`.
Query, cursor, dan semua id tetap menjadi parameter. Pengukuran berikut
memakai prepared statement dengan `force_custom_plan`, `LIMIT 21`, dan dua
eksekusi pada SQL yang dirakit repository.

| Kasus | Node utama dan index | Baris | Eksekusi 1 (ms) | Eksekusi 2 (ms) |
|---|---|---:|---:|---:|
| `Syndication`, tanpa query | Sort → per-author Limit → Index Scan; `idx_articles_author_created` | 21 | 0.364 | 0.187 |
| `Syndication`, `kebijakan` | Sort → per-author Limit → Index Scan; `idx_articles_author_created` | 21 | 1.854 | 1.739 |
| `Syndication`, `hidrokarbon` | Sort → per-author Limit → Index Scan; `idx_articles_author_created` | 4 | 3156.963 | 233.154 |
| `Syndication`, `zzxqv` | Sort → per-author Limit → Index Scan; `idx_articles_author_created` | 0 | 203.670 | 197.478 |
| `Budi Santoso`, tanpa query | Sort → per-author Limit → Index Scan; `idx_articles_author_created` | 21 | 8.564 | 0.143 |

Kasus `hidrokarbon` dan `zzxqv` masih lambat: masing-masing scan per-author
membuang 150236 dan 150238 baris untuk mencari hasil langka atau tidak ada.
Kasus Budi hangat 0.143 ms, tidak lebih buruk dari angka sebelum perbaikan
0.239 ms. Pengukuran API lokal untuk `GET /articles?author=Syndication&limit=20`
memberi HTTP 200 dalam 15.399 ms lalu 1.113 ms.

### Perbandingan kata jarang: bentuk lama vs LATERAL

Pada 8 November 2026, bentuk lama `author_id = ANY(ARRAY['author-000001',
'author-giant'])` diukur dua kali dengan `LIMIT 21`. Untuk `hidrokarbon`, ia
memakai `Sort → Bitmap Heap Scan` dengan indeks `idx_articles_search`, memberi
4 baris dalam 7.208 ms lalu 1.071 ms (6 baris dibuang oleh filter). Untuk
`zzxqv`, plan yang sama memberi 0 baris dalam 1.405 ms lalu 0.949 ms; tidak
ada baris yang dibuang karena bitmap index scan memberi 0 baris. Bentuk
LATERAL repository untuk `zzxqv` memindai `idx_articles_author_created`; saat
diisolasi ke `author-giant`, node `Index Scan` melaporkan `Rows Removed by
Filter: 300441`, 0 hasil, dan 251.753 ms. Pada LATERAL dua author sekaligus,
node tersebut ber-loop dua kali dan EXPLAIN melaporkan rata-rata 150238 baris
terbuang per loop. Ini adalah hasil pengukuran, bukan perubahan kode.

## Generic plan lama (opsional; distribusi data sebelumnya)

Untuk statement kasus 3 yang diprepare dengan parameter `$1` untuk author,
node utama eksekusi ke-1 adalah `Index Scan using idx_articles_created`.
Pada eksekusi ke-7 node utama berubah menjadi `Sort` di atas `Bitmap Heap Scan`
dengan `BitmapAnd`; karena itu node utama berubah.

## Temuan arsip (distribusi data sebelumnya)

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

## Raw output arsip (distribusi data sebelumnya)

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

`internal/postgres/pool.go` mengatur `MaxConns` dari `DB_MAX_CONNS`,
`statement_timeout=3000`, dan `plan_cache_mode=force_custom_plan`.
`docker compose config` saat pengukuran memberi `DB_MAX_CONNS=20` dan
`REQUEST_TIMEOUT=5s`. Runner `cmd/loadtest` hanya mengirim GET dan memakai
library standar Go; pemeriksaan `which` tidak menemukan `hey`, `k6`, atau
`vegeta`. Pengujian dilakukan pada 8 November 2026 terhadap 1 juta baris di
satu mesin lokal; cache TIDAK dibangun (pengembangan lanjut).

| Skenario | Worker/durasi | Total | RPS | 200 / 400 / 503 / 500 / error koneksi | p50 | p95 | p99 | Koneksi maksimum |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Burst campuran: 60% halaman pertama, 20% Syndication, 10% `kebijakan`, 10% Budi Santoso | 200 / 20 dtk | 164639 | 8231.95 | 164639 / 0 / 0 / 0 / 0 | 20.163142 ms | 39.699694 ms | 45.060846 ms | 21 |
| Terberat: Syndication + `zzxqv` | 50 / 10 dtk | 410 | 41.00 | 410 / 0 / 0 / 0 / 0 | 1.102444604 dtk | 1.627276657 dtk | 2.076260781 dtk | 21 |

Sampel `pg_stat_activity` tiap dua detik menunjukkan maksimum 21 koneksi pada
kedua skenario. Angka ini satu di atas `DB_MAX_CONNS=20` karena query sampler
`psql` sendiri juga terhubung ke database; pengukuran ini tidak memisahkan
koneksi sampler dari koneksi aplikasi. Tidak ada HTTP 400, 503, 500, atau error
koneksi pada dua run tersebut.

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

- `internal/httpapi` menangani HTTP dan meneruskan input ke `internal/article`.
- `internal/article` memuat service, tipe domain, validasi, dan aturan bisnis.
- `internal/postgres` mengimplementasikan repository PostgreSQL untuk service.
- Skema saat ini memisahkan `authors` dan `articles`, dengan FK `articles.author_id` ke `authors(id)`.
- Categories atau comments dapat ditambah lewat migrasi bernomor baru dan paket baru di `internal/`.
- Tabel fitur baru seharusnya memakai FK ke `articles(id)` tanpa mengubah kode artikel yang ada.
- Aturan bisnis fitur tersebut diletakkan pada service paket domainnya; handler dan repository menjadi batas HTTP dan penyimpanan.
