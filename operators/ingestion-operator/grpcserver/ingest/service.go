package ingest

import (
	"context"

	"github.com/go-logr/logr"
	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"

	v1alpha1 "github.com/romashorodok/stream-platform/operators/ingestion-operator/api/romashorodok.github.io"

	"github.com/romashorodok/stream-platform/operators/ingestion-operator/grpcserver/container"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/resource/ingestresource"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/resource/istioresource"
	"go.uber.org/fx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IngestControllerService struct {
	ingestioncontrollerpb.UnimplementedIngestControllerServiceServer

	logger                logr.Logger
	k8s                   client.Client
	ingestResourceManager *ingestresource.IngestResourceManager
	istioResourceManager  *istioresource.IstioResourceManager
	ingestSystem          *ingestresource.IngestSystem
}

var _ ingestioncontrollerpb.IngestControllerServiceServer = (*IngestControllerService)(nil)

const (
	DEFAULT_GATEWAY_PORT     = 80
	DEFAULT_GATEWAY_APP_NAME = "gateway"
)

type StartIngestSystemParams struct {
	Context       context.Context
	AppName       string
	Namespace     string
	Template      *v1alpha1.IngestTemplate
	Username      string
	BroadcasterID string
}

func (s *IngestControllerService) StartServer(context context.Context, req *ingestioncontrollerpb.StartServerRequest) (*ingestioncontrollerpb.StartServerResponse, error) {
	log := container.WithLogr(context)

	ingestTemplates := container.WithIngestTemplates()

	ingestTemplate, err := ingestTemplates.Get(req.IngestTemplate)
	if err != nil {
		log.Error(err, "Unable find ingest template")
		return nil, status.Errorf(codes.NotFound, "not found ingest template")
	}

	_ = s.ingestSystem.StopIngestSystem(context, req.Deployment, req.Namespace)

	if err := s.ingestSystem.StartIngestSystem(ingestresource.StartIngestSystemParams{
		Context:       context,
		AppName:       req.Deployment,
		Namespace:     req.Namespace,
		Template:      ingestTemplate,
		Username:      req.Meta.Username,
		BroadcasterID: req.Meta.BroadcasterId,
	}); err != nil {
		s.logger.Error(err, "unable stop ingest system")
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &ingestioncontrollerpb.StartServerResponse{
		Deployment: req.Deployment,
		Namespace:  req.Namespace,
	}, nil
}

func (s *IngestControllerService) StopServer(context context.Context, req *ingestioncontrollerpb.StopServerRequest) (*ingestioncontrollerpb.StopServerResponse, error) {

	if err := s.ingestSystem.StopIngestSystem(context, req.Deployment, req.Namespace); err != nil {
		s.logger.Error(err, "unable stop ingest system")

		return nil, status.Error(codes.Internal, err.Error())
	}

	return &ingestioncontrollerpb.StopServerResponse{}, nil
}

type IngestControllerServiceParams struct {
	fx.In
	Logger                logr.Logger
	K8s                   client.Client
	IngestResourceManager *ingestresource.IngestResourceManager
	IstioResourceManager  *istioresource.IstioResourceManager
	IngesetSystem         *ingestresource.IngestSystem
}

func NewIngestControllerService(params IngestControllerServiceParams) *IngestControllerService {
	return &IngestControllerService{
		k8s:                   params.K8s,
		ingestResourceManager: params.IngestResourceManager,
		istioResourceManager:  params.IstioResourceManager,
		logger:                params.Logger,
		ingestSystem:          params.IngesetSystem,
	}
}
