package main

import (
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	subjectpb "github.com/romashorodok/stream-platform/gen/golang/subject/v1alpha"
	"github.com/romashorodok/stream-platform/pkg/subject"
)

func main() {
	conn, _ := nats.Connect(nats.DefaultURL)
	defer conn.Drain()

	js, _ := conn.JetStream()
	js.AddStream(subject.INGEST_DESTROYING_STREAM_CONFIG)

	broadcasterID := uuid.NullUUID{}.UUID.String()
	username := "nullable user"

	_, _ = subject.JsPublishProtobufWithID(js, subject.NewIngestDestroyed(broadcasterID), broadcasterID, &subject.IngestDestroyed{
		Destroyed: true,

		Meta: &subjectpb.BroadcasterMeta{
			BroadcasterId: broadcasterID,
			Username:      username,
		},
	})

	id, _ := uuid.NewUUID()
	broadcasterID = id.String()
	username = "nullable user"

	_, _ = subject.JsPublishProtobufWithID(js, subject.NewIngestDestroyed(broadcasterID), broadcasterID, &subject.IngestDestroyed{
		Destroyed: true,

		Meta: &subjectpb.BroadcasterMeta{
			BroadcasterId: broadcasterID,
			Username:      username,
		},
	})
}
