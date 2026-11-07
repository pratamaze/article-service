# Output mentah EXPLAIN halaman pertama

Diambil melalui `make psql` pada 8 November 2026. Tiap blok memuat run 1 lalu
run 2, dengan SQL literal yang sama dengan cabang `ListArticles` tanpa cursor
dan `LIMIT 21`.

## 1. author-giant, tanpa query

```text
run 1
Limit  (cost=0.42..25.20 rows=21 width=117) (actual time=0.031..0.118 rows=21 loops=1)
  Buffers: shared hit=13
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..352599.77 rows=298924 width=117) (actual time=0.029..0.115 rows=21 loops=1)
        Filter: (author_id = 'author-giant'::text)
        Rows Removed by Filter: 43
        Buffers: shared hit=13
Planning:
  Buffers: shared hit=181
Planning Time: 0.667 ms
Execution Time: 0.147 ms

run 2
Limit  (cost=0.42..25.20 rows=21 width=117) (actual time=0.011..0.076 rows=21 loops=1)
  Buffers: shared hit=13
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..352599.77 rows=298924 width=117) (actual time=0.011..0.074 rows=21 loops=1)
        Filter: (author_id = 'author-giant'::text)
        Rows Removed by Filter: 43
        Buffers: shared hit=13
Planning Time: 0.068 ms
Execution Time: 0.092 ms
```

## 2. author-000100, tanpa query

```text
run 1
Limit  (cost=0.42..86.72 rows=21 width=117) (actual time=0.083..8.643 rows=21 loops=1)
  Buffers: shared hit=2 read=22
  ->  Index Scan using idx_articles_author_created on articles  (cost=0.42..152.47 rows=37 width=117) (actual time=0.082..8.636 rows=21 loops=1)
        Index Cond: (author_id = 'author-000100'::text)
        Buffers: shared hit=2 read=22
Planning Time: 0.044 ms
Execution Time: 8.660 ms

run 2
Limit  (cost=0.42..86.72 rows=21 width=117) (actual time=0.033..0.099 rows=21 loops=1)
  Buffers: shared hit=24
  ->  Index Scan using idx_articles_author_created on articles  (cost=0.42..152.47 rows=37 width=117) (actual time=0.032..0.097 rows=21 loops=1)
        Index Cond: (author_id = 'author-000100'::text)
        Buffers: shared hit=24
Planning Time: 0.131 ms
Execution Time: 0.119 ms
```

## 3. author-giant, query kebijakan

```text
run 1
Limit  (cost=0.42..2386.32 rows=21 width=117) (actual time=0.127..2.497 rows=21 loops=1)
  Buffers: shared hit=937
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..354362.41 rows=3119 width=117) (actual time=0.126..2.494 rows=21 loops=1)
        Filter: ((search_vector @@ '''kebijakan'''::tsquery) AND (author_id = 'author-giant'::text))
        Rows Removed by Filter: 6166
        Buffers: shared hit=937
Planning:
  Buffers: shared hit=30
Planning Time: 0.682 ms
Execution Time: 2.519 ms

run 2
Limit  (cost=0.42..2386.32 rows=21 width=117) (actual time=0.104..2.623 rows=21 loops=1)
  Buffers: shared hit=937
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..354362.41 rows=3119 width=117) (actual time=0.103..2.619 rows=21 loops=1)
        Filter: ((search_vector @@ '''kebijakan'''::tsquery) AND (author_id = 'author-giant'::text))
        Rows Removed by Filter: 6166
        Buffers: shared hit=937
Planning:
  Buffers: shared hit=1
Planning Time: 0.149 ms
Execution Time: 2.644 ms
```

## 4. author-000100, query kebijakan

```text
run 1
Limit  (cost=90.64..90.65 rows=1 width=117) (actual time=2.774..2.776 rows=0 loops=1)
  Buffers: shared hit=13 read=5
  ->  Sort  (cost=90.64..90.65 rows=1 width=117) (actual time=2.773..2.774 rows=0 loops=1)
        Sort Key: created_at DESC, id DESC
        Sort Method: quicksort  Memory: 25kB
        Buffers: shared hit=13 read=5
        ->  Bitmap Heap Scan on articles  (cost=86.62..90.63 rows=1 width=117) (actual time=2.733..2.734 rows=0 loops=1)
              Recheck Cond: ((author_id = 'author-000100'::text) AND (search_vector @@ '''kebijakan'''::tsquery))
              Buffers: shared hit=7 read=5
              ->  BitmapAnd  (cost=86.62..86.62 rows=1 width=0) (actual time=2.730..2.731 rows=0 loops=1)
                    Buffers: shared hit=7 read=5
                    ->  Bitmap Index Scan on idx_articles_author_created  (cost=0.00..4.70 rows=37 width=0) (actual time=0.026..0.026 rows=35 loops=1)
                          Index Cond: (author_id = 'author-000100'::text)
                          Buffers: shared hit=3
                    ->  Bitmap Index Scan on idx_articles_search  (cost=0.00..81.66 rows=10442 width=0) (actual time=2.674..2.674 rows=10000 loops=1)
                          Index Cond: (search_vector @@ '''kebijakan'''::tsquery)
                          Buffers: shared hit=4 read=5
Planning:
  Buffers: shared hit=1
Planning Time: 0.158 ms
Execution Time: 2.837 ms

run 2
Limit  (cost=90.64..90.65 rows=1 width=117) (actual time=1.940..1.941 rows=0 loops=1)
  Buffers: shared hit=12
  ->  Sort  (cost=90.64..90.65 rows=1 width=117) (actual time=1.938..1.940 rows=0 loops=1)
        Sort Key: created_at DESC, id DESC
        Sort Method: quicksort  Memory: 25kB
        Buffers: shared hit=12
        ->  Bitmap Heap Scan on articles  (cost=86.62..90.63 rows=1 width=117) (actual time=1.933..1.934 rows=0 loops=1)
              Recheck Cond: ((author_id = 'author-000100'::text) AND (search_vector @@ '''kebijakan'''::tsquery))
              Buffers: shared hit=12
              ->  BitmapAnd  (cost=86.62..86.62 rows=1 width=0) (actual time=1.931..1.932 rows=0 loops=1)
                    Buffers: shared hit=12
                    ->  Bitmap Index Scan on idx_articles_author_created  (cost=0.00..4.70 rows=37 width=0) (actual time=0.020..0.021 rows=35 loops=1)
                          Index Cond: (author_id = 'author-000100'::text)
                          Buffers: shared hit=3
                    ->  Bitmap Index Scan on idx_articles_search  (cost=0.00..81.66 rows=10442 width=0) (actual time=1.833..1.834 rows=10000 loops=1)
                          Index Cond: (search_vector @@ '''kebijakan'''::tsquery)
                          Buffers: shared hit=9
Planning:
  Buffers: shared hit=1
Planning Time: 0.178 ms
Execution Time: 1.965 ms
```

## 5. author-giant, query berita

```text
run 1
Limit  (cost=0.42..62.81 rows=21 width=117) (actual time=0.025..0.125 rows=21 loops=1)
  Buffers: shared hit=28
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..354653.07 rows=119380 width=117) (actual time=0.024..0.122 rows=21 loops=1)
        Filter: ((search_vector @@ '''berita'''::tsquery) AND (author_id = 'author-giant'::text))
        Rows Removed by Filter: 153
        Buffers: shared hit=28
Planning:
  Buffers: shared hit=1
Planning Time: 0.126 ms
Execution Time: 0.136 ms

run 2
Limit  (cost=0.42..62.81 rows=21 width=117) (actual time=0.013..0.072 rows=21 loops=1)
  Buffers: shared hit=28
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..354653.07 rows=119380 width=117) (actual time=0.012..0.070 rows=21 loops=1)
        Filter: ((search_vector @@ '''berita'''::tsquery) AND (author_id = 'author-giant'::text))
        Rows Removed by Filter: 153
        Buffers: shared hit=28
Planning:
  Buffers: shared hit=1
Planning Time: 0.067 ms
Execution Time: 0.079 ms
```

## 6. tanpa author, query berita

```text
run 1
Limit  (cost=0.42..18.96 rows=21 width=117) (actual time=0.007..0.044 rows=21 loops=1)
  Buffers: shared hit=11
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..352851.74 rows=399711 width=117) (actual time=0.007..0.043 rows=21 loops=1)
        Filter: (search_vector @@ '''berita'''::tsquery)
        Rows Removed by Filter: 33
        Buffers: shared hit=11
Planning:
  Buffers: shared hit=1
Planning Time: 0.071 ms
Execution Time: 0.051 ms

run 2
Limit  (cost=0.42..18.96 rows=21 width=117) (actual time=0.007..0.040 rows=21 loops=1)
  Buffers: shared hit=11
  ->  Index Scan using idx_articles_created on articles  (cost=0.42..352851.74 rows=399711 width=117) (actual time=0.006..0.038 rows=21 loops=1)
        Filter: (search_vector @@ '''berita'''::tsquery)
        Rows Removed by Filter: 33
        Buffers: shared hit=11
Planning:
  Buffers: shared hit=1
Planning Time: 0.048 ms
Execution Time: 0.046 ms
```
