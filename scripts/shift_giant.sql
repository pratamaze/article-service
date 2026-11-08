-- Rekonstruksi yang setara dengan perubahan manual pada data utama (dijalankan
-- setelah seed). Perintah asli tidak tercatat. Menghasilkan giant yang lebih tua
-- dari artikel lain, berbeda dari keluaran seed yang menyebarkan giant merata.
UPDATE articles SET created_at = created_at - interval '4 years'
WHERE author_id = 'author-giant';
