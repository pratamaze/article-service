# Progress assessment

| Item | Status | Bukti |
|---|---|---|
| Skema dan migrasi | Terverifikasi | `migrations/0001_create_tables.up.sql` dan `0002_create_indexes.up.sql` dibaca. |
| Data uji | Terverifikasi | Query baca pada 8 November 2026 menghasilkan 23.334 author dan 1.000.000 artikel. |
| Repository SQL | Selesai | `internal/article/article.go` dan `internal/postgres/repository.go` dibuat; `GOCACHE=/tmp/article-service-go-build go test ./...` lulus. |
| Bukti EXPLAIN (6 kasus) | Selesai | `docs/evidence.md`; 6 kasus halaman pertama masing-masing diukur dua kali dengan `EXPLAIN (ANALYZE, BUFFERS)` melalui `make psql`. |
| API POST | Selesai | `POST /articles`: batas body 1 MB, validasi rune, ULID server, dan pemetaan 400/422. Diverifikasi dengan curl pada 8 November 2026. |
| API GET | Selesai | `GET /articles`: keyset cursor base64 URL-safe mikrodetik, limit 1--100, author/name search, dan error 400. Diverifikasi dengan curl pada 8 November 2026. |
| Pengaman dasar | Selesai | Timeout per request, `ReadHeaderTimeout`, `statement_timeout=3000`, logging error internal, dan custom plan. |
| Test | Selesai | Unit `TestCursorRoundTripAndInvalidValues`, `TestCreateValidation`, dan `TestListLimitValidation`; integrasi `TestIntegrationI1KeysetNoDuplicateOrMissing` sampai `I6PoolRuntimeSettings`. `make test` dan `go test ./... -count=1 -race` lulus pada 8 November 2026. |
| Docker Compose untuk app | Selesai | `docker compose --env-file /dev/null config` valid pada 8 November 2026; app+db jalan pada clone bersih tanpa `.env`. |
| README | TIDAK DIKETAHUI | Belum diverifikasi pada Langkah 1. |
| Disclosure AI | Terverifikasi | `docs/ai-usage.md` ada; bagian "Satu kegagalan AI" masih berupa teks petunjuk kosong. |

## Temuan

- `docker compose ps` pada 8 November 2026 menunjukkan service `db` sehat dan
  memetakan port host 5434 ke PostgreSQL 5432.
- Index artikel yang terdaftar: `articles_pkey`, `idx_articles_created`,
  `idx_articles_author_created`, dan `idx_articles_search`.
- Sebelum langkah ini tidak ada file explain di `docs/`.
- Enam plan halaman pertama tidak memakai `Seq Scan`; tidak ada `Sort` dengan
  lebih dari 10.000 baris dan tidak ada `Execution Time` di atas 300 ms.
- Generic-plan check kasus `author-giant` + `kebijakan` berubah dari `Index Scan`
  pada eksekusi pertama menjadi `Sort` di atas `Bitmap Heap Scan` pada eksekusi
  ketujuh.

## Langkah 2 (API HTTP dan pool)

- `go build ./...`, `go vet ./...`, dan `go test ./...` selesai pada 8 November
  2026. Perintah test mengeluarkan peringatan cache modul read-only, tetapi
  seluruh package melaporkan hasil lulus atau tidak memiliki test.
- Server sementara pada port 18080 diverifikasi dengan curl. POST valid memberi
  201; author tidak ada 422; title kosong dan author_id tidak ada memberi 400.
  Artikel uji `01M6VPYJA8YRTFMYCAHNYDCXB8` dihapus dan psql melaporkan `DELETE 1`.
- GET limit/cursor menghasilkan dua halaman tanpa ID berulang; `Syndication`
  memberi 200; author tidak ada memberi `items: []`; `Admin`, cursor rusak,
  limit 1000, dan author+author_id memberi 400. Query `berita` dan query kosong
  yang dipangkas memberi 200.

## Langkah 3 (test otomatis)

- `make test` membuat ulang database terpisah `articles_test`, menerapkan dua
  migrasi up, lalu menjalankan `go test ./... -v -count=1` dengan
  `TEST_DATABASE_URL`. I1--I6 lulus tanpa skip pada 8 November 2026.
- Test unit mencakup cursor mikrodetik dan nilai rusak, batas validasi POST
  berbasis rune, serta default/batas limit. Test HTTP-integrasi mencakup keyset
  55 artikel (50 timestamp sama), nama author/ambigu, normalisasi, edit dan
  search vector, input rusak, serta runtime parameter pool.
- Refactor minimum: `postgres.NewPool` dipindahkan ke `internal/postgres/pool.go`
  dan dipakai oleh `main.go`, sehingga I6 memeriksa factory yang sama dengan API.
- `go test ./... -count=1 -race` lulus. Sanity check I1 dengan `id DESC`
  sementara dihapus gagal (`got 95 ids, want 226`); `repository.go` dipulihkan
  dan `git diff -- internal/postgres/repository.go` kosong sebelum test akhir.
- Count `articles` database produksi dibaca sebelum dan sesudah test: keduanya
  `1000000`; test hanya menulis ke `articles_test`.

## Langkah 4 (runnable dari clone bersih)

- Ditambahkan `Dockerfile` multi-stage untuk `cmd/api` dengan basis builder
  `golang:1.26-alpine` sesuai `go.mod`, image akhir
  `gcr.io/distroless/static-debian12:nonroot`, dan proses jalan sebagai non-root.
- Ditambahkan `.dockerignore` berisi `.env`, `seed.log`, `.git`, `docs`, dan `bin`.
- `docker-compose.yml` diperbarui: service `app` ditambahkan, `depends_on` ke
  `db` memakai `condition: service_healthy`, port app host default `8080`,
  `DATABASE_URL`/`DB_MAX_CONNS`/`REQUEST_TIMEOUT` diberi default, dan
  `container_name` pada `db` dihapus.
- Semua variabel `db` memakai default `${VAR:-nilai}` agar compose valid tanpa
  `.env`; verifikasi `env -u ... docker compose --env-file /dev/null config`
  berhasil pada 8 November 2026.
- Inisialisasi database baru otomatis dipasang hanya dari file up migration:
  `01_create_tables.up.sql`, `02_create_indexes.up.sql`, dan
  `03_sample_data.sql` (read-only).
- Ditambahkan `scripts/sample_data.sql` berisi 3 author (`201`/`202` sama-sama
  bernama Budi, `203` bernama Sari) dan 4 artikel contoh.
- `Makefile` diperbarui: `-include .env`, default `?=` untuk
  `POSTGRES_USER/POSTGRES_PASSWORD/POSTGRES_DB/POSTGRES_PORT` (serta default
  runtime app), dan target baru `app` (`docker compose up -d --build`).
- Uji clone bersih di `/tmp/article-clone` (tanpa `.env`):
  `POSTGRES_PORT=5440 APP_PORT=18081 docker compose -p articleclone up -d --build`
  berhasil; `docker compose -p articleclone ps` menunjukkan `db` healthy.
- Verifikasi endpoint clone bersih via curl:
  `GET /articles?limit=2` -> 200 (berisi data contoh),
  `GET /articles?author=budi` -> 200 (memuat artikel author `201` dan `202`),
  `POST /articles` dengan `author_id=201` -> 201,
  `POST /articles` dengan `author_id` tidak ada -> 422.
- Stack uji ditutup dengan `docker compose -p articleclone down` (tanpa `-v`).
  Repo asli tetap jalan (`docker compose ps` menampilkan `article-db` healthy),
  dan query `SELECT count(*) FROM articles;` di database asli tetap `1000000`.

## Langkah berikutnya

TIDAK DIKETAHUI.

## Langkah 5 (struktur folder)

- Program seed dipindahkan dari `scripts/seed` ke `cmd/seed`; referensi Makefile
  diperbarui.
- Bukti EXPLAIN digabungkan ke `docs/evidence.md`; file sumber lama dihapus
  setelah isinya dipertahankan.
- Tidak ada perintah database yang dijalankan pada langkah ini.
