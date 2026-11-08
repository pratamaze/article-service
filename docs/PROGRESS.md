# Progress assessment

## Status akhir, 8 November 2026

| Langkah | Status | Bukti |
|---|---|---|
| API, repository, dan migrasi | Terverifikasi | `make test` memakai `articles_test`; test integrasi mencakup keyset, author, edit, input tidak valid, pengaturan pool, search bahasa campur, dan multi-author. |
| Multi-author | Diterapkan | Lebih dari satu author memakai LATERAL per author. Hasil sebelum pergeseran giant dan hasil sesudahnya diberi label terpisah di `docs/evidence.md`. |
| Load test | Terverifikasi | `cmd/loadtest` menjalankan burst 200 worker selama 20 detik dan skenario berat 50 worker selama 10 detik. Angka lengkap berada di evidence. |
| Bentuk data | Terverifikasi | `articles` tetap 1.000.000 baris. Seed 20.000 di `articles_test` menyebarkan giant di bucket 1 sampai 10, sedangkan data utama menempatkan 300441 giant pada bucket 1 sampai 2. |
| Prepared statement | Terverifikasi | Pada data saat ini, node dan index tetap sama di eksekusi 1 dan 7; default mengubah estimate, sedangkan `force_custom_plan` mempertahankan estimate custom. |
| Dokumen pengiriman | Selesai | README, keputusan desain, penggunaan AI, evidence, aturan kerja, dan `.gitignore` telah diperbarui. `scripts/shift_giant.sql` mencatat rekonstruksi setara perubahan data utama. |

## Hal yang tidak diketahui

- Perintah asli untuk pergeseran `created_at` giant tidak tercatat.
- Biaya LATERAL pada 50 juta artikel belum diukur.
- Bentuk lama `author_id = ANY` pada data giant yang tersebar merata belum diukur.
