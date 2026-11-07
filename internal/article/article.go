// Package article contains the domain types used by the article repository.
package article

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/oklog/ulid/v2"
)

var (
	ErrAuthorNotFound  = errors.New("author not found")
	ErrNotFound        = errors.New("article not found")
	ErrAmbiguousAuthor = errors.New("author name is ambiguous")
)

// Article is the compact representation returned by an article listing.
type Article struct {
	ID        string
	AuthorID  string
	Title     string
	Excerpt   string
	CreatedAt time.Time
}

// Cursor identifies the final item of a page in the deterministic article order.
type Cursor struct {
	CreatedAt time.Time
	ID        string
}

// ListParams controls a keyset-paginated article listing.
type ListParams struct {
	Query     string
	AuthorIDs []string
	Limit     int
	Cursor    *Cursor
}

// Repository is the storage dependency used by Service.
type Repository interface {
	ResolveAuthorIDs(ctx context.Context, name string) ([]string, error)
	CreateArticle(ctx context.Context, id, authorID, title, body string) (time.Time, error)
	ListArticles(ctx context.Context, params ListParams) ([]Article, bool, error)
}

// Service applies article business rules before delegating to the repository.
type Service struct {
	repository Repository
}

// NewService creates an article service.
func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

// CreateInput is the client-provided portion of an article.
type CreateInput struct {
	AuthorID string `json:"author_id"`
	Title    string `json:"title"`
	Body     string `json:"body"`
}

// Create generates an ID and creates a validated article.
func (s *Service) Create(ctx context.Context, input CreateInput) (Article, error) {
	if strings.TrimSpace(input.AuthorID) == "" {
		return Article{}, errors.New("author_id wajib diisi")
	}
	if n := utf8.RuneCountInString(input.Title); n < 1 || n > 300 {
		return Article{}, errors.New("title harus terdiri dari 1 sampai 300 karakter")
	}
	if n := utf8.RuneCountInString(input.Body); n < 1 || n > 100000 {
		return Article{}, errors.New("body harus terdiri dari 1 sampai 100000 karakter")
	}

	id := ulid.Make().String()
	createdAt, err := s.repository.CreateArticle(ctx, id, input.AuthorID, input.Title, input.Body)
	if err != nil {
		return Article{}, err
	}
	return Article{ID: id, AuthorID: input.AuthorID, Title: input.Title, CreatedAt: createdAt}, nil
}

// ListInput is the public query shape accepted by the service.
type ListInput struct {
	Query    string
	Author   string
	AuthorID string
	Limit    int
	Cursor   *Cursor
}

// List returns an ordered page and a cursor for its final item when another page exists.
func (s *Service) List(ctx context.Context, input ListInput) ([]Article, string, bool, error) {
	if input.Limit < 1 || input.Limit > 100 {
		return nil, "", false, errors.New("limit harus antara 1 dan 100")
	}
	if input.Author != "" && input.AuthorID != "" {
		return nil, "", false, errors.New("author dan author_id tidak boleh digunakan bersamaan")
	}

	params := ListParams{Query: strings.TrimSpace(input.Query), Limit: input.Limit, Cursor: input.Cursor}
	if input.Author != "" {
		ids, err := s.repository.ResolveAuthorIDs(ctx, input.Author)
		if err != nil {
			return nil, "", false, err
		}
		if len(ids) == 0 {
			return []Article{}, "", false, nil
		}
		params.AuthorIDs = ids
	} else if input.AuthorID != "" {
		params.AuthorIDs = []string{input.AuthorID}
	}

	items, hasMore, err := s.repository.ListArticles(ctx, params)
	if err != nil {
		return nil, "", false, err
	}
	if !hasMore || len(items) == 0 {
		return items, "", hasMore, nil
	}
	nextCursor, err := EncodeCursor(Cursor{CreatedAt: items[len(items)-1].CreatedAt, ID: items[len(items)-1].ID})
	if err != nil {
		return nil, "", false, err
	}
	return items, nextCursor, true, nil
}

type encodedCursor struct {
	CreatedAt int64  `json:"created_at"`
	ID        string `json:"id"`
}

// EncodeCursor serializes a keyset cursor using microseconds since Unix epoch.
func EncodeCursor(cursor Cursor) (string, error) {
	if cursor.ID == "" {
		return "", errors.New("cursor id kosong")
	}
	payload, err := json.Marshal(encodedCursor{CreatedAt: cursor.CreatedAt.UnixMicro(), ID: cursor.ID})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

// DecodeCursor parses a URL-safe cursor containing microseconds since Unix epoch.
func DecodeCursor(value string) (Cursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, errors.New("cursor tidak valid")
	}
	var encoded encodedCursor
	if err := json.Unmarshal(payload, &encoded); err != nil || encoded.ID == "" {
		return Cursor{}, errors.New("cursor tidak valid")
	}
	return Cursor{CreatedAt: time.UnixMicro(encoded.CreatedAt).UTC(), ID: encoded.ID}, nil
}
