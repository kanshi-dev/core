package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kanshi-dev/core/internal/db"
	pb "github.com/kanshi-dev/core/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	if err := validateReport(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	profileTargets, err := json.Marshal(req.GetProfileTargets())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to encode profile targets")
	}
	err = s.queries.UpsertAgentReport(
		ctx,
		db.UpsertAgentReportParams{
			AgentID:        req.AgentId,
			Hostname:       req.Hostname,
			Os:             req.Os,
			Platform:       req.Platform,
			Arch:           req.Arch,
			CpuCores:       req.CpuCores,
			TotalMemory:    req.TotalMemory,
			DiskSize:       req.DiskSize,
			Version:        req.Version,
			ProfileTargets: profileTargets,
		},
	)
	if err != nil {
		return nil, err
	}

	return s.ack(ctx, req.AgentId, 1), nil
}

func (s *Server) IngestBatch(ctx context.Context, req *pb.Batch) (*pb.Ack, error) {
	if s.queries == nil {
		return nil, ErrNoDatabase
	}
	if err := validateBatch(req, time.Now()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	count := len(req.Points)

	if count == 0 {
		return s.ack(ctx, req.AgentId, 0), nil
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

		tags[i] = append([]string{}, p.Tags...)
	}

	encodedTags, err := json.Marshal(tags)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to encode metric tags")
	}
	err = s.queries.InsertMetricsBatch(ctx, db.InsertMetricsBatchParams{
		AgentIDS:   agentIDs,
		Names:      names,
		Values:     values,
		Timestamps: timestamps,
		Tags:       encodedTags,
	})
	if err != nil {
		log.Printf("failed to insert metrics batch: %v", err)
		return nil, status.Error(codes.Internal, "failed to insert metrics batch")
	}

	if err := s.queries.UpsertAgentHeartbeat(ctx, req.AgentId); err != nil {
		log.Printf("warning: failed to upsert heartbeat for agent %s: %v", req.AgentId, err)
	}

	return s.ack(ctx, req.AgentId, int64(count)), nil
}

func (s *Server) UploadProfile(ctx context.Context, req *pb.ProfileUpload) (*pb.Ack, error) {
	if s.queries == nil {
		return nil, ErrNoDatabase
	}
	if err := validateProfileUpload(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var err error
	if req.GetError() != "" {
		_, err = s.queries.FailProfileCapture(ctx, db.FailProfileCaptureParams{
			Error: pgtype.Text{String: req.GetError(), Valid: true}, ID: req.GetCaptureId(), AgentID: req.GetAgentId(),
		})
	} else {
		_, err = s.queries.CompleteProfileCapture(ctx, db.CompleteProfileCaptureParams{
			Filename:    pgtype.Text{String: req.GetFilename(), Valid: true},
			ContentType: pgtype.Text{String: req.GetContentType(), Valid: true},
			Artifact:    req.GetArtifact(), ID: req.GetCaptureId(), AgentID: req.GetAgentId(),
		})
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "active profile capture not found")
	}
	if err != nil {
		log.Printf("failed to store profile capture: %v", err)
		return nil, status.Error(codes.Internal, "failed to store profile capture")
	}
	return &pb.Ack{Accepted: 1}, nil
}

func (s *Server) ack(ctx context.Context, agentID string, accepted int64) *pb.Ack {
	ack := &pb.Ack{Accepted: accepted}
	capture, err := s.queries.ClaimProfileCapture(ctx, agentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ack
	}
	if err != nil {
		log.Printf("warning: failed to claim profile capture for agent %s: %v", agentID, err)
		return ack
	}
	ack.ProfileCommand = &pb.ProfileCommand{
		CaptureId:       capture.ID,
		TargetName:      capture.TargetName,
		ProfileType:     capture.ProfileType,
		DurationSeconds: int32(capture.DurationSeconds),
		ExpiresUnixNano: capture.ExpiresAt.Time.UnixNano(),
	}
	return ack
}
