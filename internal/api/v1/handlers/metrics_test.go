package handlers

import (
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kanshi-dev/core/internal/service"
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
		{"seven-day window", now.Add(-7 * 24 * time.Hour), now, fiber.StatusNoContent},
		{"window over seven days", now.Add(-7*24*time.Hour - time.Second), now, fiber.StatusBadRequest},
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

func TestAggregateBucketLimit(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	app := fiber.New()
	app.Get("/", GetAggregatedMetrics(service.NewMetricsService(nil)))

	for _, tt := range []struct {
		name, interval string
		span           time.Duration
		want           int
	}{
		{"one-hour interval", "1h", 7 * 24 * time.Hour, fiber.StatusInternalServerError},
		{"one thousand buckets", "30s", 1000 * 30 * time.Second, fiber.StatusInternalServerError},
		{"too many buckets", "30s", 1000*30*time.Second + time.Second, fiber.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			query := url.Values{"agentId": {"a"}, "name": {"cpu.used_percent"}, "interval": {tt.interval}, "from": {now.Add(-tt.span).Format(time.RFC3339)}, "to": {now.Format(time.RFC3339)}}
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
