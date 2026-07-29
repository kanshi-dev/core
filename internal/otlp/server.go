package otlp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
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

const (
	MaxBatchRecords  = 1000
	MaxAttributes    = 64
	MaxAttributeKey  = 128
	MaxAttributeJSON = 32 << 10
	MaxLogBody       = 16 << 10
)

type Server struct {
	queries        *db.Queries
	traceRetention time.Duration
	logRetention   time.Duration
	now            func() time.Time
}

type TraceServer struct {
	collectortracev1.UnimplementedTraceServiceServer
	*Server
}

type LogsServer struct {
	collectorlogsv1.UnimplementedLogsServiceServer
	*Server
}

func NewServer(queries *db.Queries, traceRetention, logRetention time.Duration) *Server {
	return &Server{
		queries:        queries,
		traceRetention: traceRetention,
		logRetention:   logRetention,
		now:            time.Now,
	}
}

func (s *Server) Traces() *TraceServer { return &TraceServer{Server: s} }
func (s *Server) Logs() *LogsServer    { return &LogsServer{Server: s} }

func (s *TraceServer) Export(ctx context.Context, req *collectortracev1.ExportTraceServiceRequest) (*collectortracev1.ExportTraceServiceResponse, error) {
	if s.queries == nil {
		return nil, status.Error(codes.Unavailable, "database unavailable")
	}
	spans, err := s.parseSpans(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	// ponytail: bounded sequential inserts are enough for 1000 records; use COPY if measured ingest throughput requires it.
	for _, span := range spans {
		if err := s.queries.InsertSpan(ctx, span); err != nil {
			return nil, status.Error(codes.Internal, "failed to persist traces")
		}
	}
	return &collectortracev1.ExportTraceServiceResponse{}, nil
}

func (s *LogsServer) Export(ctx context.Context, req *collectorlogsv1.ExportLogsServiceRequest) (*collectorlogsv1.ExportLogsServiceResponse, error) {
	if s.queries == nil {
		return nil, status.Error(codes.Unavailable, "database unavailable")
	}
	logs, err := s.parseLogs(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	for _, record := range logs {
		if err := s.queries.InsertLog(ctx, record); err != nil {
			return nil, status.Error(codes.Internal, "failed to persist logs")
		}
	}
	return &collectorlogsv1.ExportLogsServiceResponse{}, nil
}

func (s *Server) parseSpans(req *collectortracev1.ExportTraceServiceRequest) ([]db.InsertSpanParams, error) {
	var out []db.InsertSpanParams
	now := s.now()
	for _, resourceSpans := range req.GetResourceSpans() {
		serviceName, err := serviceName(resourceSpans.GetResource())
		if err != nil {
			return nil, err
		}
		for _, scopeSpans := range resourceSpans.GetScopeSpans() {
			for _, span := range scopeSpans.GetSpans() {
				if len(out) == MaxBatchRecords {
					return nil, fmt.Errorf("trace batch exceeds %d spans", MaxBatchRecords)
				}
				parsed, err := parseSpan(span, serviceName, now, s.traceRetention)
				if err != nil {
					return nil, err
				}
				out = append(out, parsed)
			}
		}
	}
	return out, nil
}

func (s *Server) parseLogs(req *collectorlogsv1.ExportLogsServiceRequest) ([]db.InsertLogParams, error) {
	var out []db.InsertLogParams
	now := s.now()
	for _, resourceLogs := range req.GetResourceLogs() {
		serviceName, err := serviceName(resourceLogs.GetResource())
		if err != nil {
			return nil, err
		}
		for _, scopeLogs := range resourceLogs.GetScopeLogs() {
			for _, record := range scopeLogs.GetLogRecords() {
				if len(out) == MaxBatchRecords {
					return nil, fmt.Errorf("log batch exceeds %d records", MaxBatchRecords)
				}
				parsed, err := parseLog(record, serviceName, now, s.logRetention)
				if err != nil {
					return nil, err
				}
				out = append(out, parsed)
			}
		}
	}
	return out, nil
}

func parseSpan(span *tracev1.Span, serviceName string, now time.Time, retention time.Duration) (db.InsertSpanParams, error) {
	if len(span.GetTraceId()) != 16 || allZero(span.GetTraceId()) ||
		len(span.GetSpanId()) != 8 || allZero(span.GetSpanId()) ||
		(len(span.GetParentSpanId()) != 0 && (len(span.GetParentSpanId()) != 8 || allZero(span.GetParentSpanId()))) {
		return db.InsertSpanParams{}, errors.New("span has invalid trace or span ID")
	}
	if span.GetName() == "" || len(span.GetName()) > 255 {
		return db.InsertSpanParams{}, errors.New("span operation must be 1 to 255 bytes")
	}
	if span.GetKind() < tracev1.Span_SPAN_KIND_UNSPECIFIED || span.GetKind() > tracev1.Span_SPAN_KIND_CONSUMER {
		return db.InsertSpanParams{}, errors.New("span kind is invalid")
	}
	if span.GetStatus().GetCode() < tracev1.Status_STATUS_CODE_UNSET || span.GetStatus().GetCode() > tracev1.Status_STATUS_CODE_ERROR {
		return db.InsertSpanParams{}, errors.New("span status is invalid")
	}
	start, err := timestamp(span.GetStartTimeUnixNano(), now, retention)
	if err != nil {
		return db.InsertSpanParams{}, fmt.Errorf("invalid span start time: %w", err)
	}
	end, err := timestamp(span.GetEndTimeUnixNano(), now, retention)
	if err != nil || end.Before(start) {
		return db.InsertSpanParams{}, errors.New("invalid span end time")
	}
	attributes, err := attributesJSON(span.GetAttributes())
	if err != nil {
		return db.InsertSpanParams{}, err
	}
	statusMessage := span.GetStatus().GetMessage()
	if len(statusMessage) > 1024 {
		return db.InsertSpanParams{}, errors.New("span status message exceeds 1024 bytes")
	}
	return db.InsertSpanParams{
		TraceID:       hex.EncodeToString(span.GetTraceId()),
		SpanID:        hex.EncodeToString(span.GetSpanId()),
		ParentSpanID:  hex.EncodeToString(span.GetParentSpanId()),
		ServiceName:   serviceName,
		Operation:     span.GetName(),
		SpanKind:      int16(span.GetKind()),
		StatusCode:    int16(span.GetStatus().GetCode()),
		StatusMessage: statusMessage,
		StartTime:     pgtype.Timestamptz{Time: start, Valid: true},
		EndTime:       pgtype.Timestamptz{Time: end, Valid: true},
		DurationMs:    float64(end.Sub(start)) / float64(time.Millisecond),
		Attributes:    attributes,
	}, nil
}

func parseLog(record *logsv1.LogRecord, serviceName string, now time.Time, retention time.Duration) (db.InsertLogParams, error) {
	ts, err := timestamp(record.GetTimeUnixNano(), now, retention)
	if err != nil {
		return db.InsertLogParams{}, fmt.Errorf("invalid log timestamp: %w", err)
	}
	if len(record.GetTraceId()) != 0 && (len(record.GetTraceId()) != 16 || allZero(record.GetTraceId())) {
		return db.InsertLogParams{}, errors.New("log has invalid trace ID")
	}
	if len(record.GetSpanId()) != 0 && (len(record.GetSpanId()) != 8 || allZero(record.GetSpanId())) {
		return db.InsertLogParams{}, errors.New("log has invalid span ID")
	}
	body, err := valueJSON(record.GetBody())
	if err != nil {
		return db.InsertLogParams{}, err
	}
	if len(body) > MaxLogBody {
		return db.InsertLogParams{}, fmt.Errorf("log body exceeds %d bytes", MaxLogBody)
	}
	attributes, err := attributesJSON(record.GetAttributes())
	if err != nil {
		return db.InsertLogParams{}, err
	}
	severity := record.GetSeverityText()
	if severity == "" {
		severity = record.GetSeverityNumber().String()
	}
	if len(severity) > 32 {
		return db.InsertLogParams{}, errors.New("log severity exceeds 32 bytes")
	}
	return db.InsertLogParams{
		Ts:          pgtype.Timestamptz{Time: ts, Valid: true},
		ServiceName: serviceName,
		Severity:    severity,
		Body:        body,
		TraceID:     hex.EncodeToString(record.GetTraceId()),
		SpanID:      hex.EncodeToString(record.GetSpanId()),
		Attributes:  attributes,
	}, nil
}

func allZero(value []byte) bool {
	for _, b := range value {
		if b != 0 {
			return false
		}
	}
	return true
}

func serviceName(resource *resourcev1.Resource) (string, error) {
	if _, err := attributesJSON(resource.GetAttributes()); err != nil {
		return "", fmt.Errorf("invalid resource attributes: %w", err)
	}
	for _, attribute := range resource.GetAttributes() {
		if attribute.GetKey() == "service.name" {
			name := attribute.GetValue().GetStringValue()
			if name != "" && len(name) <= 255 {
				return name, nil
			}
		}
	}
	return "", errors.New("resource requires service.name of 1 to 255 bytes")
}

func timestamp(nanos uint64, now time.Time, retention time.Duration) (time.Time, error) {
	if nanos == 0 || nanos > math.MaxInt64 {
		return time.Time{}, errors.New("timestamp is required")
	}
	value := time.Unix(0, int64(nanos))
	if value.After(now.Add(5 * time.Minute)) {
		return time.Time{}, errors.New("timestamp is too far in the future")
	}
	if value.Before(now.Add(-retention)) {
		return time.Time{}, errors.New("timestamp is older than retention")
	}
	return value, nil
}

func attributesJSON(attributes []*commonv1.KeyValue) ([]byte, error) {
	if len(attributes) > MaxAttributes {
		return nil, fmt.Errorf("attributes exceed %d entries", MaxAttributes)
	}
	values := make(map[string]any, len(attributes))
	for _, attribute := range attributes {
		if attribute.GetKey() == "" || len(attribute.GetKey()) > MaxAttributeKey {
			return nil, fmt.Errorf("attribute key must be 1 to %d bytes", MaxAttributeKey)
		}
		value, err := anyValue(attribute.GetValue())
		if err != nil {
			return nil, err
		}
		values[attribute.GetKey()] = value
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return nil, errors.New("invalid attributes")
	}
	if len(encoded) > MaxAttributeJSON {
		return nil, fmt.Errorf("attributes exceed %d bytes", MaxAttributeJSON)
	}
	return encoded, nil
}

func valueJSON(value *commonv1.AnyValue) (string, error) {
	decoded, err := anyValue(value)
	if err != nil {
		return "", err
	}
	if text, ok := decoded.(string); ok {
		return text, nil
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return "", errors.New("invalid log body")
	}
	return string(encoded), nil
}

func anyValue(value *commonv1.AnyValue) (any, error) {
	if value == nil {
		return "", nil
	}
	switch v := value.GetValue().(type) {
	case *commonv1.AnyValue_StringValue:
		if len(v.StringValue) > MaxLogBody {
			return nil, errors.New("attribute string is too large")
		}
		return v.StringValue, nil
	case *commonv1.AnyValue_BoolValue:
		return v.BoolValue, nil
	case *commonv1.AnyValue_IntValue:
		return v.IntValue, nil
	case *commonv1.AnyValue_DoubleValue:
		if math.IsNaN(v.DoubleValue) || math.IsInf(v.DoubleValue, 0) {
			return nil, errors.New("attribute number must be finite")
		}
		return v.DoubleValue, nil
	case *commonv1.AnyValue_BytesValue:
		if len(v.BytesValue) > MaxLogBody {
			return nil, errors.New("attribute bytes are too large")
		}
		return hex.EncodeToString(v.BytesValue), nil
	case *commonv1.AnyValue_ArrayValue:
		out := make([]any, 0, len(v.ArrayValue.GetValues()))
		for _, item := range v.ArrayValue.GetValues() {
			decoded, err := anyValue(item)
			if err != nil {
				return nil, err
			}
			out = append(out, decoded)
		}
		return out, nil
	case *commonv1.AnyValue_KvlistValue:
		out := make(map[string]any, len(v.KvlistValue.GetValues()))
		for _, item := range v.KvlistValue.GetValues() {
			decoded, err := anyValue(item.GetValue())
			if err != nil {
				return nil, err
			}
			out[item.GetKey()] = decoded
		}
		return out, nil
	default:
		return "", nil
	}
}
