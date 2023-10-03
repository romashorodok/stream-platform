package subject

import (
	"time"

	"github.com/nats-io/nats.go"
)

const INGEST_DESTROYING_STREAM = "INGEST-DESTROYING"

var INGEST_DESTROYING_STREAM_CONFIG = &nats.StreamConfig{
	Name:      INGEST_DESTROYING_STREAM,
	Retention: nats.InterestPolicy,
	Subjects:  []string{IngestAnyUserDestroyed},
	Discard:   nats.DiscardOld,
	// NOTE: It's not replace nack subjects. Even with same msg id
	// Duplicates: time.Millisecond * 100,
	Duplicates: time.Millisecond,
}
