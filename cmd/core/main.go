package main

import (
	"context"
	"log"
	"net"

	"github.com/kanshi-dev/core/internal/db"
	"github.com/kanshi-dev/core/internal/ingest"
	pb "github.com/kanshi-dev/core/proto"
	"google.golang.org/grpc"
)

func main() {

	ctx := context.Background()

	pool, err := db.NewPool(ctx)

	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}

	defer pool.Close()

	quueries := db.New(pool)

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterIngestServiceServer(grpcServer, ingest.NewServer(
		quueries,
	))

	log.Println("kanshi-core listening on :50051")

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
