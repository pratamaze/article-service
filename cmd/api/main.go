package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("gagal membuat pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	var version string
	if err := pool.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		slog.Error("gagal query", "err", err)
		os.Exit(1)
	}
	fmt.Println("terhubung:", version)
}