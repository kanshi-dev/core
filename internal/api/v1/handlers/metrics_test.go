package handlers

import (
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestParseTimeRange(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tests := []struct {
		name string
		from time.Time
		to   time.Time
		want int
	}{
		{"historical one-hour window", now.Add(-48 * time.Hour), now.Add(-47 * time.Hour), fiber.StatusNoContent},
		{"window over one hour", now.Add(-2 * time.Hour), now, fiber.StatusBadRequest},
		{"equal endpoints", now.Add(-time.Hour), now.Add(-time.Hour), fiber.StatusNoContent},
		{"reversed endpoints", now, now.Add(-time.Minute), fiber.StatusBadRequest},
		{"future endpoint", now, now.Add(time.Minute), fiber.StatusBadRequest},
	}

	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		if _, _, err := parseTimeRange(c); err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		return c.SendStatus(fiber.StatusNoContent)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := url.Values{
				"from": {tt.from.Format(time.RFC3339)},
				"to":   {tt.to.Format(time.RFC3339)},
			}
			resp, err := app.Test(httptest.NewRequest("GET", "/?"+query.Encode(), nil))
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != tt.want {
				t.Fatalf("got status %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
}
