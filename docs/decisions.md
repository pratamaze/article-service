# Keputusan desain

Stack terdiri dari Go (`net/http`), PostgreSQL 16, pgx, REST, dan JSON, tanpa ORM
atau search engine terpisah. Angka diukur pada 1.000.000 artikel di satu mesin lokal;
"hangat" berarti eksekusi kedua. Bukti lengkap ada di [evidence.md](evidence.md).

## 1. Volume

Keputusan: urutan memakai `created_at DESC, id DESC` dengan index dan keyset
pagination. Cursor berisi `created_at` presisi mikrodetik serta id. Tidak ada total
count, respons memakai `next_cursor` dan `has_more`, dan list mengembalikan excerpt
200 karakter, bukan body. Limit default 20 dan maksimum 100.

Alasan: `OFFSET` tetap membaca lalu membuang baris. Pasangan `(created_at, id)`
mencegah artikel dengan waktu sama hilang atau ganda pada batas halaman. Test I1
menggunakan 55 artikel, 50 di antaranya memiliki `created_at` sama, tanpa artikel
hilang atau ganda. Jika id dihapus dari `ORDER BY`, test gagal dengan 95 id ditemukan
dari 226. Kelemahannya, layanan tidak dapat lompat ke halaman tertentu dan tidak
memberikan jumlah total.

## 2. Author timpang

Keputusan: gunakan index `(author_id, created_at DESC, id DESC)`. Satu id memakai
`author_id = $n`; beberapa id memakai satu subquery LATERAL per id, lalu hasil digabung
dan diambil 21 teratas.

Pada data saat ini, `author-giant` memiliki 300441 artikel dan tanpa query selesai
dalam 0.055 ms. Pada pengukuran arsip, `Syndication` dengan bentuk lama
`author_id = ANY` selesai 250.481 ms, sedangkan bentuk LATERAL selesai 0.187 ms.
Planner bentuk lama memperkirakan 323783 baris, memilih `idx_articles_created`, dan
membuang 434464 baris sebelum menemukan 21 hasil. Kata sangat jarang atau tidak ada
tetap menjadi kelemahan untuk author raksasa dengan nama kembar: `hidrokarbon` 233.154
ms dan `zzxqv` 197.478 ms, dibanding bentuk lama 1.071 ms dan 0.949 ms. Biaya pada
volume jauh lebih besar belum diukur.

Pengukuran arsip tersebut diberi label distribusi data sebelumnya di evidence. Pada
data yang tersebar merata, bentuk lama belum diukur.

## 3. Nama author tidak unik

Keputusan: nama dicocokkan persis tanpa membedakan huruf besar-kecil, dengan spasi
dirapikan melalui `name_normalized`. Hasilnya gabungan semua author bernama tersebut;
setiap item memuat `author_id`. Lebih dari 20 kecocokan menghasilkan 400, nama tidak
ada menghasilkan 200 dengan `items` kosong, dan penggunaan `author` bersama
`author_id` menghasilkan 400.

Test I2, I3, dan I8 memeriksa perilaku tersebut, termasuk urutan dan pagination tiga
author bernama sama. Resolusi nama berada pada rentang 0.015 sampai 0.028 ms.
Kelemahannya, nama sangat umum ditolak, pencocokan sebagian tidak didukung, dan
`author_id` adalah filter yang stabil bila nama berubah.

## 4. Bahasa campur

Keputusan: full-text memakai konfigurasi `simple` secara eksplisit tanpa deteksi
bahasa, melalui `websearch_to_tsquery('simple', ...)`. Test I7 mencakup artikel
Indonesia, Inggris, dan campuran. Kelemahannya adalah tidak ada stemming: `meluncur`
tidak menemukan `meluncurkan`, `launch` tidak menemukan `launches`, dan `flooded`
tidak menemukan `flooding`.

## 5. Artikel diedit

Keputusan: `search_vector` adalah generated column sehingga ikut berubah saat UPDATE.
UPDATE hanya menyentuh title, body, dan `updated_at`; urutan tidak memakai
`updated_at`, sehingga posisi artikel stabil. Test I4 membuktikan kata lama hilang,
kata baru muncul, serta `created_at` dan posisi tetap. Endpoint PATCH tidak dibuat;
edit dibuktikan melalui test repository. Tidak ada penanganan dua edit bersamaan.

## 6. Baca tinggi dan bursty

Keputusan: pool maksimum 20 koneksi, timeout per request, `statement_timeout` 3
detik, limit maksimum 100, dan `plan_cache_mode = force_custom_plan`.

Burst 200 worker selama 20 detik menghasilkan 164639 permintaan, 8231.95 RPS, seluruh
respons 200, dan p95 39.699694 ms. Kasus `Syndication` dengan `zzxqv`, 50 worker
selama 10 detik, menghasilkan 410 respons 200, p50 1.102444604 detik, dan p95
1.627276657 detik. Koneksi maksimum 21, termasuk satu koneksi sampler `psql`, untuk
pool 20. Pada pengukuran arsip, generic plan 77.426 ms dibanding custom plan 2.223 ms
di eksekusi ke-7. Pada data saat ini, node sama dan waktunya 2.490 ms dibanding 0.509
ms. Load generator berjalan pada mesin yang sama dengan database; `statement_timeout`
dan timeout request tidak terpicu dalam pengujian.

## 7. Sengaja tidak dibuat

Cache dan singleflight, PATCH, rate limiting, search engine terpisah, partisi, dan
read replica tidak dibuat. Semua itu merupakan jalur skala lanjut.

## 8. Keterbatasan data uji

Data sintetis berisi 1 juta baris, body pendek, dan kata uji ditanam tanpa hubungan
dengan author. Pada data utama, `created_at` seluruh 300441 artikel
`author-giant` digeser mundur empat tahun secara manual setelah seed dijalankan,
sehingga giant lebih tua dari artikel lain. Perintah persisnya tidak tercatat;
[`scripts/shift_giant.sql`](../scripts/shift_giant.sql) adalah rekonstruksi setara.
Seed di repo menyebarkan giant merata dalam waktu sehingga hasil dari `make seed` dapat
berbeda dari evidence. Alasan atau niat pergeseran manual itu TIDAK DIKETAHUI.
Pengukuran sebelum pergeseran diberi label tersendiri di evidence.
