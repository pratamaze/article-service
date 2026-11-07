INSERT INTO authors (id, name) VALUES
  ('201', 'Budi'),
  ('202', 'Budi'),
  ('203', 'Sari');

INSERT INTO articles (id, author_id, title, body, created_at, updated_at) VALUES
  ('art-001', '201', 'Berita Kota', 'Contoh artikel pertama dari Budi.', now() - interval '5 minutes', now() - interval '5 minutes'),
  ('art-002', '202', 'Catatan Pagi', 'Contoh artikel kedua dari Budi lain.', now() - interval '4 minutes', now() - interval '4 minutes'),
  ('art-003', '203', 'Teknologi Hari Ini', 'Artikel contoh dari Sari untuk pencarian.', now() - interval '3 minutes', now() - interval '3 minutes'),
  ('art-004', '201', 'Kuliner Nusantara', 'Artikel tambahan untuk pagination dasar.', now() - interval '2 minutes', now() - interval '2 minutes');
