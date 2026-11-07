// Package article contains the domain types used by the article repository.
package article

import (
	"errors"
	"time"
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
