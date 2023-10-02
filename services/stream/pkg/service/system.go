package service

import (
	"log"

	"github.com/romashorodok/stream-platform/pkg/envutils"
	"github.com/romashorodok/stream-platform/pkg/variables"
	"go.uber.org/fx"
)

type StreamSystemConfig struct {
	Standalone     bool
	Deployment     string
	Namespace      string
	IngestUri      string
	IngestTemplate string
}

func NewStreamSystemConfig() *StreamSystemConfig {
	standaloneRaw := envutils.Env(variables.STREAM_STANDALONE, variables.STREAM_STANDALONE_DEFAULT)
	standalone, err := envutils.ParseBool(standaloneRaw)
	if err != nil {
		log.Printf("[ERROR] wrong standalone %s. Fallback to %s", standaloneRaw, variables.INGEST_FAIL_FAST_DEFAULT)
		standalone, _ = envutils.ParseBool(variables.STREAM_STANDALONE_DEFAULT)
	}

	return &StreamSystemConfig{
		Standalone:     *standalone,
		Deployment:     envutils.Env(variables.STREAM_STANDALONE_INGEST_DEPLOYMENT, variables.STREAM_STANDALONE_INGEST_DEPLOYMENT_DEFAULT),
		Namespace:      envutils.Env(variables.STREAM_STANDALONE_INGEST_NAMESPACE, variables.STREAM_STANDALONE_INGEST_NAMESPACE_DEFAULT),
		IngestUri:      envutils.Env(variables.STREAM_STANDALONE_INGEST_URI, variables.STREAM_STANDALONE_INGEST_URI_DEFAULT),
		IngestTemplate: envutils.Env(variables.STREAM_INGEST_TEMPLATE, variables.STREAM_INGEST_TEMPLATE_DEFAULT),
	}
}

var StreamSystemModule = fx.Module("system",
	fx.Provide(NewStreamSystemConfig),
)
