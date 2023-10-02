package ingestcontroller

import (
	"context"

	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"
	"github.com/romashorodok/stream-platform/services/stream/pkg/service"
	"google.golang.org/grpc"
)

type IngestController interface {
	StartServer(ctx context.Context, in *ingestioncontrollerpb.StartServerRequest, opts ...grpc.CallOption) (*ingestioncontrollerpb.StartServerResponse, error)
	StopServer(ctx context.Context, in *ingestioncontrollerpb.StopServerRequest, opts ...grpc.CallOption) (*ingestioncontrollerpb.StopServerResponse, error)
}

type StandaloneIngestControllerStub struct {
	ingestioncontrollerpb.UnimplementedIngestControllerServiceServer

	config *service.StreamSystemConfig
}

func (ctrl *StandaloneIngestControllerStub) StartServer(ctx context.Context, in *ingestioncontrollerpb.StartServerRequest, _ ...grpc.CallOption) (*ingestioncontrollerpb.StartServerResponse, error) {
	return &ingestioncontrollerpb.StartServerResponse{
		Deployment: ctrl.config.Deployment,
		Namespace:  ctrl.config.Namespace,
	}, nil
}

func (*StandaloneIngestControllerStub) StopServer(ctx context.Context, in *ingestioncontrollerpb.StopServerRequest, _ ...grpc.CallOption) (*ingestioncontrollerpb.StopServerResponse, error) {
	return &ingestioncontrollerpb.StopServerResponse{}, nil
}

var _ IngestController = (*StandaloneIngestControllerStub)(nil)
var _ ingestioncontrollerpb.IngestControllerServiceClient = (*StandaloneIngestControllerStub)(nil)

type StandaloneIngestControllerStubParams struct {
	Config *service.StreamSystemConfig
}

func NewStandaloneIngestControllerStub(params StandaloneIngestControllerStubParams) *StandaloneIngestControllerStub {
	return &StandaloneIngestControllerStub{config: params.Config}
}
