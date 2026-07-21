package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestHealth(t *testing.T) {
	tests := []struct {
		name   string
		ping   func(context.Context) error
		code   int
		status string
		db     string
	}{
		{"ok", func(context.Context) error { return nil }, 200, "ok", "ok"},
		{"error", func(context.Context) error { return errors.New("down") }, 503, "degraded", "error"},
		{"timeout", func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }, 503, "degraded", "error"},
		{"absent", nil, 503, "degraded", "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := NewServer(nil, nil, tt.ping).App.Test(
				httptest.NewRequest("GET", "/health", nil),
				fiber.TestConfig{Timeout: 2 * time.Second},
			)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()

			var body struct {
				Status string `json:"status"`
				DB     string `json:"db"`
			}
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != tt.code || body.Status != tt.status || body.DB != tt.db {
				t.Fatalf("got code=%d body=%+v", response.StatusCode, body)
			}
		})
	}
}
