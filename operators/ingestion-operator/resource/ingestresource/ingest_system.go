package ingestresource

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/nats-io/nats.go"
	subjectpb "github.com/romashorodok/stream-platform/gen/golang/subject/v1alpha"
	v1alpha1 "github.com/romashorodok/stream-platform/operators/ingestion-operator/api/romashorodok.github.io"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/resource/istioresource"
	"github.com/romashorodok/stream-platform/pkg/subject"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IngestSystem struct {
	k8s client.Client

	ingestResourceManager *IngestResourceManager
	istioResourceManager  *istioresource.IstioResourceManager
	js                    nats.JetStreamContext
	log                   logr.Logger
}

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

func (s *IngestSystem) StartIngestSystem(params StartIngestSystemParams) error {
	ingressGatewayAppName := fmt.Sprintf("%s-gateway", params.AppName)
	virtualServiceAppName := fmt.Sprintf("%s-virtual-service", params.AppName)
	ingestHostName := fmt.Sprintf("%s.%s", params.AppName, "localhost")
	ingestServiceName := params.AppName
	owner := params.AppName

	// TODO: How to deal with namespace
	// NOTE: When I delete a namespace, the namespace changes its state to 'terminating,' and I can't create it again until the termination is complete.
	// Create it with prefix like `username-uuid.new()`

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
		ingest := s.ingestResourceManager.IngestDeploymentByTemplate(IngestDeploymentByTemplateParams{
			Namespace:     params.Namespace,
			AppName:       params.AppName,
			Template:      params.Template,
			Replicas:      1,
			Owner:         owner,
			BroadcasterID: params.BroadcasterID,
			Username:      params.Username,
		})

		return s.k8s.Create(params.Context, ingest)
	})

	g.Go(func() error {
		service := s.ingestResourceManager.IngestHeadlessService(IngestHeadlessServiceParams{
			Template:          params.Template,
			IngestServiceName: ingestServiceName,
			AppName:           params.AppName,
			Namespace:         params.Namespace,
			Owner:             owner,
		})

		return s.k8s.Create(params.Context, service)
	})

	g.Go(func() error {
		gateway := s.istioResourceManager.IstioGateway(istioresource.IstioGatewayParams{
			Number:                 DEFAULT_GATEWAY_PORT,
			Protocol:               "HTTP",
			IngressGatewaySelector: DEFAULT_GATEWAY_APP_NAME,
			IngressGatewayAppName:  ingressGatewayAppName,
			IngestHostName:         ingestHostName,
			Namespace:              params.Namespace,
			Owner:                  owner,
		})

		return s.k8s.Create(params.Context, gateway)
	})

	g.Go(func() error {
		virtualService := s.istioResourceManager.IstioVirtualService(istioresource.IstioVirtualServiceParams{
			IngestHostName: ingestHostName,
			// Istio has build it service discovery. By that name it will search dns of the service
			IngestServiceNameHost: ingestServiceName,
			IngestPorts:           params.Template.Spec.Ports,
			VirtualServiceAppName: virtualServiceAppName,
			IngressGatewayAppName: ingressGatewayAppName,
			Namespace:             params.Namespace,
			Owner:                 owner,
		})

		return s.k8s.Create(params.Context, virtualService)
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("unable deploy ingest system. Error: %s", err)
	}

	return nil
}

func (s *IngestSystem) StopIngestSystem(context context.Context, appName string, namespace string) error {
	ingressGatewayAppName := fmt.Sprintf("%s-gateway", appName)
	virtualServiceAppName := fmt.Sprintf("%s-virtual-service", appName)
	ingestServiceName := appName
	owner := appName

	// NOTE: If delete by namespace i need wait until namespace will terminated

	g := new(errgroup.Group)
	var broadcasterID string

	g.Go(func() error {
		ingest, err := s.ingestResourceManager.GetIngestByAppName(GetIngestByAppNameParams{
			Context:   context,
			Namespace: namespace,
			AppName:   appName,
		})
		if err != nil {
			return fmt.Errorf("unable find ingest deployment for %s/%s. Error: %s", namespace, appName, err)
		}

		broadcasterID = ingest.Labels[BROADCASTER_ID]

		return s.k8s.Delete(context, ingest)
	})

	g.Go(func() error {
		service, err := s.ingestResourceManager.GetIngestServiceByAppName(GetIngestServiceByAppName{
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

	if broadcasterID == "" {
		return fmt.Errorf("not found broadcasterID for %s/%s", namespace, appName)
	}

	_, err := s.sendDestroyNotification(broadcasterID, appName)
	if err != nil {
		s.log.Error(err, "Unable send destroy ingest system", "appName", appName, "broadcasterID", broadcasterID)
	}

	return nil
}

func (s *IngestSystem) sendDestroyNotification(broadcasterID, appName string) (*nats.PubAck, error) {
	return subject.JsPublishProtobufWithID(s.js, subject.NewIngestDestroyed(broadcasterID), broadcasterID, &subject.IngestDestroyed{
		Destroyed: true,
		Meta: &subjectpb.BroadcasterMeta{
			BroadcasterId: broadcasterID,
			Username:      appName,
		},
	})
}

type IngestSystemParams struct {
	fx.In

	K8s                   client.Client
	IngestResourceManager *IngestResourceManager
	IstioResourceManager  *istioresource.IstioResourceManager
	JS                    nats.JetStreamContext
	Log                   logr.Logger
}

func NewIngestSystem(params IngestSystemParams) *IngestSystem {
	return &IngestSystem{
		k8s:                   params.K8s,
		ingestResourceManager: params.IngestResourceManager,
		istioResourceManager:  params.IstioResourceManager,
		js:                    params.JS,
	}
}
