package ingest

import (
	"context"
	"testing"

	pb "github.com/kanshi-dev/core/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAPIKeyAuth(t *testing.T) {
	interceptor := APIKeyAuth("secret")
	methods := []string{
		pb.IngestService_ReportAgent_FullMethodName,
		pb.IngestService_IngestBatch_FullMethodName,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			called := false
			handler := func(context.Context, any) (any, error) {
				called = true
				return "ok", nil
			}

			for name, values := range map[string][]string{
				"missing":  nil,
				"invalid":  {"wrong"},
				"multiple": {"secret", "secret"},
			} {
				t.Run(name, func(t *testing.T) {
					ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{"x-api-key": values})
					response, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: method}, handler)
					if response != nil || status.Code(err) != codes.Unauthenticated || called {
						t.Fatalf("got response=%v code=%s called=%v", response, status.Code(err), called)
					}
				})
			}

			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-api-key", "secret"))
			response, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: method}, handler)
			if response != "ok" || err != nil || !called {
				t.Fatalf("got response=%v err=%v called=%v", response, err, called)
			}
		})
	}
}
