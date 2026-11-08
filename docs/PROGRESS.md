# Progress assessment

## Langkah 4b — penyelarasan bukti dengan bentuk data uji (8 November 2026)

| Item | Status | Bukti |
|---|---|---|
| Bentuk data `articles` | Terverifikasi | Count tetap 1.000.000 sebelum dan sesudah; seluruh 300441 `author-giant` berada pada bucket waktu 1--2 dan tidak ada pada bucket 9--10. Satu baris maksimum masuk bucket 11 menurut semantik `width_bucket`. |
| Perbandingan seed | Terverifikasi | `git diff b0763ae 923e1ba -M` hanya memindahkan seed. Seed 20.000 khusus `articles_test` menyebarkan giant pada bucket 1--10 (578--633 per bucket), tidak seperti `articles`. Sebab perubahan data utama TIDAK DIKETAHUI. |
| Matriks halaman pertama | Terverifikasi | Delapan `EXPLAIN (ANALYZE, BUFFERS)` literal memakai SQL cabang `ListArticles`, `LIMIT 21`, dua kali tiap kasus. Tidak ada `Seq Scan` atau waktu di atas 300 ms. |
| Prepared statement | Terverifikasi | Kasus giant + `kebijakan` default mengganti estimate ke generic pada eksekusi 6--7 tetapi node/index tetap `Index Scan idx_articles_author_created`; `force_custom_plan` mempertahankan estimate custom. |
| Dokumentasi | Diperbarui | `docs/evidence.md` menambah bentuk data, matriks aktual, dan label tegas untuk seluruh tabel/raw output distribusi sebelumnya. |

## Langkah 4 — pemeriksaan kecil, load test, dan extensibility (8 November 2026)

| Item | Status | Bukti |
|---|---|---|
| Distribusi `author-giant` | Terverifikasi | Dari 1.000 artikel terbaru ada 0 `author-giant`; pada offset 400.000--400.999 juga 0. |
| Kata jarang: bentuk lama vs LATERAL | Terverifikasi | Bentuk lama `ANY(...)` untuk `hidrokarbon` selesai 7.208/1.071 ms dan `zzxqv` 1.405/0.949 ms; LATERAL terisolasi `author-giant` untuk `zzxqv` membuang 300441 baris dan selesai 251.753 ms. |
| Load test GET-only | Terverifikasi | `cmd/loadtest` (standard library) menjalankan burst 200 worker/20 dtk: 164639 request, 8231.95 RPS, seluruhnya 200; heavy 50 worker/10 dtk: 410 request, 41.00 RPS, seluruhnya 200. |
| Pool dan koneksi | Terverifikasi | Compose memakai `DB_MAX_CONNS=20` dan `REQUEST_TIMEOUT=5s`; puncak `pg_stat_activity` 21 pada kedua run, termasuk koneksi sampler `psql`. |
| Data dan lifecycle app | Terverifikasi | `articles` berjumlah 1000000 sebelum dan sesudah; app dihentikan dengan `docker compose stop app`, tanpa menghapus volume. |
| Extensibility | Terdokumentasi | `docs/evidence.md` menguraikan batas handler/service/repository serta jalur migrasi dan paket baru untuk fitur ber-FK ke `articles(id)`. |

## Langkah 3b — multi-author Syndication (8 November 2026)

| Item | Status | Bukti |
|---|---|---|
| Diagnosa Syndication | Terverifikasi | `syndication` mengembalikan `author-000001` (35 artikel) dan `author-giant` (300441 artikel); giant memang ada dalam daftar. |
| Keputusan LATERAL | Diterapkan | Alternatif hangat 0.126 ms tanpa query dan 1.485 ms dengan `kebijakan`, keduanya di bawah 50 ms dan lebih cepat dari `ANY`. |
| Perbaikan repository | Terverifikasi | Lebih dari satu author memakai `unnest($1::text[])` dan LATERAL per-author, dengan query, cursor, urutan, dan `limit+1` di dalam serta di luar. Jalur satu author tidak berubah. |
| Urutan dan pagination | Terverifikasi | `TestIntegrationI8MultiAuthorOrdering` membandingkan seluruh halaman limit 5 dari tiga author bernama sama, termasuk query, dengan `ANY(...) ORDER BY created_at DESC, id DESC`. |
| Test | Terverifikasi | `make test` PASS untuk I1–I8; `go test ./... -count=1 -race` PASS terhadap `articles_test`. |
| Sisa kasus lambat | Terverifikasi | `Syndication` + `hidrokarbon` 3156.963/233.154 ms dan `zzxqv` 203.670/197.478 ms tetap lambat; fakta dan plan dicatat di `docs/evidence.md`. |

`articles` berjumlah 1000000 sebelum diagnosa dan 1000000 sesudah test. Tidak
ada perintah tulis terhadap database tersebut pada langkah ini; database yang
ditulis oleh test hanya `articles_test`.

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
