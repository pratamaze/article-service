CREATE TABLE authors (
  id              text PRIMARY KEY,
  name            text NOT NULL,
  name_normalized text GENERATED ALWAYS AS (
    lower(btrim(regexp_replace(name, '\s+', ' ', 'g')))
  ) STORED,
  CONSTRAINT authors_name_not_blank
    CHECK (btrim(regexp_replace(name, '\s+', ' ', 'g')) <> '')
);

CREATE TABLE articles (
  id         text PRIMARY KEY,
  author_id  text NOT NULL REFERENCES authors (id),
  title      text NOT NULL,
  body       text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  search_vector tsvector GENERATED ALWAYS AS (
    to_tsvector('simple'::regconfig, coalesce(title, '') || ' ' || coalesce(body, ''))
  ) STORED,
  CONSTRAINT articles_title_len CHECK (char_length(title) BETWEEN 1 AND 300),
  CONSTRAINT articles_body_len  CHECK (char_length(body)  BETWEEN 1 AND 100000)
);