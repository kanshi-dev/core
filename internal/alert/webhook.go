package alert

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kanshi-dev/core/internal/db"
)

// Dispatcher delivers alert transitions to a global list of webhook URLs.
type Dispatcher struct {
	urls    []string
	secret  string
	q       *db.Queries
	client  *http.Client
	backoff []time.Duration // waits between retries; attempts = len(backoff)+1
}

func NewDispatcher(urls []string, secret string, q *db.Queries) *Dispatcher {
	return &Dispatcher{
		urls:    urls,
		secret:  secret,
		q:       q,
		client:  &http.Client{Timeout: 10 * time.Second},
		backoff: []time.Duration{time.Second, 3 * time.Second},
	}
}

func (d *Dispatcher) enabled() bool { return d != nil && len(d.urls) > 0 }

// payload is the versioned webhook body. Bump Version on any shape change.
type payload struct {
	Version   int       `json:"version"`
	Event     string    `json:"event"` // firing | resolved
	RuleID    int64     `json:"ruleId"`
	RuleName  string    `json:"ruleName"`
	Metric    string    `json:"metric"`
	AgentID   string    `json:"agentId"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Timestamp time.Time `json:"timestamp"`
}

// Deliver sends one transition to every configured URL and records an aggregate
// delivery status on the event row.
// ponytail: aggregate status across URLs; per-URL delivery rows only if
// multi-endpoint visibility is needed.
func (d *Dispatcher) Deliver(ctx context.Context, r db.AlertRule, ev db.AlertEvent) {
	body, err := json.Marshal(payload{
		Version:   1,
		Event:     ev.State,
		RuleID:    r.ID,
		RuleName:  r.Name,
		Metric:    r.Metric,
		AgentID:   ev.AgentID,
		Value:     ev.Value,
		Threshold: r.Threshold,
		Timestamp: ev.CreatedAt.Time,
	})
	if err != nil {
		d.markStatus(ctx, ev.ID, "failed", err.Error())
		return
	}

	sig := ""
	if d.secret != "" {
		mac := hmac.New(sha256.New, []byte(d.secret))
		mac.Write(body)
		sig = "sha256=" + hex.EncodeToString(mac.Sum(nil))
	}

	var failures []string
	for _, url := range d.urls {
		if err := d.deliverOne(ctx, url, body, sig); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", url, err))
		}
	}
	if len(failures) > 0 {
		d.markStatus(ctx, ev.ID, "failed", strings.Join(failures, "; "))
		return
	}
	d.markStatus(ctx, ev.ID, "delivered", "")
}

// deliverOne POSTs the body to one URL, retrying only on failure up to the
// bounded backoff. A 2xx succeeds and is never retried.
func (d *Dispatcher) deliverOne(ctx context.Context, url string, body []byte, sig string) error {
	attempts := len(d.backoff) + 1
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(d.backoff[attempt-1]):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err // malformed URL: unretryable
		}
		req.Header.Set("Content-Type", "application/json")
		if sig != "" {
			req.Header.Set("X-Kanshi-Signature", sig)
		}
		resp, err := d.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("status %d", resp.StatusCode)
	}
	return lastErr
}

func (d *Dispatcher) markStatus(ctx context.Context, id int64, status, errMsg string) {
	if d.q == nil {
		return
	}
	if err := d.q.UpdateEventWebhookStatus(ctx, db.UpdateEventWebhookStatusParams{
		ID:            id,
		WebhookStatus: status,
		WebhookError:  errText(errMsg),
	}); err != nil {
		log.Printf("alert: update webhook status failed (event %d): %v", id, err)
	}
}

func errText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	if len(s) > 500 {
		s = s[:500]
	}
	return pgtype.Text{String: s, Valid: true}
}
