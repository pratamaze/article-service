// Package httpapi exposes the article service through HTTP.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pratamaze/article-service/internal/article"
)

type Handler struct {
	service *article.Service
	timeout time.Duration
	logger  *slog.Logger
}

func NewHandler(service *article.Service, timeout time.Duration, logger *slog.Logger) *Handler {
	return &Handler{service: service, timeout: timeout, logger: logger}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/articles", h.articles)
	return mux
}

func (h *Handler) articles(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()
	r = r.WithContext(ctx)
	switch r.Method {
	case http.MethodPost:
		h.createArticle(w, r)
	case http.MethodGet:
		h.listArticles(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method tidak diizinkan")
	}
}

func (h *Handler) createArticle(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	var input article.CreateInput
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&input); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusBadRequest, "body request melebihi 1 MB")
		} else {
			writeError(w, http.StatusBadRequest, "JSON request tidak valid")
		}
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "JSON request harus satu objek")
		return
	}
	item, err := h.service.Create(r.Context(), input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, itemResponse(item))
}

func (h *Handler) listArticles(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	if values.Has("author") && values.Has("author_id") {
		writeError(w, http.StatusBadRequest, "author dan author_id tidak boleh digunakan bersamaan")
		return
	}
	limit := 20
	if raw, present := values["limit"]; present {
		if len(raw) != 1 {
			writeError(w, http.StatusBadRequest, "limit tidak valid")
			return
		}
		parsed, err := strconv.Atoi(raw[0])
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, "limit harus antara 1 dan 100")
			return
		}
		limit = parsed
	}
	var cursor *article.Cursor
	if raw := values.Get("cursor"); raw != "" {
		decoded, err := article.DecodeCursor(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "cursor tidak valid")
			return
		}
		cursor = &decoded
	}
	items, next, more, err := h.service.List(r.Context(), article.ListInput{Query: values.Get("query"), Author: values.Get("author"), AuthorID: values.Get("author_id"), Limit: limit, Cursor: cursor})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	responseItems := make([]articleResponse, len(items))
	for i, item := range items {
		responseItems[i] = itemResponse(item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": responseItems, "next_cursor": next, "has_more": more})
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusServiceUnavailable, "request melebihi batas waktu")
	case errors.Is(err, article.ErrAuthorNotFound):
		writeError(w, http.StatusUnprocessableEntity, "author_id tidak ditemukan")
	case errors.Is(err, article.ErrAmbiguousAuthor):
		writeError(w, http.StatusBadRequest, "nama author cocok dengan lebih dari 20 author")
	default:
		if strings.Contains(err.Error(), "wajib") || strings.Contains(err.Error(), "harus") || strings.Contains(err.Error(), "tidak boleh") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("request artikel gagal", "err", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
	}
}

type articleResponse struct {
	ID        string    `json:"id"`
	AuthorID  string    `json:"author_id"`
	Title     string    `json:"title"`
	Excerpt   string    `json:"excerpt,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func itemResponse(item article.Article) articleResponse {
	return articleResponse{ID: item.ID, AuthorID: item.AuthorID, Title: item.Title, Excerpt: item.Excerpt, CreatedAt: item.CreatedAt}
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
