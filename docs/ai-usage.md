# Penggunaan AI

## Bagian yang dibantu AI

- Eksplorasi ide dan analisis constraint soal, serta usulan desain (skema, index,
  pagination, pencarian): asisten AI percakapan (Claude).
- Penulisan kode (repository, API, test, Dockerfile, Compose, program load test, skrip
  seed): AI coding agent, per langkah, dengan prompt yang membatasi tugasnya.
- Penulisan dokumentasi (decisions, evidence, README): dibantu AI, lalu dicocokkan
  dengan hasil pengukuran.

## Yang saya periksa sendiri

- Menjalankan make test, EXPLAIN ANALYZE, load test, dan uji clone bersih.
- Memeriksa laporan agent dengan git log, curl, dan query langsung ke database.
- Memeriksa bentuk data uji dan mencatat bahwa created_at giant digeser setelah seed.
- Memutuskan untuk tidak membangun cache, PATCH, dan rate limiting.

## Satu kegagalan AI

Agent melaporkan "go test lulus" untuk repository, padahal belum ada satu pun file
test; output sebenarnya hanya "[no test files]". Saya mengetahuinya saat memeriksa
laporan itu, lalu meminta test dibuat dan menjalankannya sendiri. Sejak itu setiap
prompt saya mewajibkan klaim berasal dari output perintah yang dijalankan, dan saya
membaca daftar test per nama, bukan hanya kata "lulus".
