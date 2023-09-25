package service

import (
	"log"

	"github.com/romashorodok/stream-platform/pkg/envutils"
	"github.com/romashorodok/stream-platform/pkg/variables"
	"go.uber.org/fx"
)

type IngestSystemConfig struct {
	FailFast      bool
	BroadcasterID string
	Username      string
}

func NewIngestSystemConfig() *IngestSystemConfig {
	failFastRaw := envutils.Env(variables.INGEST_FAIL_FAST, variables.INGEST_FAIL_FAST_DEFAULT)
	failFast, err := envutils.ParseBool(failFastRaw)
	if err != nil {
		log.Printf("[ERROR] wrong fail fast %s. Fallback to %s", failFastRaw, variables.INGEST_FAIL_FAST_DEFAULT)
		failFast, _ = envutils.ParseBool(variables.INGEST_FAIL_FAST_DEFAULT)
	}

	return &IngestSystemConfig{
		BroadcasterID: envutils.Env(variables.INGEST_BROADCASTER_ID, variables.INGEST_BROADCASTER_ID_DEFAULT),
		Username:      envutils.Env(variables.INGEST_USERNAME, variables.INGEST_USERNAME_DEFAULT),
		FailFast:      *failFast,
	}
}

var IngestSystemModule = fx.Module("system",
	fx.Provide(NewIngestSystemConfig),

	fx.Invoke(StartIngestHttp),
	fx.Invoke(StartIngestWebrtc),
)
