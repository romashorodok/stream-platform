package ingest

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"

	v1alpha1 "github.com/romashorodok/stream-platform/operators/ingestion-operator/api/romashorodok.github.io"

	"github.com/romashorodok/stream-platform/operators/ingestion-operator/grpcserver/container"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/resource/ingestresource"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/resource/istioresource"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
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
}

var _ ingestioncontrollerpb.IngestControllerServiceServer = (*IngestControllerService)(nil)

const (
	DEFAULT_GATEWAY_PORT     = 80
	DEFAULT_GATEWAY_APP_NAME = "gateway"
)

func (s *IngestControllerService) StartIngestSystem(context context.Context, appName string, namespace string, template *v1alpha1.IngestTemplate) error {
	ingressGatewayAppName := fmt.Sprintf("%s-gateway", appName)
	virtualServiceAppName := fmt.Sprintf("%s-virtual-service", appName)
	ingestHostName := fmt.Sprintf("%s.%s", appName, "localhost")
	ingestServiceName := appName
	owner := appName

	// if err := s.k8s.Create(context,
	// 	s.ingestResourceManager.IngestNamespace(ingestresource.IngestNamespaceParams{
	// 		Namespace: appName,
	// 		Owner:     owner,
	// 	}),
	// ); err != nil {
	// 	return fmt.Errorf("unable deploy ingest system namespace. Error: %s", err)
	// }
	// namespace = appName

	g := new(errgroup.Group)

	g.Go(func() error {
		ingest := s.ingestResourceManager.IngestDeploymentByTemplate(ingestresource.IngestDeploymentByTemplateParams{
			Namespace: namespace,
			AppName:   appName,
			Template:  template,
			Replicas:  1,
			Owner:     owner,
		})

		return s.k8s.Create(context, ingest)
	})

	g.Go(func() error {
		service := s.ingestResourceManager.IngestHeadlessService(ingestresource.IngestHeadlessServiceParams{
			Template:          template,
			IngestServiceName: ingestServiceName,
			AppName:           appName,
			Namespace:         namespace,
			Owner:             owner,
		})

		return s.k8s.Create(context, service)
	})

	g.Go(func() error {
		gateway := s.istioResourceManager.IstioGateway(istioresource.IstioGatewayParams{
			Number:                 DEFAULT_GATEWAY_PORT,
			Protocol:               "HTTP",
			IngressGatewaySelector: DEFAULT_GATEWAY_APP_NAME,
			IngressGatewayAppName:  ingressGatewayAppName,
			IngestHostName:         ingestHostName,
			Namespace:              namespace,
			Owner:                  owner,
		})

		return s.k8s.Create(context, gateway)
	})

	g.Go(func() error {
		virtualService := s.istioResourceManager.IstioVirtualService(istioresource.IstioVirtualServiceParams{
			IngestHostName: ingestHostName,
			// Istio has build it service discovery. By that name it will search dns of the service
			IngestServiceNameHost: ingestServiceName,
			IngestPorts:           template.Spec.Ports,
			VirtualServiceAppName: virtualServiceAppName,
			IngressGatewayAppName: ingressGatewayAppName,
			Namespace:             namespace,
			Owner:                 owner,
		})

		return s.k8s.Create(context, virtualService)
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("unable deploy ingest system. Error: %s", err)
	}

	return nil
}

func (s *IngestControllerService) StopIngestSystem(context context.Context, appName string, namespace string) error {
	ingressGatewayAppName := fmt.Sprintf("%s-gateway", appName)
	virtualServiceAppName := fmt.Sprintf("%s-virtual-service", appName)
	ingestServiceName := appName
	owner := appName

	// NOTE: If delete by namespace i need wait until namespace will terminated

	g := new(errgroup.Group)

	g.Go(func() error {
		ingest, err := s.ingestResourceManager.GetIngestByAppName(ingestresource.GetIngestByAppNameParams{
			Context:   context,
			Namespace: namespace,
			AppName:   appName,
		})
		if err != nil {
			return fmt.Errorf("unable find ingest deployment for %s/%s. Error: %s", namespace, appName, err)
		}

		return s.k8s.Delete(context, ingest)
	})

	g.Go(func() error {
		service, err := s.ingestResourceManager.GetIngestServiceByAppName(ingestresource.GetIngestServiceByAppName{
			Context:           context,
			Namespace:         namespace,
			IngestServiceName: ingestServiceName,
		})
		if err != nil {
			return fmt.Errorf("unable find ingest service for %s/%s. Error: %s", namespace, ingestServiceName, err)
		}

		return s.k8s.Delete(context, service)
	})

	g.Go(func() error {
		gateway, err := s.istioResourceManager.GetIstioGatewayByAppName(istioresource.GetIstioGatewayByAppNameParams{
			Context:              context,
			IngresGatewayAppName: ingressGatewayAppName,
			Namespace:            namespace,
			Owner:                owner,
		})
		if err != nil {
			return fmt.Errorf("unable find gateway for %s/%s. Error: %s", namespace, appName, err)
		}

		return s.k8s.Delete(context, gateway)
	})

	g.Go(func() error {
		virtualService, err := s.istioResourceManager.GetIstioVirtualServiceByAppName(istioresource.GetIstioVirtualServiceByAppNameParams{
			Context:               context,
			VirtualServiceAppName: virtualServiceAppName,
			Namespace:             namespace,
			Owner:                 owner,
		})
		if err != nil {
			return fmt.Errorf("unable find virtual service for %s/%s. Error: %s", namespace, appName, err)
		}

		return s.k8s.Delete(context, virtualService)
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("unable stop ingest system. Error: %s", err)
	}

	return nil
}

func (s *IngestControllerService) StartServer(context context.Context, req *ingestioncontrollerpb.StartServerRequest) (*ingestioncontrollerpb.StartServerResponse, error) {
	log := container.WithLogr(context)

	ingestTemplates := container.WithIngestTemplates()

	ingestTemplate, err := ingestTemplates.Get(req.IngestTemplate)

	if err != nil {
		log.Error(err, "Unable find ingest template")
		return nil, status.Errorf(codes.NotFound, "not found ingest template")
	}

	_ = s.StopIngestSystem(context, req.Deployment, req.Namespace)

	if err := s.StartIngestSystem(context, req.Deployment, req.Namespace, ingestTemplate); err != nil {
		s.logger.Error(err, "unable stop ingest system")

		return nil, status.Error(codes.Internal, err.Error())
	}

	return &ingestioncontrollerpb.StartServerResponse{
		Deployment: req.Deployment,
		Namespace:  req.Namespace,
	}, nil
}

func (s *IngestControllerService) StopServer(context context.Context, req *ingestioncontrollerpb.StopServerRequest) (*ingestioncontrollerpb.StopServerResponse, error) {

	if err := s.StopIngestSystem(context, req.Deployment, req.Namespace); err != nil {
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
}

func NewIngestControllerService(params IngestControllerServiceParams) *IngestControllerService {
	return &IngestControllerService{
		k8s:                   params.K8s,
		ingestResourceManager: params.IngestResourceManager,
		istioResourceManager:  params.IstioResourceManager,
		logger:                params.Logger,
	}
}
