// Package postgres provides PostgreSQL-backed repositories.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pratamaze/article-service/internal/article"
)

// Repository stores articles in PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a repository using pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ResolveAuthorIDs finds at most 21 exact normalized-name matches; idx_authors_name_norm is expected.
func (r *Repository) ResolveAuthorIDs(ctx context.Context, name string) ([]string, error) {
	const query = `
SELECT id
FROM authors
WHERE name_normalized = lower(btrim(regexp_replace($1, '\s+', ' ', 'g')))
ORDER BY id
LIMIT 21`

	rows, err := r.pool.Query(ctx, query, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]string, 0, 21)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) > 20 {
		return nil, article.ErrAmbiguousAuthor
	}
	return ids, nil
}

// CreateArticle inserts an article and returns its database-assigned creation time; the authors primary key is checked.
func (r *Repository) CreateArticle(ctx context.Context, id, authorID, title, body string) (time.Time, error) {
	const query = `
INSERT INTO articles (id, author_id, title, body)
VALUES ($1, $2, $3, $4)
RETURNING created_at`

	var createdAt time.Time
	err := r.pool.QueryRow(ctx, query, id, authorID, title, body).Scan(&createdAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return time.Time{}, article.ErrAuthorNotFound
		}
		return time.Time{}, err
	}
	return createdAt, nil
}

// ListArticles returns one page in newest-first order; it uses idx_articles_created, idx_articles_author_created,
// and/or idx_articles_search according to the supplied filters.
func (r *Repository) ListArticles(ctx context.Context, params article.ListParams) ([]article.Article, bool, error) {
	queryText := strings.TrimSpace(params.Query)
	if params.Limit < 1 {
		return nil, false, errors.New("list limit must be positive")
	}
	if len(params.AuthorIDs) > 1 {
		return r.listArticlesForMultipleAuthors(ctx, params, queryText)
	}

	const selectClause = `SELECT id, author_id, title, left(body, 200), created_at FROM articles`
	const orderLimit = ` ORDER BY created_at DESC, id DESC LIMIT $%d`
	const authorWhere = ` WHERE author_id = $%d`
	const searchWhere = ` WHERE search_vector @@ websearch_to_tsquery('simple', $%d)`
	const authorSearchWhere = ` WHERE author_id = $%d AND search_vector @@ websearch_to_tsquery('simple', $%d)`

	args := make([]any, 0, 4)
	where := ""
	if len(params.AuthorIDs) == 1 && queryText == "" {
		args = append(args, params.AuthorIDs[0])
		where = format(authorWhere, len(args))
	} else if len(params.AuthorIDs) == 0 && queryText != "" {
		args = append(args, queryText)
		where = format(searchWhere, len(args))
	} else if len(params.AuthorIDs) == 1 {
		args = append(args, params.AuthorIDs[0], queryText)
		where = format(authorSearchWhere, 1, 2)
	}

	if params.Cursor != nil {
		args = append(args, params.Cursor.CreatedAt, params.Cursor.ID)
		separator := " WHERE "
		if where != "" {
			separator = " AND "
		}
		where += separator + format(`(created_at, id) < ($%d, $%d)`, len(args)-1, len(args))
	}
	args = append(args, params.Limit+1)
	query := selectClause + where + format(orderLimit, len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	items := make([]article.Article, 0, params.Limit+1)
	for rows.Next() {
		var item article.Article
		if err := rows.Scan(&item.ID, &item.AuthorID, &item.Title, &item.Excerpt, &item.CreatedAt); err != nil {
			return nil, false, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(items) > params.Limit
	if hasMore {
		items = items[:params.Limit]
	}
	return items, hasMore, nil
}

// listArticlesForMultipleAuthors takes the first limit+1 rows per author before
// merging them. This avoids scanning idx_articles_created for duplicate author
// names that include a very large author; see docs/evidence.md.
func (r *Repository) listArticlesForMultipleAuthors(ctx context.Context, params article.ListParams, queryText string) ([]article.Article, bool, error) {
	args := []any{params.AuthorIDs}
	where := " WHERE author_id = a.id"
	if queryText != "" {
		args = append(args, queryText)
		where += format(" AND search_vector @@ websearch_to_tsquery('simple', $%d)", len(args))
	}
	if params.Cursor != nil {
		args = append(args, params.Cursor.CreatedAt, params.Cursor.ID)
		where += format(" AND (created_at, id) < ($%d, $%d)", len(args)-1, len(args))
	}
	args = append(args, params.Limit+1)
	limit := len(args)
	query := fmt.Sprintf(`
SELECT x.id, x.author_id, x.title, x.excerpt, x.created_at
FROM unnest($1::text[]) AS a(id)
CROSS JOIN LATERAL (
	SELECT id, author_id, title, left(body, 200) AS excerpt, created_at
	FROM articles%s
	ORDER BY created_at DESC, id DESC
	LIMIT $%d
) AS x
ORDER BY x.created_at DESC, x.id DESC
LIMIT $%d`, where, limit, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	items := make([]article.Article, 0, params.Limit+1)
	for rows.Next() {
		var item article.Article
		if err := rows.Scan(&item.ID, &item.AuthorID, &item.Title, &item.Excerpt, &item.CreatedAt); err != nil {
			return nil, false, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(items) > params.Limit
	if hasMore {
		items = items[:params.Limit]
	}
	return items, hasMore, nil
}

// UpdateArticle changes only mutable article content and updated_at; articles_pkey is expected for lookup.
func (r *Repository) UpdateArticle(ctx context.Context, id, title, body string) error {
	const query = `
UPDATE articles
SET title = $2, body = $3, updated_at = now()
WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id, title, body)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return article.ErrNotFound
	}
	return nil
}

func format(pattern string, values ...int) string {
	args := make([]any, len(values))
	for i, value := range values {
		args[i] = value
	}
	return fmt.Sprintf(pattern, args...)
}
