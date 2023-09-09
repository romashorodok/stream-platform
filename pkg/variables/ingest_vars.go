package variables

import "github.com/google/uuid"

const (
	INGEST_BROADCASTER_ID = "INGEST_BROADCASTER_ID"
	INGEST_USERNAME       = "INGEST_USERNAME"
	NATS_HOST             = "NATS_HOST"
	NATS_PORT             = "NATS_PORT"
)

const (
	INGEST_USERNAME_DEFAULT = "admin"
	NATS_HOST_DEFAULT       = "0.0.0.0"
	NATS_PORT_DEFAULT       = "4222"
)

var INGEST_BROADCASTER_ID_DEFAULT = uuid.NullUUID{}.UUID.String()
