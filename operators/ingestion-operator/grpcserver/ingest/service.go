package ingest

import (
	"context"

	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/grpcserver/container"
)

type IngestControllerService struct {
	ingestioncontrollerpb.UnimplementedIngestControllerServiceServer
}

func (IngestControllerService) StartServer(context context.Context, req *ingestioncontrollerpb.StartServerRequest) (*ingestioncontrollerpb.StartServerResponse, error) {
	log := container.WithLogr(context)

	log.Info("From start server", "someField", "somefieldValue")

	return &ingestioncontrollerpb.StartServerResponse{}, nil
}
