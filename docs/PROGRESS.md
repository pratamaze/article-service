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
| Clone bersih | Terverifikasi | Clone sementara tanpa `.env` berhasil dibangun dan dijalankan terisolasi sebagai project `articleclone` pada port 5440/18081. GET list dan filter `author=budi` memberi 200; POST author `201` memberi 201 dan author tidak ada memberi 422; `go build ./...` lulus. |
| Evidence | Terverifikasi | `docs/evidence.md` berisi EXPLAIN list, resolusi nama kembar, pemeriksaan prepared statement, dan hasil I7. |
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
| EXPLAIN nama kembar | Ya | `docs/evidence.md`, `## Nama kembar dan resolusi nama`; mencakup `Syndication` dan `Budi Santoso`. |
| EXPLAIN resolusi nama ke `authors.name_normalized` | Ya | `docs/evidence.md`, `## Nama kembar dan resolusi nama`; memakai `idx_authors_name_norm`. |
| EXPLAIN cursor di tengah data | Ya | `docs/evidence.md`, `## Pengukuran awal (dengan cursor)` memakai predikat keyset `(created_at, id) < (...)`. |
| Test dokumen Indonesia, Inggris, dan campuran | Ya | `internal/postgres/integration_test.go`, `TestIntegrationI7MixedLanguageSearch`; seluruh query di-scope ke author khusus. |
| Skrip atau hasil load test | Tidak | Pencarian repo untuk `k6`, `vegeta`, `hey`, `load test`, dan `stress` tidak menemukan bukti. |
| Cache di kode | Tidak | Pencarian `cache`/`singleflight` pada `*.go` tidak menemukan implementasi; hanya folder `internal/cache` kosong. |
| Penjelasan extensibility categories/comments | Tidak | `docs/decisions.md` kosong dan README tidak menjelaskan penambahan categories atau comments. |

## Hasil perintah audit

- `git status --short` tidak menghasilkan perubahan sebelum pembaruan file ini.
- Verifikasi clone bersih (8 November 2026): `POSTGRES_PORT=5440 APP_PORT=18081 docker compose -p articleclone up -d --build` berhasil; `db` berstatus healthy. `GET /articles?limit=2` memberi 200 dan data contoh, sedangkan `GET /articles?author=budi` memberi 200 dengan author_id `201` dan `202`. POST valid memakai author `201` memberi 201; POST dengan `author_id` tidak ada memberi 422. `go build ./...` di clone lulus. Project uji dihentikan memakai `docker compose -p articleclone down` tanpa `-v`, lalu clone sementara dihapus.
- Setelah uji clone, `docker compose ps` di repo asli menunjukkan `article-db` tetap healthy, dan `SELECT count(*) FROM articles` menghasilkan `1000000`.
- `make test` hanya menjalankan `DROP DATABASE IF EXISTS articles_test WITH (FORCE)`,
  `CREATE DATABASE articles_test`, migrasi up ke `articles_test`, lalu test.
- Test PASS langkah 3 (8 November 2026): `make test` membuat ulang
  `articles_test`, lalu `TestIntegrationI1KeysetNoDuplicateOrMissing` sampai
  `TestIntegrationI7MixedLanguageSearch` semuanya PASS tanpa SKIP. I7
  memverifikasi dokumen Indonesia, Inggris, dan campuran termasuk batas tanpa
  stemming serta query tiga spasi. Package `cmd/api`
  dan `cmd/seed` tidak memiliki test.
- EXPLAIN langkah 3 pada database `articles` hanya memakai `SELECT`, `EXPLAIN`,
  `PREPARE`, `DEALLOCATE`, serta `SET`/`RESET plan_cache_mode`; jumlah artikel
  sebelum pengukuran adalah `1000000`. Dua kasus `Syndication` melebihi 300 ms
  dan telah dicatat apa adanya di `docs/evidence.md`.
