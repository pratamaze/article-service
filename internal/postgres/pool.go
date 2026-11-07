package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates the application database pool with the required session settings.
func NewPool(ctx context.Context, databaseURL string, maxConns int32) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	config.MaxConns = maxConns
	config.ConnConfig.RuntimeParams["statement_timeout"] = "3000"
	config.ConnConfig.RuntimeParams["plan_cache_mode"] = "force_custom_plan"
	return pgxpool.NewWithConfig(ctx, config)
}
