package article

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCursorRoundTripAndInvalidValues(t *testing.T) {
	want := Cursor{CreatedAt: time.Date(2026, 11, 8, 12, 0, 0, 123456000, time.UTC), ID: "01JABC"}
	encoded, err := EncodeCursor(want)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) || got.ID != want.ID {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
	for _, value := range []string{"%%", "bm90LWpzb24", "eyJjcmVhdGVkX2F0IjoxfQ"} {
		if _, err := DecodeCursor(value); err == nil {
			t.Errorf("DecodeCursor(%q) succeeded", value)
		}
	}
}

func TestCreateValidation(t *testing.T) {
	svc := NewService(createStub{})
	for _, input := range []CreateInput{
		{AuthorID: "a", Title: "", Body: "body"}, {AuthorID: "a", Title: strings.Repeat("a", 301), Body: "body"},
		{AuthorID: "a", Title: "title", Body: ""}, {AuthorID: "a", Title: "title", Body: strings.Repeat("b", 100001)}, {Title: "title", Body: "body"},
	} {
		if _, err := svc.Create(context.Background(), input); err == nil {
			t.Errorf("invalid input accepted: %#v", input)
		}
	}
	for _, input := range []CreateInput{{AuthorID: "a", Title: strings.Repeat("a", 300), Body: "body"}, {AuthorID: "a", Title: strings.Repeat("é", 300), Body: "body"}} {
		if _, err := svc.Create(context.Background(), input); err != nil {
			t.Errorf("valid input rejected: %v", err)
		}
	}
}

type createStub struct{}

func (createStub) ResolveAuthorIDs(context.Context, string) ([]string, error) { return nil, nil }
func (createStub) CreateArticle(context.Context, string, string, string, string) (time.Time, error) {
	return time.Now(), nil
}
func (createStub) ListArticles(context.Context, ListParams) ([]Article, bool, error) {
	return nil, false, nil
}
