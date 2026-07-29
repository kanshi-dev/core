package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kanshi-dev/core/internal/db"
)

type TelemetryService struct {
	queries *db.Queries
}

type ServiceSummary struct {
	ServiceName   string  `json:"serviceName"`
	RequestCount  int64   `json:"requestCount"`
	ErrorCount    int64   `json:"errorCount"`
	ErrorRate     float64 `json:"errorRate"`
	AvgDurationMs float64 `json:"avgDurationMs"`
	P95DurationMs float64 `json:"p95DurationMs"`
}

type TraceSummary struct {
	TraceID       string    `json:"traceId"`
	ServiceName   string    `json:"serviceName"`
	RootOperation string    `json:"rootOperation"`
	StartTime     time.Time `json:"startTime"`
	EndTime       time.Time `json:"endTime"`
	DurationMs    float64   `json:"durationMs"`
	StatusCode    int16     `json:"statusCode"`
	SpanCount     int64     `json:"spanCount"`
}

type Span struct {
	TraceID       string          `json:"traceId"`
	SpanID        string          `json:"spanId"`
	ParentSpanID  string          `json:"parentSpanId"`
	ServiceName   string          `json:"serviceName"`
	Operation     string          `json:"operation"`
	SpanKind      int16           `json:"spanKind"`
	StatusCode    int16           `json:"statusCode"`
	StatusMessage string          `json:"statusMessage"`
	StartTime     time.Time       `json:"startTime"`
	EndTime       time.Time       `json:"endTime"`
	DurationMs    float64         `json:"durationMs"`
	Attributes    json.RawMessage `json:"attributes"`
}

type LogRecord struct {
	Timestamp   time.Time       `json:"timestamp"`
	ServiceName string          `json:"serviceName"`
	Severity    string          `json:"severity"`
	Body        string          `json:"body"`
	TraceID     string          `json:"traceId"`
	SpanID      string          `json:"spanId"`
	Attributes  json.RawMessage `json:"attributes"`
}

type TraceSearch struct {
	From, To    time.Time
	ServiceName string
	StatusCode  int16
	MinDuration float64
	TraceID     string
	Limit       int32
}

type LogSearch struct {
	From, To    time.Time
	ServiceName string
	TraceID     string
	SpanID      string
	Limit       int32
}

func NewTelemetryService(q *db.Queries) *TelemetryService {
	return &TelemetryService{queries: q}
}

func (s *TelemetryService) ListServices(ctx context.Context, from, to time.Time, limit int32) ([]ServiceSummary, error) {
	if s.queries == nil {
		return nil, ErrNoDatabase
	}
	rows, err := s.queries.ListServiceSummaries(ctx, db.ListServiceSummariesParams{
		FromTime: pgtime(from), ToTime: pgtime(to), ResultLimit: limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ServiceSummary, 0, len(rows))
	for _, row := range rows {
		errorRate := 0.0
		if row.RequestCount > 0 {
			errorRate = float64(row.ErrorCount) / float64(row.RequestCount)
		}
		out = append(out, ServiceSummary{
			ServiceName: row.ServiceName, RequestCount: row.RequestCount, ErrorCount: row.ErrorCount,
			ErrorRate: errorRate, AvgDurationMs: row.AvgDurationMs, P95DurationMs: row.P95DurationMs,
		})
	}
	return out, nil
}

func (s *TelemetryService) SearchTraces(ctx context.Context, search TraceSearch) ([]TraceSummary, error) {
	if s.queries == nil {
		return nil, ErrNoDatabase
	}
	rows, err := s.queries.SearchTraces(ctx, db.SearchTracesParams{
		FromTime: pgtime(search.From), ToTime: pgtime(search.To), ServiceName: search.ServiceName,
		StatusCode: search.StatusCode, MinDurationMs: search.MinDuration,
		TraceID: search.TraceID, ResultLimit: search.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]TraceSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, TraceSummary{
			TraceID: row.TraceID, ServiceName: row.ServiceName, RootOperation: row.RootOperation,
			StartTime: row.StartTime.Time, EndTime: row.EndTime.Time, DurationMs: row.DurationMs,
			StatusCode: row.StatusCode, SpanCount: row.SpanCount,
		})
	}
	return out, nil
}

func (s *TelemetryService) GetTrace(ctx context.Context, traceID string) ([]Span, error) {
	if s.queries == nil {
		return nil, ErrNoDatabase
	}
	rows, err := s.queries.GetTraceSpans(ctx, traceID)
	if err != nil {
		return nil, err
	}
	out := make([]Span, 0, len(rows))
	for _, row := range rows {
		out = append(out, Span{
			TraceID: row.TraceID, SpanID: row.SpanID, ParentSpanID: row.ParentSpanID,
			ServiceName: row.ServiceName, Operation: row.Operation, StatusCode: row.StatusCode,
			SpanKind:      row.SpanKind,
			StatusMessage: row.StatusMessage, StartTime: row.StartTime.Time, EndTime: row.EndTime.Time,
			DurationMs: row.DurationMs, Attributes: row.Attributes,
		})
	}
	return out, nil
}

func (s *TelemetryService) SearchLogs(ctx context.Context, search LogSearch) ([]LogRecord, error) {
	if s.queries == nil {
		return nil, ErrNoDatabase
	}
	rows, err := s.queries.SearchLogs(ctx, db.SearchLogsParams{
		FromTime: pgtime(search.From), ToTime: pgtime(search.To), ServiceName: search.ServiceName,
		TraceID: search.TraceID, SpanID: search.SpanID, ResultLimit: search.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]LogRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, LogRecord{
			Timestamp: row.Ts.Time, ServiceName: row.ServiceName, Severity: row.Severity,
			Body: row.Body, TraceID: row.TraceID, SpanID: row.SpanID, Attributes: row.Attributes,
		})
	}
	return out, nil
}

func pgtime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}
