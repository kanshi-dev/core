package main

import (
	"context"
	"log"
	"net"

	"github.com/kanshi-dev/core/internal/api"
	"github.com/kanshi-dev/core/internal/db"
	"github.com/kanshi-dev/core/internal/ingest"
	pb "github.com/kanshi-dev/core/proto"
	"google.golang.org/grpc"
)

func main() {

	//Init Database
	ctx := context.Background()
	pool, err := db.NewPool(ctx)

	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()
	queries := db.New(pool)

	// Init GRPC
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterIngestServiceServer(grpcServer, ingest.NewServer(queries))

	go func() {
		log.Println("kanshi-core listening on :50051")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Init Api
	apiServer := api.NewServer(queries)

	if err := apiServer.App.Listen(":8080"); err != nil {
		log.Fatal(err)
	}

}
