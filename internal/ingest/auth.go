package ingest

import (
	"context"
	"crypto/subtle"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func APIKeyAuth(apiKey string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		values := metadata.ValueFromIncomingContext(ctx, "x-api-key")
		if len(values) != 1 || subtle.ConstantTimeCompare([]byte(values[0]), []byte(apiKey)) != 1 {
			return nil, status.Error(codes.Unauthenticated, "invalid API key")
		}
		return handler(ctx, req)
	}
}
