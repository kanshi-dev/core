package ingest

import (
	"context"
	"log"

	pb "github.com/kanshi-dev/core/proto"
)

type Server struct {
	pb.UnimplementedIngestServiceServer
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) IngestBatch(ctx context.Context, req *pb.Batch) (*pb.Ack, error) {
	log.Printf("received batch from %s: %d point", req.AgentId, len(req.Points))

	return &pb.Ack{
		Accepted: int64(len(req.Points)),
	}, nil
}
