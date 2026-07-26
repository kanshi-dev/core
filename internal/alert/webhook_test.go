package alert

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kanshi-dev/core/internal/db"
)

func testEvent() (db.AlertRule, db.AlertEvent) {
	r := db.AlertRule{ID: 1, Name: "high cpu", Metric: "cpu.used_percent", Threshold: 90}
	ev := db.AlertEvent{
		ID: 7, RuleID: 1, AgentID: "agent-a", State: "firing", Value: 95,
		CreatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}
	return r, ev
}

func TestDeliverSuccessNoRetry(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewDispatcher([]string{srv.URL}, "", nil)
	r, ev := testEvent()
	d.Deliver(context.Background(), r, ev)

	if got := atomic.LoadInt32(&count); got != 1 {
		t.Fatalf("expected exactly 1 delivery, got %d", got)
	}
}

func TestDeliverFailureRetriesBounded(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	d := NewDispatcher([]string{srv.URL}, "", nil)
	d.backoff = []time.Duration{time.Millisecond, time.Millisecond} // 3 attempts, fast
	r, ev := testEvent()
	d.Deliver(context.Background(), r, ev)

	if got := atomic.LoadInt32(&count); got != 3 {
		t.Fatalf("expected 3 bounded attempts, got %d", got)
	}
}

func TestDeliverSignsBody(t *testing.T) {
	const secret = "s3cr3t"
	sigCh := make(chan string, 1)
	bodyCh := make(chan []byte, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		bodyCh <- body
		sigCh <- req.Header.Get("X-Kanshi-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewDispatcher([]string{srv.URL}, secret, nil)
	r, ev := testEvent()
	d.Deliver(context.Background(), r, ev)

	body := <-bodyCh
	gotSig := <-sigCh

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	wantSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if gotSig != wantSig {
		t.Fatalf("signature mismatch: got %q want %q", gotSig, wantSig)
	}
}
