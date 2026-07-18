package ingest

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kanshi-dev/core/internal/db"
	pb "github.com/kanshi-dev/core/proto"
)

var ErrNoDatabase = errors.New("database connection not established")

type Server struct {
	pb.UnimplementedIngestServiceServer
	queries *db.Queries
}

func NewServer(queries *db.Queries) *Server {
	return &Server{
		queries: queries,
	}
}

func (s *Server) ReportAgent(
	ctx context.Context,
	req *pb.AgentReport,
) (*pb.Ack, error) {

	if s.queries == nil {
		return nil, ErrNoDatabase
	}

	err := s.queries.UpsertAgentReport(
		ctx,
		db.UpsertAgentReportParams{
			AgentID:     req.AgentId,
			Hostname:    req.Hostname,
			Os:          req.Os,
			Platform:    req.Platform,
			Arch:        req.Arch,
			CpuCores:    req.CpuCores,
			TotalMemory: req.TotalMemory,
			DiskSize:    req.DiskSize,
			Version:     req.Version,
		},
	)
	if err != nil {
		return nil, err
	}

	return &pb.Ack{Accepted: 1}, nil
}

func (s *Server) IngestBatch(ctx context.Context, req *pb.Batch) (*pb.Ack, error) {
	if s.queries == nil {
		return nil, ErrNoDatabase
	}

	count := len(req.Points)

	if count == 0 {
		return &pb.Ack{Accepted: 0}, nil
	}

	agentIDs := make([]string, count)
	names := make([]string, count)
	values := make([]float64, count)
	timestamps := make([]pgtype.Timestamptz, count)
	tags := make([][]string, count)

	for i, p := range req.Points {
		agentIDs[i] = req.AgentId
		names[i] = p.Name
		values[i] = p.Value

		timestamps[i] = pgtype.Timestamptz{
			Time:  time.Unix(0, p.TimestampUnixNano),
			Valid: true,
		}

		tags[i] = p.Tags
	}

	err := s.queries.InsertMetricsBatch(ctx, db.InsertMetricsBatchParams{
		AgentIDS:   agentIDs,
		Names:      names,
		Values:     values,
		Timestamps: timestamps,
		Tags:       tags,
	})
	if err != nil {
		log.Printf("failed to insert metrics batch: %v", err)
		return nil, err
	}

	if err := s.queries.UpsertAgentHeartbeat(ctx, req.AgentId); err != nil {
		log.Printf("warning: failed to upsert heartbeat for agent %s: %v", req.AgentId, err)
		return &pb.Ack{Accepted: int64(count)}, nil
	}

	return &pb.Ack{Accepted: int64(count)}, nil
}
