package main

import (
	"context"
	"log"
	"net"
	"os"

	"github.com/kanshi-dev/core/internal/api"
	"github.com/kanshi-dev/core/internal/db"
	"github.com/kanshi-dev/core/internal/ingest"
	"github.com/kanshi-dev/core/internal/service"
	pb "github.com/kanshi-dev/core/proto"
	"google.golang.org/grpc"
)

func main() {
	apiKey := os.Getenv("KANSHI_API_KEY")
	if apiKey == "" {
		log.Fatal("configuration error: KANSHI_API_KEY is required")
	}

	//Init Database
	ctx := context.Background()
	pool, err := db.NewPool(ctx)

	if err != nil {
		log.Printf("Warning: failed to connect to db: %v. Continuing without DB.", err)
	} else {
		defer pool.Close()
	}

	var queries *db.Queries
	var ping func(context.Context) error
	if pool != nil {
		queries = db.New(pool)
		ping = pool.Ping
	}

	// Init GRPC
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(ingest.APIKeyAuth(apiKey)))
	pb.RegisterIngestServiceServer(grpcServer, ingest.NewServer(queries))

	go func() {
		log.Println("kanshi-core listening on :50051")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	//Setup Services
	agentService := service.NewAgentsService(queries)
	metricsService := service.NewMetricsService(queries)

	// Init Api
	apiServer := api.NewServer(agentService, metricsService, ping)

	if err := apiServer.App.Listen(":8080"); err != nil {
		log.Fatal(err)
	}

}
