package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/pratamaze/article-service/internal/article"
	"github.com/pratamaze/article-service/internal/httpapi"
	"github.com/pratamaze/article-service/internal/postgres"
)

func main() {
	logger := slog.Default()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL wajib diisi")
		os.Exit(1)
	}
	maxConns, err := envPositiveInt("DB_MAX_CONNS", 20)
	if err != nil {
		logger.Error("konfigurasi DB_MAX_CONNS tidak valid", "err", err)
		os.Exit(1)
	}
	timeout, err := envPositiveDuration("REQUEST_TIMEOUT", 5*time.Second)
	if err != nil {
		logger.Error("konfigurasi REQUEST_TIMEOUT tidak valid", "err", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	pool, err := postgres.NewPool(ctx, databaseURL, int32(maxConns))
	if err != nil {
		logger.Error("gagal membuat pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		logger.Error("gagal menghubungi database", "err", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              ":" + envOrDefault("PORT", "8080"),
		Handler:           httpapi.NewHandler(article.NewService(postgres.NewRepository(pool)), timeout, logger).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	logger.Info("API berjalan", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server berhenti", "err", err)
		os.Exit(1)
	}
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envPositiveInt(name string, fallback int) (int, error) {
	parsed, err := strconv.Atoi(envOrDefault(name, strconv.Itoa(fallback)))
	if err != nil || parsed < 1 {
		return 0, errors.New("harus bilangan bulat positif")
	}
	return parsed, nil
}

func envPositiveDuration(name string, fallback time.Duration) (time.Duration, error) {
	parsed, err := time.ParseDuration(envOrDefault(name, fallback.String()))
	if err != nil || parsed <= 0 {
		return 0, errors.New("harus durasi positif, misalnya 5s")
	}
	return parsed, nil
}
