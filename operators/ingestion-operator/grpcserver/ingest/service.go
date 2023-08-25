package ingest

import (
	"context"
	"errors"

	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/api/v1alpha1"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/grpcserver/container"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IngestServerDeploymentOpts struct {
	Name      string
	Namespace string
	Replicas  int32
}

type IngestServerDeployment struct {
	template  v1alpha1.IngestTemplate
	client    client.Client
	name      string
	namespace string
	replicas  int32
}

func NewIngestDeploymentFactory(client client.Client, template v1alpha1.IngestTemplate, opts *IngestServerDeploymentOpts) *IngestServerDeployment {
	return &IngestServerDeployment{
		client:    client,
		template:  template,
		name:      opts.Name,
		namespace: opts.Namespace,
		replicas:  opts.Replicas,
	}
}

func (d *IngestServerDeployment) NewDeploymentByTemplate() *appsv1.Deployment {
	deploymentLabels := labels.Set{
		"app.kubernetes.io/created-by": d.template.Name,
		"app.kubernetes.io/owned-by":   d.name,
	}
	podLabels := labels.Set{
		"app": d.name,
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.name,
			Namespace: d.namespace,
			Labels:    deploymentLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &d.replicas,
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: podLabels},

				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    d.name,
							Image:   d.template.Spec.Image,
							Command: []string{"sleep"},
							Args:    []string{"infinity"},
						},
					},
				},
			},
		},
	}
}

type IngestServerResourceManager struct {
	client client.Client
}

const OWNED_BY = "app.kubernetes.io/owned-by"

func (m *IngestServerResourceManager) GetByOwnedByDeploymentName(context context.Context, namespace, name string) (*appsv1.Deployment, error) {
	var ingestServer appsv1.Deployment
	namespacedName := types.NamespacedName{Namespace: namespace, Name: name}

	if err := m.client.Get(context, namespacedName, &ingestServer); err != nil {
		return nil, err
	}

	ownerName, found := ingestServer.Labels[OWNED_BY]

	if !found {
		return nil, errors.New("resource don't has owner. Cannot get it")
	}

	if ownerName != name {
		return nil, errors.New("trying to get non-owner resource access")
	}

	return &ingestServer, nil
}

type IngestControllerService struct {
	ingestioncontrollerpb.UnimplementedIngestControllerServiceServer

	client          client.Client
	resourceManager IngestServerResourceManager
}

var _ ingestioncontrollerpb.IngestControllerServiceServer = (*IngestControllerService)(nil)

func NewIngestControllerService(client client.Client) *IngestControllerService {
	return &IngestControllerService{
		client:          client,
		resourceManager: IngestServerResourceManager{client: client},
	}
}

func (s *IngestControllerService) StartServer(context context.Context, req *ingestioncontrollerpb.StartServerRequest) (*ingestioncontrollerpb.StartServerResponse, error) {
	log := container.WithLogr(context)

	ingestTemplates := container.WithIngestTemplates()

	ingestTemplate, err := ingestTemplates.Get(req.IngestTemplate)

	if err != nil {
		log.Error(err, "Unable find ingest template")
		return nil, status.Errorf(codes.NotFound, "not found ingest template")
	}

	ingestFactory := NewIngestDeploymentFactory(s.client, *ingestTemplate, &IngestServerDeploymentOpts{
		Name:      req.Deployment,
		Namespace: req.Namespace,
		Replicas:  1,
	})

	ingestDeployment := ingestFactory.NewDeploymentByTemplate()

	if err := s.client.Create(context, ingestDeployment); err != nil {
		log.Error(err, "Unable create deployment")
		return nil, status.Errorf(codes.Aborted, "unable create deployment. %s", err)
	}

	return &ingestioncontrollerpb.StartServerResponse{
		Deployment: ingestDeployment.Name,
		Namespace:  ingestDeployment.Namespace,
	}, nil
}

func (s *IngestControllerService) StopServer(context context.Context, req *ingestioncontrollerpb.StopServerRequest) (*ingestioncontrollerpb.StopServerResponse, error) {
	log := container.WithLogr(context)

	resource, err := s.resourceManager.GetByOwnedByDeploymentName(context, req.Namespace, req.Deployment)

	if err != nil {
		log.Error(err, "Unable get ingest resource")
		return nil, status.Errorf(codes.NotFound, "not found ingest server for %s/%s", req.Namespace, req.Deployment)
	}

	err = s.client.Delete(context, resource)

	if err != nil {
		log.Error(err, "Unable delete ingest server resource")
		return nil, status.Error(codes.FailedPrecondition, "unable delete ingest deployment")
	}

	return &ingestioncontrollerpb.StopServerResponse{}, nil
}
