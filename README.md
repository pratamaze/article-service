# Article service

Layanan REST untuk membuat dan menampilkan artikel dengan PostgreSQL, pagination
keyset, pencarian full-text, serta filter author.

## Menjalankan

Pada clone bersih, jalankan `docker compose up -d --build`. Port bawaan database
adalah 5433 dan API adalah 8080.

```sh
curl 'http://localhost:8080/articles?limit=2'
curl 'http://localhost:8080/articles?author=budi'
curl -X POST http://localhost:8080/articles \
  -H 'Content-Type: application/json' \
  -d '{"author_id":"201","title":"Artikel baru","body":"Isi artikel"}'
```

Data contoh menyediakan author `201`, `202`, dan `203`.

## Menjalankan test

Database harus berjalan. Jalankan `make test`; target ini membuat database terpisah
`articles_test`, menerapkan migrasi, lalu menjalankan test Go.

## Data uji besar

Gunakan target yang tersedia, misalnya `make seed N=1000000`. Seed menulis ke database
utama dan menyebarkan `author-giant` merata dalam waktu; angka dapat berbeda dari
[keterbatasan data uji](docs/decisions.md#8-keterbatasan-data-uji).

## API ringkas

| Endpoint | Parameter atau body | Respons utama |
|---|---|---|
| `GET /articles` | `query`, `author`, `author_id`, `limit`, `cursor` | 200, `{items,next_cursor,has_more}` |
| `POST /articles` | JSON: `author_id`, `title`, `body` | 201 artikel baru |

`limit` default 20 dan berada pada rentang 1 sampai 100. `author` dan `author_id`
tidak dapat dipakai bersama. Kode status utama adalah 400 untuk input tidak valid, 422
untuk author id tidak ada, 503 untuk batas waktu, dan 500 untuk kesalahan internal.

## Struktur folder

`cmd/api` aplikasi HTTP.
`cmd/loadtest` program load test.
`cmd/seed` pembuat data sintetis.
`internal/article` domain dan aturan bisnis.
`internal/httpapi` handler HTTP.
`internal/postgres` akses PostgreSQL.
`migrations` definisi skema dan index.
`scripts` data contoh dan pemeriksaan skema.
`docs` keputusan, bukti, dan penggunaan AI.

## Dokumentasi

- [Keputusan desain](docs/decisions.md)
- [Bukti pengukuran](docs/evidence.md)
- [Penggunaan AI](docs/ai-usage.md)
