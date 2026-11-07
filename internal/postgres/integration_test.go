package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
	"github.com/pratamaze/article-service/internal/article"
	"github.com/pratamaze/article-service/internal/httpapi"
)

type integrationEnv struct {
	pool    *pgxpool.Pool
	repo    *Repository
	handler http.Handler
}

func newIntegrationEnv(t *testing.T) integrationEnv {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is empty")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	if config.ConnConfig.Database != "articles_test" {
		t.Fatalf("TEST_DATABASE_URL database = %q, want articles_test", config.ConnConfig.Database)
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(pool.Close)
	repo := NewRepository(pool)
	return integrationEnv{pool: pool, repo: repo, handler: httpapi.NewHandler(article.NewService(repo), time.Second, slog.New(slog.NewTextHandler(os.Stderr, nil))).Routes()}
}

func testID() string { return ulid.Make().String() }
func (e integrationEnv) author(t *testing.T, name string) string {
	t.Helper()
	id := testID()
	if _, err := e.pool.Exec(context.Background(), "INSERT INTO authors (id, name) VALUES ($1, $2)", id, name); err != nil {
		t.Fatalf("insert author: %v", err)
	}
	return id
}
func (e integrationEnv) article(t *testing.T, authorID, title, body string) (string, time.Time) {
	t.Helper()
	id := testID()
	created, err := e.repo.CreateArticle(context.Background(), id, authorID, title, body)
	if err != nil {
		t.Fatalf("create article: %v", err)
	}
	return id, created
}
func (e integrationEnv) get(t *testing.T, raw string) listResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/articles"+raw, nil)
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d: %s", raw, rec.Code, rec.Body.String())
	}
	var response listResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	return response
}
func status(e integrationEnv, method, path, body string) int {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec.Code
}

type listResponse struct {
	Items []struct {
		ID        string    `json:"id"`
		AuthorID  string    `json:"author_id"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"items"`
	Next string `json:"next_cursor"`
	More bool   `json:"has_more"`
}

func ids(items []struct {
	ID        string    `json:"id"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
}) []string {
	out := make([]string, len(items))
	for i := range items {
		out[i] = items[i].ID
	}
	return out
}

func TestIntegrationI1KeysetNoDuplicateOrMissing(t *testing.T) {
	e := newIntegrationEnv(t)
	authorID := e.author(t, "keyset")
	wantTime := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 55; i++ {
		id, _ := e.article(t, authorID, fmt.Sprintf("keyset %02d", i), "body")
		if i < 50 {
			if _, err := e.pool.Exec(context.Background(), "UPDATE articles SET created_at = $2 WHERE id = $1", id, wantTime); err != nil {
				t.Fatal(err)
			}
		}
	}
	rows, err := e.pool.Query(context.Background(), "SELECT id FROM articles ORDER BY created_at DESC, id DESC")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var expected []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		expected = append(expected, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	var got []string
	cursor := ""
	for {
		path := "?limit=20"
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		page := e.get(t, path)
		got = append(got, ids(page.Items)...)
		if !page.More {
			break
		}
		cursor = page.Next
		if cursor == "" {
			t.Fatal("has_more without next_cursor")
		}
	}
	if len(got) != len(expected) {
		t.Fatalf("got %d ids, want %d", len(got), len(expected))
	}
	seen := map[string]bool{}
	for _, id := range got {
		if seen[id] {
			t.Fatalf("duplicate id %s", id)
		}
		seen[id] = true
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("position %d = %s, want %s", i, got[i], expected[i])
		}
	}
}

func TestIntegrationI2AuthorNamesAndAmbiguity(t *testing.T) {
	e := newIntegrationEnv(t)
	first, second := e.author(t, "Budi"), e.author(t, " BUDI ")
	firstArticle, _ := e.article(t, first, "one", "body")
	secondArticle, _ := e.article(t, second, "two", "body")
	page := e.get(t, "?author=budi")
	got := map[string]bool{}
	authors := map[string]bool{}
	for _, item := range page.Items {
		got[item.ID] = true
		authors[item.AuthorID] = true
	}
	if !got[firstArticle] || !got[secondArticle] || !authors[first] || !authors[second] {
		t.Fatal("normalized Budi did not return both authors' articles")
	}
	for i := 0; i < 21; i++ {
		e.author(t, "Ambiguous Test Name")
	}
	if code := status(e, http.MethodGet, "/articles?author=Ambiguous+Test+Name", ""); code != http.StatusBadRequest {
		t.Fatalf("ambiguous status = %d, want 400", code)
	}
	if page := e.get(t, "?author=does-not-exist"); len(page.Items) != 0 {
		t.Fatalf("missing author returned %d items", len(page.Items))
	}
	page = e.get(t, "?author_id="+first)
	if len(page.Items) != 1 || page.Items[0].ID != firstArticle {
		t.Fatalf("author_id results = %v, want only %s", ids(page.Items), firstArticle)
	}
}

func TestIntegrationI3AuthorNormalization(t *testing.T) {
	e := newIntegrationEnv(t)
	var baseline []string
	for _, name := range []string{"Budi", "budi", " BUDI ", "Budi   "} {
		page := e.get(t, "?author="+url.QueryEscape(name))
		got := ids(page.Items)
		if baseline == nil {
			baseline = got
		} else if strings.Join(got, ",") != strings.Join(baseline, ",") {
			t.Fatalf("%q ids differ: %v vs %v", name, got, baseline)
		}
	}
}

func TestIntegrationI4UpdateRefreshesSearchWithoutMovingPosition(t *testing.T) {
	e := newIntegrationEnv(t)
	authorID := e.author(t, "editor")
	id, created := e.article(t, authorID, "banjir jakarta", "body")
	if !contains(e.get(t, "?query=banjir").Items, id) {
		t.Fatal("banjir search did not find article")
	}
	before := allIDs(t, e)
	if err := e.repo.UpdateArticle(context.Background(), id, "macet jakarta", "body"); err != nil {
		t.Fatal(err)
	}
	if contains(e.get(t, "?query=banjir").Items, id) || !contains(e.get(t, "?query=macet").Items, id) {
		t.Fatal("search vector was not refreshed")
	}
	var after time.Time
	if err := e.pool.QueryRow(context.Background(), "SELECT created_at FROM articles WHERE id = $1", id).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if !created.Equal(after) {
		t.Fatalf("created_at changed: %s -> %s", created, after)
	}
	if strings.Join(before, ",") != strings.Join(allIDs(t, e), ",") {
		t.Fatal("unfiltered position changed after edit")
	}
	if err := e.repo.UpdateArticle(context.Background(), testID(), "x", "x"); err != article.ErrNotFound {
		t.Fatalf("missing update error = %v", err)
	}
}

func TestIntegrationI5BadInputDoesNotReturn500(t *testing.T) {
	e := newIntegrationEnv(t)
	for _, tc := range []struct {
		method, path, body string
		want               int
	}{{http.MethodGet, "/articles?cursor=not-a-cursor", "", 400}, {http.MethodGet, "/articles?limit=1000", "", 400}, {http.MethodGet, "/articles?author=a&author_id=b", "", 400}, {http.MethodPost, "/articles", `{"author_id":"missing","title":"title","body":"body"}`, 422}, {http.MethodPost, "/articles", `{`, 400}, {http.MethodPost, "/articles", `{"author_id":"missing","title":"title","body":"` + strings.Repeat("x", 100001) + `"}`, 400}} {
		if got := status(e, tc.method, tc.path, tc.body); got != tc.want {
			t.Errorf("%s %s: status %d, want %d", tc.method, tc.path, got, tc.want)
		}
	}
}

func TestIntegrationI6PoolRuntimeSettings(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is empty")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if config.ConnConfig.Database != "articles_test" {
		t.Fatalf("TEST_DATABASE_URL database = %q, want articles_test", config.ConnConfig.Database)
	}
	pool, err := NewPool(context.Background(), databaseURL, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	var plan, timeout string
	if err := conn.QueryRow(context.Background(), "SHOW plan_cache_mode").Scan(&plan); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(context.Background(), "SHOW statement_timeout").Scan(&timeout); err != nil {
		t.Fatal(err)
	}
	if plan != "force_custom_plan" || timeout != "3s" {
		t.Fatalf("settings = plan_cache_mode %q, statement_timeout %q", plan, timeout)
	}
}

func contains(items []struct {
	ID        string    `json:"id"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
}, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}
func allIDs(t *testing.T, e integrationEnv) []string {
	t.Helper()
	rows, err := e.pool.Query(context.Background(), "SELECT id FROM articles ORDER BY created_at DESC, id DESC")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		result = append(result, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}
