package ingest

import (
	"context"
	"fmt"

	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/grpcserver/container"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IngestControllerService struct {
	ingestioncontrollerpb.UnimplementedIngestControllerServiceServer

	client client.Client
}

func NewIngestControllerService(client client.Client) *IngestControllerService {
	return &IngestControllerService{client: client}
}

func (s *IngestControllerService) StartServer(context context.Context, req *ingestioncontrollerpb.StartServerRequest) (*ingestioncontrollerpb.StartServerResponse, error) {
	log := container.WithLogr(context)

	log.Info("From start server", "someField", "somefieldValue")

	objectList := &corev1.PodList{}

	if err := s.client.List(context, objectList, client.InNamespace("default")); err != nil {
		fmt.Println(err)
		return nil, status.Errorf(codes.NotFound, "not found objects in k8s")
	}

	fmt.Println("Found objects in kube-system: ", objectList)

	return &ingestioncontrollerpb.StartServerResponse{}, nil
}
