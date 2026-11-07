# Konteks assessment

Layanan ini menyimpan dan menampilkan artikel dengan Go, pgx/v5 + pgxpool, dan
SQL tanpa ORM. Urutan daftar adalah `created_at DESC, id DESC`; pagination
memakai keyset dan tanpa total count. Search memakai
`websearch_to_tsquery('simple', $q)` pada `articles.search_vector`. Nama author
dinormalisasi dengan `lower(btrim(regexp_replace(...)))`; lebih dari 20 author
cocok berarti `ErrAmbiguousAuthor`.

Jangan gunakan query catch-all, interpolasi input ke SQL, atau mengubah migrasi
dan seed. Jangan menjalankan TRUNCATE, DROP, seed, migrate-down, atau compose
down dengan volume. Awal sesi baca file ini dan `docs/PROGRESS.md`; kerjakan
satu langkah saja. Akhir sesi perbarui `docs/PROGRESS.md`, commit, lalu jelaskan
dalam lima baris bahasa sederhana apa yang dikerjakan dan alasannya.

## Keputusan pool

API membaca `DATABASE_URL` (wajib), `DB_MAX_CONNS` (default 20), dan
`REQUEST_TIMEOUT` (default 5s). Setiap koneksi memakai `statement_timeout=3000`
dan `plan_cache_mode=force_custom_plan`: pengukuran prepared statement pada 8
November 2026 berubah ke generic plan pada eksekusi ke-6 dan ke-7, sedangkan
custom plan tetap memakai plan parameter aktual.
