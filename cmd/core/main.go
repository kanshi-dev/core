package main

import (
	"context"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/kanshi-dev/core/internal/alert"
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
	dashboardKey := os.Getenv("KANSHI_DASHBOARD_KEY")
	if dashboardKey == "" {
		log.Fatal("configuration error: KANSHI_DASHBOARD_KEY is required")
	}

	//Init Database
	ctx := context.Background()
	pool, err := db.NewPool(ctx)

	if err != nil {
		log.Printf("Warning: failed to connect to db: %v. Continuing without DB.", err)
	} else {
		defer pool.Close()
		if err := db.Migrate(ctx, pool); err != nil {
			log.Fatalf("failed to apply database migrations: %v", err)
		}
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
	alertService := service.NewAlertsService(queries)

	// Start alert evaluation and webhook delivery when a database is available.
	if queries != nil {
		webhooks := alert.NewDispatcher(parseWebhookURLs(os.Getenv("KANSHI_WEBHOOK_URLS")), os.Getenv("KANSHI_WEBHOOK_SECRET"), queries)
		evaluator := alert.NewEvaluator(queries, webhooks, parseAlertInterval(os.Getenv("KANSHI_ALERT_INTERVAL")))
		go evaluator.Run(ctx)
	}

	// Init Api
	apiServer := api.NewServer(agentService, metricsService, alertService, ping, dashboardKey, os.Getenv("KANSHI_ALLOWED_ORIGINS"))

	if err := apiServer.App.Listen(":8080"); err != nil {
		log.Fatal(err)
	}

}

func parseWebhookURLs(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	urls := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			urls = append(urls, p)
		}
	}
	return urls
}

func parseAlertInterval(raw string) time.Duration {
	if raw == "" {
		return 30 * time.Second
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		log.Printf("invalid KANSHI_ALERT_INTERVAL %q, using 30s", raw)
		return 30 * time.Second
	}
	return d
}
