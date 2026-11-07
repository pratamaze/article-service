package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pratamaze/article-service/internal/article"
)

func TestListLimitValidation(t *testing.T) {
	h := NewHandler(article.NewService(noopRepository{}), time.Second, slog.Default()).Routes()
	for _, tc := range []struct {
		query string
		want  int
	}{{"", 200}, {"?limit=100", 200}, {"?limit=0", 400}, {"?limit=101", 400}, {"?limit=abc", 400}} {
		r := httptest.NewRequest(http.MethodGet, "/articles"+tc.query, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Errorf("%s: status %d, want %d", tc.query, w.Code, tc.want)
		}
	}
}

type noopRepository struct{}

func (noopRepository) ResolveAuthorIDs(context.Context, string) ([]string, error) { return nil, nil }
func (noopRepository) CreateArticle(context.Context, string, string, string, string) (time.Time, error) {
	return time.Time{}, nil
}
func (noopRepository) ListArticles(context.Context, article.ListParams) ([]article.Article, bool, error) {
	return nil, false, nil
}
