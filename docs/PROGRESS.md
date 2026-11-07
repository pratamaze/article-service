# Progress assessment

| Item | Status | Bukti |
|---|---|---|
| Skema dan migrasi | Terverifikasi | `migrations/0001_create_tables.up.sql` dan `0002_create_indexes.up.sql` dibaca. |
| Data uji | Terverifikasi | Query baca pada 8 November 2026 menghasilkan 23.334 author dan 1.000.000 artikel. |
| Repository SQL | Selesai | `internal/article/article.go` dan `internal/postgres/repository.go` dibuat; `GOCACHE=/tmp/article-service-go-build go test ./...` lulus. |
| Bukti EXPLAIN (6 kasus) | Selesai | `docs/explain.md` dan `docs/explain-raw.md`; 6 kasus halaman pertama masing-masing diukur dua kali dengan `EXPLAIN (ANALYZE, BUFFERS)` melalui `make psql`. |
| API POST | Selesai | `POST /articles`: batas body 1 MB, validasi rune, ULID server, dan pemetaan 400/422. Diverifikasi dengan curl pada 8 November 2026. |
| API GET | Selesai | `GET /articles`: keyset cursor base64 URL-safe mikrodetik, limit 1--100, author/name search, dan error 400. Diverifikasi dengan curl pada 8 November 2026. |
| Pengaman dasar | Selesai | Timeout per request, `ReadHeaderTimeout`, `statement_timeout=3000`, logging error internal, dan custom plan. |
| Test | Selesai | Unit `TestCursorRoundTripAndInvalidValues`, `TestCreateValidation`, dan `TestListLimitValidation`; integrasi `TestIntegrationI1KeysetNoDuplicateOrMissing` sampai `I6PoolRuntimeSettings`. `make test` dan `go test ./... -count=1 -race` lulus pada 8 November 2026. |
| Docker Compose untuk app | TIDAK DIKETAHUI | Tidak dikerjakan pada Langkah 2. |
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

## Langkah berikutnya

TIDAK DIKETAHUI.
