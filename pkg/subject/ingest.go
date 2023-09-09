package subject

import (
	"strings"

	subjectpb "github.com/romashorodok/stream-platform/gen/golang/subject/v1alpha"
)

// https://stackoverflow.com/a/74571646

// TODO: make better name
const IngestAnyUserDeployed = "ingest.*.deployed"

type IngestDeployed = subjectpb.IngestDeployed

func NewIngestDeployed(broadcasterID string) string {
	return strings.Replace(IngestAnyUserDeployed, "*", broadcasterID, 1)
}

const IngestAnyUserDestroyed = "public.ingest.in.*.destroyed.protobuf"

type IngestDestroyed = subjectpb.IngestDestroyed

func NewIngestDestroyed(broadcasterID string) string {
	return strings.Replace(IngestAnyUserDestroyed, "*", broadcasterID, 1)
}
