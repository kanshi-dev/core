package db

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations.sql
var migrations string

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, migrations)
	return err
}

func ConfigureTelemetryRetention(ctx context.Context, pool *pgxpool.Pool, traceRetention, logRetention time.Duration) error {
	for table, retention := range map[string]time.Duration{
		"otel_spans": traceRetention,
		"otel_logs":  logRetention,
	} {
		if _, err := pool.Exec(ctx, "SELECT remove_retention_policy($1::regclass, if_exists => TRUE)", table); err != nil {
			return fmt.Errorf("remove %s retention policy: %w", table, err)
		}
		if _, err := pool.Exec(ctx, "SELECT add_retention_policy($1::regclass, $2::interval, if_not_exists => TRUE)", table, retention.String()); err != nil {
			return fmt.Errorf("add %s retention policy: %w", table, err)
		}
	}
	return nil
}
