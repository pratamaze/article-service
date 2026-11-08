## Article service

Layanan artikel berbasis Go dan PostgreSQL.

## Struktur folder

```text
cmd/
  api/       aplikasi HTTP
  seed/      pembuat data simulasi
internal/
  article/   domain artikel
  httpapi/   HTTP handler
  postgres/  akses PostgreSQL
  author/    disiapkan untuk keputusan berikutnya
  cache/     disiapkan untuk keputusan berikutnya
  config/    disiapkan untuk keputusan berikutnya
migrations/  migrasi database
docs/        keputusan, evidence, dan penggunaan AI
scripts/     SQL data contoh dan sanity check
```

## Menjalankan layanan

```sh
docker compose up
```

## Menjalankan test

```sh
make test
```

## Membuat data simulasi

```sh
go run ./cmd/seed -n 1000000 -reset
```

## Dokumentasi

- [Keputusan](docs/decisions.md)
- [Evidence](docs/evidence.md)
- [Penggunaan AI](docs/ai-usage.md)
