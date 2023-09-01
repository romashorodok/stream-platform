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
	"go.uber.org/fx"
	uberzap "go.uber.org/zap"
	"google.golang.org/grpc"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func unaryInterceptorLogrFromZap() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		zapLogger := ctxzap.Extract(ctx)
		logrLogger := zapr.NewLogger(zapLogger)

		newCtx := context.WithValue(ctx, container.CONTEXT_LOGR, &logrLogger)

		return handler(newCtx, req)
	}
}

type LoggerInterceptorParams struct {
	fx.In

	Logger *uberzap.Logger
}

func NewLoggerInterceptor(params LoggerInterceptorParams) grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(
		grpc_middleware.ChainUnaryServer(
			grpc_zap.UnaryServerInterceptor(params.Logger),
			unaryInterceptorLogrFromZap(),
		),
	)
}

type IngestionControllerServiceParams struct {
	fx.In

	K8s                     client.Client
	Server                  *grpc.Server
	IngestControllerService *ingest.IngestControllerService
}

func RegisterIngestionControllerService(params IngestionControllerServiceParams) {
	ingestioncontrollerpb.RegisterIngestControllerServiceServer(
		params.Server,
		params.IngestControllerService,
	)
}
