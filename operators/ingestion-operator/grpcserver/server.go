package grpcserver

import (
	"context"

	"github.com/go-logr/zapr"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/grpcserver/container"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/grpcserver/ingest"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UnaryInterceptorLogrFromZap() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		zapLogger := ctxzap.Extract(ctx)
		logrLogger := zapr.NewLogger(zapLogger)

		newCtx := context.WithValue(ctx, container.CONTEXT_LOGR, &logrLogger)

		return handler(newCtx, req)
	}
}

func NewServer(client client.Client, log *zap.Logger) *grpc.Server {
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				grpc_zap.UnaryServerInterceptor(log),
				UnaryInterceptorLogrFromZap(),
			),
		),
	)

	ingestionControllerService := ingest.NewIngestControllerService(client)
	ingestioncontrollerpb.RegisterIngestControllerServiceServer(server, ingestionControllerService)

	return server
}
