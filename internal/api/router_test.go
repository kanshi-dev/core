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
			response, err := NewServer(nil, nil, tt.ping, "dashboard-secret", "").App.Test(
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

func TestAPIAuthenticationAndCORS(t *testing.T) {
	app := NewServer(nil, nil, nil, "dashboard-secret", "https://dashboard.example.com").App
	for _, tt := range []struct {
		name, authorization string
		want                int
	}{
		{"missing", "", 401},
		{"malformed", "dashboard-secret", 401},
		{"incorrect", "Bearer wrong", 401},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/agents", nil)
			req.Header.Set("Authorization", tt.authorization)
			response, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != tt.want {
				t.Fatalf("got %d, want %d", response.StatusCode, tt.want)
			}
		})
	}
	if !authorized("Bearer dashboard-secret", "dashboard-secret") {
		t.Fatal("valid bearer token was rejected")
	}

	for _, origin := range []struct{ value, want string }{
		{"https://dashboard.example.com", "https://dashboard.example.com"},
		{"https://evil.example.com", ""},
	} {
		req := httptest.NewRequest("OPTIONS", "/api/v1/agents", nil)
		req.Header.Set("Origin", origin.value)
		req.Header.Set("Access-Control-Request-Method", "GET")
		response, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if got := response.Header.Get("Access-Control-Allow-Origin"); got != origin.want {
			t.Fatalf("origin %q: got %q, want %q", origin.value, got, origin.want)
		}
	}
}
