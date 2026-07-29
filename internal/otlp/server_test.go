package otlp

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kanshi-dev/core/internal/db"
	collectorlogsv1 "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	collectortracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	logsv1 "go.opentelemetry.io/proto/otlp/logs/v1"
	resourcev1 "go.opentelemetry.io/proto/otlp/resource/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type failingDB struct{}

func (failingDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("database unavailable")
}
func (failingDB) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (failingDB) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }

func TestTraceAndLogValidation(t *testing.T) {
	now := time.Now().UTC()
	server := NewServer(nil, 7*24*time.Hour, 3*24*time.Hour)
	server.now = func() time.Time { return now }

	traceRequest := validTraceRequest(now)
	spans, err := server.parseSpans(traceRequest)
	if err != nil || len(spans) != 1 || spans[0].ServiceName != "checkout" {
		t.Fatalf("valid trace: spans=%v err=%v", spans, err)
	}

	logRequest := validLogRequest(now)
	logs, err := server.parseLogs(logRequest)
	if err != nil || len(logs) != 1 || logs[0].TraceID == "" {
		t.Fatalf("valid log: logs=%v err=%v", logs, err)
	}

	traceRequest.ResourceSpans[0].ScopeSpans[0].Spans[0].Attributes = []*commonv1.KeyValue{{
		Key: "invalid", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_DoubleValue{DoubleValue: math.NaN()}},
	}}
	if _, err := server.parseSpans(traceRequest); err == nil {
		t.Fatal("expected non-finite attribute to fail")
	}

	logRequest.ResourceLogs[0].ScopeLogs[0].LogRecords[0].TimeUnixNano = uint64(now.Add(-4 * 24 * time.Hour).UnixNano())
	if _, err := server.parseLogs(logRequest); err == nil {
		t.Fatal("expected record older than log retention to fail")
	}
}

func TestTracePersistenceFailureIsInternal(t *testing.T) {
	server := NewServer(db.New(failingDB{}), 7*24*time.Hour, 3*24*time.Hour)
	_, err := server.Traces().Export(context.Background(), validTraceRequest(time.Now()))
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal, got %v", err)
	}
}

func validTraceRequest(now time.Time) *collectortracev1.ExportTraceServiceRequest {
	return &collectortracev1.ExportTraceServiceRequest{ResourceSpans: []*tracev1.ResourceSpans{{
		Resource: serviceResource(),
		ScopeSpans: []*tracev1.ScopeSpans{{Spans: []*tracev1.Span{{
			TraceId: validID(16), SpanId: validID(8), Name: "GET /checkout",
			StartTimeUnixNano: uint64(now.Add(-time.Millisecond).UnixNano()),
			EndTimeUnixNano:   uint64(now.UnixNano()),
			Status:            &tracev1.Status{Code: tracev1.Status_STATUS_CODE_OK},
		}}}},
	}}}
}

func validLogRequest(now time.Time) *collectorlogsv1.ExportLogsServiceRequest {
	return &collectorlogsv1.ExportLogsServiceRequest{ResourceLogs: []*logsv1.ResourceLogs{{
		Resource: serviceResource(),
		ScopeLogs: []*logsv1.ScopeLogs{{LogRecords: []*logsv1.LogRecord{{
			TimeUnixNano: uint64(now.UnixNano()), SeverityText: "INFO",
			Body:    &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "paid"}},
			TraceId: validID(16), SpanId: validID(8),
		}}}},
	}}}
}

func validID(size int) []byte {
	value := make([]byte, size)
	value[0] = 1
	return value
}

func serviceResource() *resourcev1.Resource {
	return &resourcev1.Resource{Attributes: []*commonv1.KeyValue{{
		Key: "service.name", Value: &commonv1.AnyValue{Value: &commonv1.AnyValue_StringValue{StringValue: "checkout"}},
	}}}
}
