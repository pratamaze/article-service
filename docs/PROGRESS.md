# Progress assessment

## Audit pasca rombak — 8 November 2026

| Item | Status | Bukti |
|---|---|---|
| Skema dan migrasi | Terverifikasi | `make test` membuat `articles_test` dan menjalankan `migrations/0001_create_tables.up.sql` serta `0002_create_indexes.up.sql` tanpa gagal. |
| Data uji | Terverifikasi | `SELECT count(*) FROM articles` pada database `articles` adalah `1000000` sebelum dan sesudah `make test`. |
| Repository | Terverifikasi | `internal/postgres/repository.go` memakai SQL berparameter; build, vet, dan test integrasi repository lulus. |
| API | Terverifikasi | `internal/httpapi/handler.go` memiliki rute `POST` dan `GET /articles`; test unit dan integrasi lulus. |
| Pool dan timeout | Terverifikasi | `internal/postgres/pool.go` mengatur `MaxConns`, `statement_timeout=3000`, dan `plan_cache_mode=force_custom_plan`; `TestIntegrationI6PoolRuntimeSettings` PASS. |
| Test | Terverifikasi | `go build ./...`, `go vet ./...`, dan `make test` lulus; semua test yang dijalankan PASS. |
| Docker Compose app | Terverifikasi | `Dockerfile` ada; `docker-compose.yml` memiliki `app`; `docker compose --env-file /dev/null config` valid. |
| Clone bersih | TIDAK DIKETAHUI | Tidak diuji dalam audit ini. |
| Evidence | Terverifikasi | `docs/evidence.md` ada, berisi EXPLAIN list dan raw output. |
| Load test | Tidak ada bukti | Pencarian `k6`, `vegeta`, `hey`, `load test`, dan `stress` di repo tidak menghasilkan skrip atau hasil. |
| README | Terverifikasi | `README.md` ada dan memuat struktur folder, cara menjalankan layanan/test, data simulasi, dan tautan dokumentasi. |
| Disclosure AI | Terverifikasi | `docs/ai-usage.md` ada, tetapi tabel bantuan AI dan bagian kegagalan AI masih berupa placeholder. |

## Rujukan path usang

- `docs/PROGRESS.md` lama merujuk `scripts/seed`, `docs/explain.md`,
  `docs/explain-raw.md`, dan `seed`; semuanya tidak ada lagi. Rujukan tersebut
  telah digantikan oleh catatan audit ini.
- Tidak ditemukan rujukan path yang tidak ada pada `Makefile`, `Dockerfile`,
  `docker-compose.yml`, atau `AGENTS.md` yang diperiksa.

## Kekurangan terhadap considerations

| Pemeriksaan | Ya/Tidak | Lokasi atau alasan |
|---|---|---|
| `docs/evidence.md` ada | Ya | `docs/evidence.md`, `# Bukti EXPLAIN ListArticles`; mencakup SQL, matriks enam kasus, generic-plan check, dan raw output. |
| EXPLAIN list tanpa filter, author giant, author kecil | Ya | `## Pengukuran awal (dengan cursor)` dan `## Raw output`: tanpa filter, `author-giant`, `author-000100`. |
| EXPLAIN nama kembar | Tidak | Tidak ada EXPLAIN untuk `Syndication` atau `Budi Santoso` yang memetakan ke beberapa ID. |
| EXPLAIN resolusi nama ke `authors.name_normalized` | Tidak | Tidak ada EXPLAIN query `authors`/`name_normalized`. |
| EXPLAIN cursor di tengah data | Ya | `docs/evidence.md`, `## Pengukuran awal (dengan cursor)` memakai predikat keyset `(created_at, id) < (...)`. |
| Test dokumen Indonesia, Inggris, dan campuran | Tidak | `*_test.go` hanya memeriksa pencarian `banjir`/`macet`; tidak ada tiga kategori bahasa tersebut. |
| Skrip atau hasil load test | Tidak | Pencarian repo untuk `k6`, `vegeta`, `hey`, `load test`, dan `stress` tidak menemukan bukti. |
| Cache di kode | Tidak | Pencarian `cache`/`singleflight` pada `*.go` tidak menemukan implementasi; hanya folder `internal/cache` kosong. |
| Penjelasan extensibility categories/comments | Tidak | `docs/decisions.md` kosong dan README tidak menjelaskan penambahan categories atau comments. |

## Hasil perintah audit

- `git status --short` tidak menghasilkan perubahan sebelum pembaruan file ini.
- `make test` hanya menjalankan `DROP DATABASE IF EXISTS articles_test WITH (FORCE)`,
  `CREATE DATABASE articles_test`, migrasi up ke `articles_test`, lalu test.
- Test PASS: `TestCursorRoundTripAndInvalidValues`, `TestCreateValidation`,
  `TestListLimitValidation`, dan integrasi `I1` sampai `I6`. Package `cmd/api`
  dan `cmd/seed` tidak memiliki test.
