package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pratamaze/article-service/internal/article"
)

func TestRepositorySmoke(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is empty")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	repository := NewRepository(pool)
	firstPage, _, err := repository.ListArticles(ctx, article.ListParams{Limit: 2})
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(firstPage) != 2 {
		t.Fatalf("first page has %d articles, want 2", len(firstPage))
	}
	if !isDescending(firstPage[0], firstPage[1]) {
		t.Fatalf("first page is not descending: (%s, %s), (%s, %s)", firstPage[0].CreatedAt, firstPage[0].ID, firstPage[1].CreatedAt, firstPage[1].ID)
	}

	cursor := article.Cursor{CreatedAt: firstPage[1].CreatedAt, ID: firstPage[1].ID}
	secondPage, _, err := repository.ListArticles(ctx, article.ListParams{Limit: 2, Cursor: &cursor})
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	for _, second := range secondPage {
		for _, first := range firstPage {
			if second.ID == first.ID {
				t.Fatalf("second page repeats first-page id %q", second.ID)
			}
		}
	}

	syndicationIDs, err := repository.ResolveAuthorIDs(ctx, "syndication")
	if err != nil {
		t.Fatalf("resolve syndication: %v", err)
	}
	if len(syndicationIDs) != 2 {
		t.Fatalf("syndication resolved %d ids, want 2", len(syndicationIDs))
	}

	_, err = repository.ResolveAuthorIDs(ctx, "Admin")
	if !errors.Is(err, article.ErrAmbiguousAuthor) {
		t.Fatalf("resolve Admin error = %v, want ErrAmbiguousAuthor", err)
	}
}

func isDescending(left, right article.Article) bool {
	return left.CreatedAt.After(right.CreatedAt) ||
		(left.CreatedAt.Equal(right.CreatedAt) && left.ID > right.ID)
}
