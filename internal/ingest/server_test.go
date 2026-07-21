package ingest

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kanshi-dev/core/internal/db"
	pb "github.com/kanshi-dev/core/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type failingDB struct {
	calls  int
	failAt int
}

func (d *failingDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	d.calls++
	if d.calls == d.failAt {
		return pgconn.CommandTag{}, errors.New("database unavailable")
	}
	return pgconn.CommandTag{}, nil
}

func (*failingDB) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (*failingDB) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }

func TestIngestBatchErrorContract(t *testing.T) {
	req := &pb.Batch{AgentId: "agent", Points: []*pb.Point{{Name: "cpu"}}}

	t.Run("insert failure is retryable", func(t *testing.T) {
		store := &failingDB{failAt: 1}
		ack, err := NewServer(db.New(store)).IngestBatch(context.Background(), req)
		if ack != nil || status.Code(err) != codes.Internal || store.calls != 1 {
			t.Fatalf("got ack=%v code=%s calls=%d", ack, status.Code(err), store.calls)
		}
	})

	t.Run("heartbeat failure is acknowledged", func(t *testing.T) {
		store := &failingDB{failAt: 2}
		ack, err := NewServer(db.New(store)).IngestBatch(context.Background(), req)
		if err != nil || ack.GetAccepted() != 1 || store.calls != 2 {
			t.Fatalf("got ack=%v err=%v calls=%d", ack, err, store.calls)
		}
	})
}
