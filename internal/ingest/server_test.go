package ingest

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kanshi-dev/core/internal/db"
	proto "github.com/kanshi-dev/core/proto"
)

// mockDBTX mocks the pgx DBTX interface for testing.
type mockDBTX struct {
	execCallCount int
	execResult    func(call int) (pgconn.CommandTag, error)
}

func (m *mockDBTX) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	m.execCallCount++
	if m.execResult != nil {
		return m.execResult(m.execCallCount)
	}
	return pgconn.CommandTag{}, nil
}

func (m *mockDBTX) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (m *mockDBTX) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return nil
}

func TestIngestBatch_HeartbeatFailureAfterInsert(t *testing.T) {
	// Arrange
	mockDB := &mockDBTX{
		execResult: func(call int) (pgconn.CommandTag, error) {
			if call == 1 {
				// First call: InsertMetricsBatch -> success
				return pgconn.CommandTag{}, nil
			}
			// Second call: UpsertAgentHeartbeat -> error
			return pgconn.CommandTag{}, errors.New("heartbeat failed")
		},
	}
	queries := db.New(mockDB)

	s := &Server{queries: queries}

	ctx := context.Background()
	req := &proto.Batch{
		AgentId: "test-agent",
		Points: []*proto.Point{
			{Name: "metric1", Value: 1.0, TimestampUnixNano: 1000},
			{Name: "metric2", Value: 2.0, TimestampUnixNano: 2000},
		},
	}

	// Act
	resp, err := s.IngestBatch(ctx, req)

	// Assert
	if err != nil {
		t.Fatalf("expected no error when heartbeat fails after successful insert, got: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected a non-nil response")
	}
	expectedAccepted := int64(len(req.Points))
	if resp.Accepted != expectedAccepted {
		t.Fatalf("expected Ack.Accepted = %d, got %d", expectedAccepted, resp.Accepted)
	}

	// Ensure exactly two Exec calls were made (insert and heartbeat)
	if mockDB.execCallCount != 2 {
		t.Fatalf("expected 2 Exec calls, got %d", mockDB.execCallCount)
	}
}

func TestIngestBatch_InsertFailure(t *testing.T) {
	// Arrange
	mockDB := &mockDBTX{
		execResult: func(call int) (pgconn.CommandTag, error) {
			// First call: InsertMetricsBatch -> error
			if call == 1 {
				return pgconn.CommandTag{}, errors.New("insert failed")
			}
			// This should not be called
			return pgconn.CommandTag{}, nil
		},
	}
	queries := db.New(mockDB)

	s := &Server{queries: queries}

	ctx := context.Background()
	req := &proto.Batch{
		AgentId: "test-agent",
		Points: []*proto.Point{
			{Name: "metric1", Value: 1.0, TimestampUnixNano: 1000},
		},
	}

	// Act
	resp, err := s.IngestBatch(ctx, req)

	// Assert
	if err == nil {
		t.Fatalf("expected error when insert fails, got nil")
	}
	if resp != nil {
		t.Fatalf("expected nil response on error, got %v", resp)
	}
	// Ensure only one Exec call was made (the insert) and heartbeat was not attempted
	if mockDB.execCallCount != 1 {
		t.Fatalf("expected 1 Exec call, got %d", mockDB.execCallCount)
	}
}