# Progress assessment

| Item | Status | Bukti |
|---|---|---|
| Skema dan migrasi | Terverifikasi | `migrations/0001_create_tables.up.sql` dan `0002_create_indexes.up.sql` dibaca. |
| Data uji | Terverifikasi | Query baca pada 8 November 2026 menghasilkan 23.334 author dan 1.000.000 artikel. |
| Repository SQL | Selesai | `internal/article/article.go` dan `internal/postgres/repository.go` dibuat; `GOCACHE=/tmp/article-service-go-build go test ./...` lulus. |
| Bukti EXPLAIN (6 kasus) | Selesai | `docs/explain.md` dan `docs/explain-raw.md`; 6 kasus halaman pertama masing-masing diukur dua kali dengan `EXPLAIN (ANALYZE, BUFFERS)` melalui `make psql`. |
| API POST | Belum dikerjakan | Di luar Langkah 1. |
| API GET | Belum dikerjakan | Di luar Langkah 1. |
| Pengaman dasar | Belum dikerjakan | Langkah berikutnya. |
| Test | sebagian (smoke test repository) | `internal/postgres/repository_test.go` menguji pagination, `syndication`, dan author ambigu; `go test ./... -v` lulus dengan `DATABASE_URL` dari `.env`, dan test dilewati bila variabel kosong. |
| Docker Compose untuk app | TIDAK DIKETAHUI | Belum diverifikasi pada Langkah 1. |
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

## Langkah berikutnya

Langkah 2, pengaman dasar.
