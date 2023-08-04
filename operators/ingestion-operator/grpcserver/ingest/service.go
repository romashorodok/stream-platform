package ingest

import (
	"context"

	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/api/v1alpha1"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/grpcserver/container"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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

type IngestControllerService struct {
	ingestioncontrollerpb.UnimplementedIngestControllerServiceServer

	client client.Client
}

func NewIngestControllerService(client client.Client) *IngestControllerService {
	return &IngestControllerService{client: client}
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
		Name:      req.Username,
		Namespace: "default",
		Replicas:  1,
	})

	ingestDeployment := ingestFactory.NewDeploymentByTemplate()

	if err := s.client.Create(context, ingestDeployment); err != nil {
		log.Error(err, "Unable create deployment")
		return nil, status.Errorf(codes.Aborted, "unable create deployment. %s", err)
	}

	return &ingestioncontrollerpb.StartServerResponse{}, nil
}
