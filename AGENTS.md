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
