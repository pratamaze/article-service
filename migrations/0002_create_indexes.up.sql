CREATE INDEX idx_authors_name_norm
  ON authors (name_normalized);

CREATE INDEX idx_articles_created
  ON articles (created_at DESC, id DESC);

CREATE INDEX idx_articles_author_created
  ON articles (author_id, created_at DESC, id DESC);

CREATE INDEX idx_articles_search
  ON articles USING GIN (search_vector);