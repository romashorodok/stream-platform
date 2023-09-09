package ingestresource

import (
	"context"
	"errors"

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
	Template  *v1alpha1.IngestTemplate
	AppName   string
	Namespace string
	Replicas  int32
	Owner     string

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

	ingestContainer := corev1.Container{
		Name:  params.AppName,
		Image: params.Template.Spec.Image,
		Ports: params.Template.Spec.Ports,
		Env: []corev1.EnvVar{
			{Name: variables.INGEST_BROADCASTER_ID, Value: params.BroadcasterID},
			{Name: variables.INGEST_USERNAME, Value: params.Username},
			{Name: variables.NATS_HOST, Value: "nats-release-headless.nats-system.svc.cluster.local"},
			{Name: variables.NATS_PORT, Value: variables.NATS_PORT_DEFAULT},
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
				Spec:       corev1.PodSpec{Containers: []corev1.Container{ingestContainer}},
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
			Ports:     []corev1.ServicePort{},
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

	if owner != params.IngestServiceName {
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
