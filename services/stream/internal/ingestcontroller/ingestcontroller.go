package ingestcontroller

import (
	"context"

	"github.com/nats-io/nats.go"
	ingestioncontrollerpb "github.com/romashorodok/stream-platform/gen/golang/ingestion_controller_operator/v1alpha"
	subjectpb "github.com/romashorodok/stream-platform/gen/golang/subject/v1alpha"
	"github.com/romashorodok/stream-platform/pkg/subject"
	"github.com/romashorodok/stream-platform/services/stream/pkg/service"
	"google.golang.org/grpc"
)

type IngestController interface {
	StartServer(ctx context.Context, in *ingestioncontrollerpb.StartServerRequest, opts ...grpc.CallOption) (*ingestioncontrollerpb.StartServerResponse, error)
	StopServer(ctx context.Context, in *ingestioncontrollerpb.StopServerRequest, opts ...grpc.CallOption) (*ingestioncontrollerpb.StopServerResponse, error)
}

type StandaloneIngestControllerStub struct {
	ingestioncontrollerpb.UnimplementedIngestControllerServiceServer

	config *service.StreamSystemConfig
	conn   *nats.Conn
	js     nats.JetStreamContext
}

func (ctrl *StandaloneIngestControllerStub) StartServer(ctx context.Context, in *ingestioncontrollerpb.StartServerRequest, _ ...grpc.CallOption) (*ingestioncontrollerpb.StartServerResponse, error) {
	_ = subject.PublishProtobuf(
		ctrl.conn,
		/* broadcasterID should be obtained from config of the ingest server that has been deployed specifically for each broadcaster. */
		subject.NewIngestDeployed(in.Meta.BroadcasterId),
		&subject.IngestDeployed{
			Deployed: true,
			Meta:     &subjectpb.BroadcasterMeta{BroadcasterId: in.Meta.BroadcasterId, Username: in.Meta.Username},
			Egresses: []*subjectpb.IngestEgress{
				{Type: subjectpb.IngestEgressType_STREAM_TYPE_WEBRTC},
				{Type: subjectpb.IngestEgressType_STREAM_TYPE_HLS},
			},
		})

	return &ingestioncontrollerpb.StartServerResponse{
		Deployment: ctrl.config.Deployment,
		Namespace:  ctrl.config.Namespace,
	}, nil
}

func (ctrl *StandaloneIngestControllerStub) StopServer(ctx context.Context, in *ingestioncontrollerpb.StopServerRequest, _ ...grpc.CallOption) (*ingestioncontrollerpb.StopServerResponse, error) {
	_, _ = subject.JsPublishProtobufWithID(ctrl.js, subject.NewIngestDestroyed(in.Meta.BroadcasterId), in.Meta.BroadcasterId, &subject.IngestDestroyed{
		Destroyed: true,
		Meta: &subjectpb.BroadcasterMeta{
			BroadcasterId: in.Meta.BroadcasterId,
			Username:      in.Meta.Username,
		},
	})

	return &ingestioncontrollerpb.StopServerResponse{}, nil
}

var _ IngestController = (*StandaloneIngestControllerStub)(nil)
var _ ingestioncontrollerpb.IngestControllerServiceClient = (*StandaloneIngestControllerStub)(nil)

type StandaloneIngestControllerStubParams struct {
	Config *service.StreamSystemConfig
	Conn   *nats.Conn
	JS     nats.JetStreamContext
}

func NewStandaloneIngestControllerStub(params StandaloneIngestControllerStubParams) *StandaloneIngestControllerStub {
	return &StandaloneIngestControllerStub{
		config: params.Config,
		conn:   params.Conn,
		js:     params.JS,
	}
}
