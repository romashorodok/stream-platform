package ingestresource

import (
	"context"
	"errors"
	"fmt"

	v1alpha1 "github.com/romashorodok/stream-platform/operators/ingestion-operator/api/romashorodok.github.io"
	"github.com/romashorodok/stream-platform/pkg/variables"
	"go.uber.org/fx"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IngestResourceManager struct {
	k8s client.Client
}

const (
	OWNED_BY            = "app.kubernetes.io/owned-by"
	CREATED_BY          = "app.kubernetes.io/created-by"
	BROADCASTER_ID      = "romashorodok.github.io/ingest.broadcaster-id"
	ISTIO_SIDECAR_LABEL = "sidecar.istio.io/inject"
	ISTIO_SIDECAR_VALUE = "true"
)

type IngestDeploymentByTemplateParams struct {
	Template   *v1alpha1.IngestTemplate
	AppName    string
	Namespace  string
	Replicas   int32
	Owner      string
	WebrtcPort uint16

	BroadcasterID string
	Username      string
}

func (mgr *IngestResourceManager) IngestDeploymentByTemplate(params IngestDeploymentByTemplateParams) *appsv1.Deployment {
	deploymentLabels := labels.Set{
		OWNED_BY:       params.Owner,
		CREATED_BY:     params.Template.Name,
		BROADCASTER_ID: params.BroadcasterID,
	}
	appLabels := labels.Set{
		"app":               params.AppName,
		ISTIO_SIDECAR_LABEL: ISTIO_SIDECAR_VALUE,
	}

	ports := append(params.Template.Spec.Ports,
		corev1.ContainerPort{
			Name:          DEFAULT_WEBRTC_TCP_PORT_NAME,
			ContainerPort: int32(params.WebrtcPort),
			Protocol:      corev1.ProtocolTCP,
		},
		corev1.ContainerPort{
			Name:          DEFAULT_WEBRTC_UDP_PORT_NAME,
			ContainerPort: int32(params.WebrtcPort),
			Protocol:      corev1.ProtocolUDP,
		},
	)

	ingestContainer := corev1.Container{
		Name:  params.AppName,
		Image: params.Template.Spec.Image,
		Ports: ports,
		Env: []corev1.EnvVar{
			{Name: variables.INGEST_BROADCASTER_ID, Value: params.BroadcasterID},
			{Name: variables.INGEST_USERNAME, Value: params.Username},

			{Name: variables.INGEST_UDP_PORT, Value: fmt.Sprint(params.WebrtcPort)},
			{Name: variables.INGEST_TCP_PORT, Value: fmt.Sprint(params.WebrtcPort)},
			{Name: variables.INGEST_NAT_PUBLIC_IP, Value: variables.INGEST_NAT_PUBLIC_IP_DEFAULT},

			{Name: variables.NATS_HOST, Value: variables.NATS_HOST_HEADLESS},
			{Name: variables.NATS_PORT, Value: variables.NATS_PORT_DEFAULT},

			// TODO: Instead of expose node port of webrtc handle it by turn server
			{Name: variables.TURN_ENABLE, Value: "false"},
			{Name: variables.TURN_URL, Value: variables.TURN_URL_DEFAULT},
			{Name: variables.TURN_USERNAME, Value: variables.TURN_USERNAME_DEFAULT},
			{Name: variables.TURN_PASSWORD, Value: variables.TURN_PASSWORD_DEFAULT},
		},
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.AppName,
			Namespace: params.Namespace,
			Labels:    deploymentLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &params.Replicas,
			Selector: &metav1.LabelSelector{MatchLabels: appLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: appLabels},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchLabels: appLabels,
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
					Containers: []corev1.Container{ingestContainer},
				},
			},
		},
	}
}

type IngestHeadlessServiceParams struct {
	Template          *v1alpha1.IngestTemplate
	IngestServiceName string
	AppName           string
	Namespace         string
	Owner             string
	WebrtcPort        uint16
}

func (mgr *IngestResourceManager) IngestHeadlessService(params IngestHeadlessServiceParams) *corev1.Service {

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.IngestServiceName,
			Namespace: params.Namespace,
			Labels: labels.Set{
				"app":      params.IngestServiceName,
				CREATED_BY: params.Template.Name,
				OWNED_BY:   params.Owner,
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels.Set{"app": params.AppName},
		},
	}

	for _, port := range params.Template.Spec.Ports {
		service.Spec.Ports = append(
			service.Spec.Ports,
			corev1.ServicePort{
				Port:     port.ContainerPort,
				Name:     string(port.Name),
				Protocol: port.Protocol,
			},
		)
	}

	return service
}

type IngestIngressWebrtcServiceParams struct {
	Template           *v1alpha1.IngestTemplate
	GatewayServiceName string
	AppName            string
	Namespace          string
	Owner              string
	WebrtcPort         uint16
	Protocol           corev1.Protocol
	PortName           string
}

func (mgr *IngestResourceManager) IngestWebrtcGatewayService(params IngestIngressWebrtcServiceParams) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.GatewayServiceName,
			Namespace: params.Namespace,
			Labels: labels.Set{
				"app":      params.GatewayServiceName,
				CREATED_BY: params.Template.Name,
				OWNED_BY:   params.Owner,
			},
		},

		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Name:     params.PortName,
					Port:     int32(params.WebrtcPort),
					Protocol: params.Protocol,
				},
			},
			Selector: labels.Set{"app": params.AppName},
		},
	}

	return service
}

type IngestNamespaceParams struct {
	Namespace string
	Owner     string
}

func (mgr *IngestResourceManager) IngestNamespace(params IngestNamespaceParams) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: params.Namespace,
			Labels: labels.Set{
				OWNED_BY: params.Owner,
			},
		},
	}
}

type GetIngestByAppNameParams struct {
	Context   context.Context
	Namespace string
	AppName   string
}

func (mgr *IngestResourceManager) GetIngestByAppName(params GetIngestByAppNameParams) (*appsv1.Deployment, error) {
	var ingestDeployment appsv1.Deployment
	namespacedName := types.NamespacedName{Namespace: params.Namespace, Name: params.AppName}

	if err := mgr.k8s.Get(params.Context, namespacedName, &ingestDeployment); err != nil {
		return nil, err
	}

	owner, ok := ingestDeployment.Labels[OWNED_BY]
	if !ok {
		return nil, errors.New("trying to get non-owner resource access")
	}

	if owner != params.AppName {
		return nil, errors.New("trying to get non-owner resource access")
	}

	return &ingestDeployment, nil
}

type GetIngestServiceByAppName struct {
	Context           context.Context
	Namespace         string
	IngestServiceName string
	Owner             string
}

func (mgr *IngestResourceManager) GetIngestServiceByAppName(params GetIngestServiceByAppName) (*corev1.Service, error) {
	var service corev1.Service
	namespacedName := types.NamespacedName{Namespace: params.Namespace, Name: params.IngestServiceName}

	if err := mgr.k8s.Get(params.Context, namespacedName, &service); err != nil {
		return nil, err
	}

	owner, ok := service.Labels[OWNED_BY]
	if !ok {
		return nil, errors.New("trying to get non-owner resource access")
	}

	if owner != params.Owner {
		return nil, errors.New("trying to get non-owner resource access")
	}

	return &service, nil
}

type GetIngestNamespaceByAppName struct {
	Context context.Context
	AppName string
}

func (mgr *IngestResourceManager) GetIngestNamespaceByAppName(params GetIngestNamespaceByAppName) (*corev1.Namespace, error) {
	var namespace corev1.Namespace
	namespacedName := types.NamespacedName{Namespace: params.AppName, Name: params.AppName}

	if err := mgr.k8s.Get(params.Context, namespacedName, &namespace); err != nil {
		return nil, err
	}

	owner, ok := namespace.Labels[OWNED_BY]
	if !ok {
		return nil, errors.New("trying to get non-owner resource access")
	}

	if owner != params.AppName {
		return nil, errors.New("trying to get non-owner resource access")
	}

	return &namespace, nil
}

type IngestResourceManagerParams struct {
	fx.In

	K8s client.Client
}

func NewIngestResourceManager(params IngestResourceManagerParams) *IngestResourceManager {
	return &IngestResourceManager{k8s: params.K8s}
}
