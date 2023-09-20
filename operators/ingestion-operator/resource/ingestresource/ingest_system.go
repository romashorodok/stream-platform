package ingestresource

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/go-logr/logr"
	"github.com/nats-io/nats.go"
	subjectpb "github.com/romashorodok/stream-platform/gen/golang/subject/v1alpha"
	v1alpha1 "github.com/romashorodok/stream-platform/operators/ingestion-operator/api/romashorodok.github.io"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/pkg/portrange"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/resource/istioresource"
	"github.com/romashorodok/stream-platform/pkg/subject"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	FullNodePortAssignedError = errors.New("Unable find empty node port to assign the ingest")
)

type IngestSystem struct {
	k8s client.Client

	ingestResourceManager *IngestResourceManager
	istioResourceManager  *istioresource.IstioResourceManager
	js                    nats.JetStreamContext
	log                   logr.Logger
	nodePortRange         *portrange.PortRange
}

const (
	DEFAULT_GATEWAY_PORT         = 80
	DEFAULT_GATEWAY_APP_NAME     = "gateway"
	DEFAULT_WEBRTC_UDP_PORT_NAME = "webrtc-udp"
	DEFAULT_WEBRTC_TCP_PORT_NAME = "webrtc-tcp"
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
	ingestWebrtcUDPGatewayServiceName := fmt.Sprintf("%s-webrtc-udp-gateway", params.AppName)
	ingestWebrtcTCPGatewayServiceName := fmt.Sprintf("%s-webrtc-tcp-gateway", params.AppName)
	owner := params.AppName

	webrtcNodePort := s.nodePortRange.GetPort()
	if webrtcNodePort == nil {
		return FullNodePortAssignedError
	}
	webrtcPort := uint16(webrtcNodePort.Port())

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
			WebrtcPort:    webrtcPort,
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
			WebrtcPort:        webrtcPort,
		})

		return s.k8s.Create(params.Context, service)
	})

	g.Go(func() error {
		service := s.ingestResourceManager.IngestWebrtcGatewayService(IngestIngressWebrtcServiceParams{
			Template:           params.Template,
			GatewayServiceName: ingestWebrtcUDPGatewayServiceName,
			AppName:            params.AppName,
			Namespace:          params.Namespace,
			Owner:              owner,
			WebrtcPort:         webrtcPort,
			Protocol:           corev1.ProtocolUDP,
			PortName:           DEFAULT_WEBRTC_UDP_PORT_NAME,
		})

		return s.k8s.Create(params.Context, service)
	})

	g.Go(func() error {
		service := s.ingestResourceManager.IngestWebrtcGatewayService(IngestIngressWebrtcServiceParams{
			Template:           params.Template,
			GatewayServiceName: ingestWebrtcTCPGatewayServiceName,
			AppName:            params.AppName,
			Namespace:          params.Namespace,
			Owner:              owner,
			WebrtcPort:         webrtcPort,
			Protocol:           corev1.ProtocolTCP,
			PortName:           DEFAULT_WEBRTC_TCP_PORT_NAME,
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
		_ = s.nodePortRange.PortBack(webrtcNodePort.Port())

		return fmt.Errorf("unable deploy ingest system. Error: %s", err)
	}

	return nil
}

func findDeploymentIngestWebrtcPort(ingest *appsv1.Deployment) *int32 {
	if ingest == nil {
		return nil
	}

	for _, container := range ingest.Spec.Template.Spec.Containers {
		// TODO: drop sidecars

		for _, port := range container.Ports {
			if port.Name == DEFAULT_WEBRTC_TCP_PORT_NAME || port.Name == DEFAULT_WEBRTC_UDP_PORT_NAME {
				return &port.ContainerPort
			}
		}
	}

	return nil
}

func findServiceIngestWebrtcPort(ingestService *corev1.Service) *int32 {
	if ingestService == nil {
		return nil
	}

	for _, port := range ingestService.Spec.Ports {
		if port.Name == DEFAULT_WEBRTC_TCP_PORT_NAME || port.Name == DEFAULT_WEBRTC_UDP_PORT_NAME {
			return &port.Port
		}
	}

	return nil
}

func findFirstWebrtcPortInDeploymentOrService(ingest *appsv1.Deployment, service *corev1.Service) *int32 {
	var wg sync.WaitGroup
	wg.Add(2)
	first := make(chan *int32, 2)

	go func() {
		defer wg.Done()

		if result := findDeploymentIngestWebrtcPort(ingest); result != nil {
			first <- result
		}
	}()

	go func() {
		defer wg.Done()

		if result := findServiceIngestWebrtcPort(service); result != nil {
			first <- result
		}
	}()

	go func() {
		wg.Wait()
		first <- nil
	}()

	select {
	case result := <-first:
		return result
	}
}

func (s *IngestSystem) StopIngestSystem(context context.Context, appName string, namespace string) error {
	ingressGatewayAppName := fmt.Sprintf("%s-gateway", appName)
	virtualServiceAppName := fmt.Sprintf("%s-virtual-service", appName)
	ingestServiceName := appName
	ingestWebrtcUDPGatewayServiceName := fmt.Sprintf("%s-webrtc-udp-gateway", appName)
	ingestWebrtcTCPGatewayServiceName := fmt.Sprintf("%s-webrtc-tcp-gateway", appName)
	owner := appName

	// NOTE: If delete by namespace i need wait until namespace will terminated

	g := new(errgroup.Group)
	var broadcasterID string
	var ingestDeployment *appsv1.Deployment
	var ingestWebrtcGatewayService *corev1.Service

	g.Go(func() error {
		ingest, err := s.ingestResourceManager.GetIngestByAppName(GetIngestByAppNameParams{
			Context:   context,
			Namespace: namespace,
			AppName:   appName,
		})
		if err != nil {
			return fmt.Errorf("unable find ingest deployment for %s/%s. Error: %s", namespace, appName, err)
		}
		ingestDeployment = ingest

		broadcasterID = ingest.Labels[BROADCASTER_ID]

		return s.k8s.Delete(context, ingest)
	})

	g.Go(func() error {
		service, err := s.ingestResourceManager.GetIngestServiceByAppName(GetIngestServiceByAppName{
			Context:           context,
			Namespace:         namespace,
			IngestServiceName: ingestServiceName,
			Owner:             owner,
		})
		if err != nil {
			return fmt.Errorf("unable find ingest service for %s/%s. Error: %s", namespace, ingestServiceName, err)
		}

		return s.k8s.Delete(context, service)
	})

	g.Go(func() error {
		service, err := s.ingestResourceManager.GetIngestServiceByAppName(GetIngestServiceByAppName{
			Context:           context,
			Namespace:         namespace,
			IngestServiceName: ingestWebrtcUDPGatewayServiceName,
			Owner:             owner,
		})
		if err != nil {
			return fmt.Errorf("unable find ingest webrtc service for %s/%s. Error: %s", namespace, ingestWebrtcUDPGatewayServiceName, err)
		}

		ingestWebrtcGatewayService = service

		return s.k8s.Delete(context, service)
	})

	g.Go(func() error {
		service, err := s.ingestResourceManager.GetIngestServiceByAppName(GetIngestServiceByAppName{
			Context:           context,
			Namespace:         namespace,
			IngestServiceName: ingestWebrtcTCPGatewayServiceName,
			Owner:             owner,
		})
		if err != nil {
			return fmt.Errorf("unable find ingest webrtc service for %s/%s. Error: %s", namespace, ingestWebrtcTCPGatewayServiceName, err)
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
		if result := findFirstWebrtcPortInDeploymentOrService(ingestDeployment, ingestWebrtcGatewayService); result != nil {
			if err := s.nodePortRange.PortBack(portrange.Port(*result)); err != nil {
				log.Println(err)
			}
		}

		return fmt.Errorf("unable stop ingest system. Error: %s", err)
	}

	if result := findFirstWebrtcPortInDeploymentOrService(ingestDeployment, ingestWebrtcGatewayService); result != nil {
		if err := s.nodePortRange.PortBack(portrange.Port(*result)); err != nil {
			return fmt.Errorf("unable webrtc port back of %d. Err: %s", result, err)
		}
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
	NodePortRange         *portrange.PortRange
}

func NewIngestSystem(params IngestSystemParams) *IngestSystem {
	return &IngestSystem{
		k8s:                   params.K8s,
		ingestResourceManager: params.IngestResourceManager,
		istioResourceManager:  params.IstioResourceManager,
		js:                    params.JS,
		nodePortRange:         params.NodePortRange,
	}
}
