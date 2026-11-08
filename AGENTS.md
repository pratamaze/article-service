# Konteks assessment dan status akhir

Layanan ini menyimpan dan menampilkan artikel dengan Go, pgx/v5 + pgxpool, dan
SQL tanpa ORM. Urutan daftar adalah `created_at DESC, id DESC`; pagination
memakai keyset dan tanpa total count. Search memakai
`websearch_to_tsquery('simple', $q)` pada `articles.search_vector`. Nama author
dinormalisasi dengan kombinasi `lower`, `btrim`, dan `regexp_replace` di query repository; lebih dari 20 author
cocok berarti `ErrAmbiguousAuthor`.

Jangan gunakan query catch-all, interpolasi input ke SQL, atau mengubah migrasi
dan seed. Jangan menjalankan TRUNCATE, DROP, seed, migrate-down, atau compose
down dengan volume. Awal sesi baca file ini dan `docs/PROGRESS.md`; kerjakan
satu langkah saja. Akhir sesi perbarui `docs/PROGRESS.md`, commit, lalu jelaskan
dalam lima baris bahasa sederhana apa yang dikerjakan dan alasannya.

## Keputusan pool

API membaca `DATABASE_URL` (wajib), `DB_MAX_CONNS` (default 20), dan
`REQUEST_TIMEOUT` (default 5s). Setiap koneksi memakai `statement_timeout=3000`
dan `plan_cache_mode=force_custom_plan`: pada pengukuran data saat ini, prepared
statement default mengubah estimate pada eksekusi ke-6 dan ke-7, sedangkan
custom plan mempertahankan estimate parameter aktual. Lihat `docs/evidence.md`.

## Menjalankan test

Jalankan `make test` untuk membuat ulang database terpisah `articles_test`,
menerapkan migrasi, lalu menjalankan `go test ./... -v -count=1` dengan
`TEST_DATABASE_URL`. Target ini tidak menggunakan database `articles`.
Untuk race detector, sumber `.env` lalu jalankan `go test ./... -count=1 -race`
dengan `TEST_DATABASE_URL` yang menunjuk ke `articles_test`.

## Menjalankan service dari clone bersih

`docker-compose.yml` sudah punya nilai default untuk service `db` dan `app`,
jadi bisa jalan tanpa file `.env`.

1. Jalankan stack:
	`docker compose up -d --build`
2. Atur port terpisah bila perlu:
	`POSTGRES_PORT=5440 APP_PORT=18081 docker compose -p articleclone up -d --build`
3. Untuk menghentikan stack uji, gunakan tanpa hapus volume:
	`docker compose -p articleclone down`

## Dokumen pengiriman

Dokumen akhir ada pada `README.md`, `docs/decisions.md`, `docs/evidence.md`,
dan `docs/ai-usage.md`. Rekonstruksi setara pergeseran waktu giant berada pada
`scripts/shift_giant.sql`; perintah perubahan asli tidak tercatat.
