BEGIN;

-- 1. Normalisasi nama
INSERT INTO authors (id, name) VALUES
  ('a1', 'Budi'), ('a2', '   budi   '), ('a3', 'ÉMILE'), ('a4', 'émile');
SELECT id, name, name_normalized FROM authors ORDER BY id;
-- Harapan: a1 dan a2 sama-sama 'budi'. a3 dan a4 sama-sama 'émile' (jika locale UTF-8).

-- 2. Insert artikel, search_vector terisi otomatis
INSERT INTO articles (id, author_id, title, body) VALUES
  ('x1', 'a1', 'Banjir Jakarta', 'Hujan deras mengguyur ibu kota'),
  ('x2', 'a1', 'Startup lokal', 'Pemerintah meluncurkan program untuk startup');

SELECT id FROM articles
 WHERE search_vector @@ websearch_to_tsquery('simple', 'banjir');
-- Harapan: x1

-- 3. Urutan dengan created_at kembar (kedua baris dibuat di transaksi yang sama)
SELECT id FROM articles ORDER BY created_at DESC, id DESC;
-- Harapan: x2 lalu x1 (id menjadi penentu urutan)

-- 4. Siklus edit
SELECT created_at AS created_before FROM articles WHERE id = 'x1' \gset
UPDATE articles SET title = 'Macet Jakarta', updated_at = now() WHERE id = 'x1';

SELECT id FROM articles
 WHERE search_vector @@ websearch_to_tsquery('simple', 'banjir');
-- Harapan: 0 baris (kata lama hilang)

SELECT id FROM articles
 WHERE search_vector @@ websearch_to_tsquery('simple', 'macet');
-- Harapan: x1 (kata baru muncul)

SELECT created_at = :'created_before' AS created_at_tetap
  FROM articles WHERE id = 'x1';
-- Harapan: t

-- 5. Apakah index bisa dipakai? (hanya cek kelayakan, BUKAN bukti performa)
SET LOCAL enable_seqscan = off;
EXPLAIN SELECT id FROM articles
  WHERE search_vector @@ websearch_to_tsquery('simple', 'macet');
EXPLAIN SELECT id FROM articles ORDER BY created_at DESC, id DESC LIMIT 20;
EXPLAIN SELECT id FROM articles
  WHERE author_id = 'a1' ORDER BY created_at DESC, id DESC LIMIT 20;
-- Harapan: berturut-turut memakai idx_articles_search, idx_articles_created,
-- idx_articles_author_created

ROLLBACK;