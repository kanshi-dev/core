package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	host := getenv("DB_HOST", "localhost")
	port := getenv("DB_PORT", "5432")
	user := getenv("DB_USER", "kanshi")
	pass := getenv("DB_PASSWORD", "kanshi")
	name := getenv("DB_NAME", "kanshi")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, pass, host, port, name,
	)

	var pool *pgxpool.Pool
	var err error

	for i := range 10 {
		pool, err = pgxpool.New(ctx, dsn)
		if err == nil {
			err = pool.Ping(ctx)
			if err == nil {
				return pool, nil
			}
			pool.Close()
		}

		log.Printf("Failed to connect to database (attempt %d/10): %v. Retrying in 5 seconds...", i+1, err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}

	return nil, fmt.Errorf("could not connect to database after 10 attempts: %w", err)
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
