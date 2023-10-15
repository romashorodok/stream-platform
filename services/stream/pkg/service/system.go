package service

import (
	"log"

	"github.com/romashorodok/stream-platform/pkg/envutils"
	"github.com/romashorodok/stream-platform/pkg/variables"
	"go.uber.org/fx"
)

type StreamSystemConfig struct {
	Standalone       bool
	IngestStandalone *IngestStandaloneConfig
}

type IngestStandaloneConfig struct {
	Deployment        string
	Namespace         string
	IngestUri         string
	IngestTemplate    string
	IngestWebrtcRoute string
	IngestHLSRoute    string
}

func NewStreamSystemConfig() *StreamSystemConfig {
	standaloneRaw := envutils.Env(variables.STREAM_STANDALONE, variables.STREAM_STANDALONE_DEFAULT)
	standalone, err := envutils.ParseBool(standaloneRaw)
	if err != nil {
		log.Printf("[ERROR] wrong standalone %s. Fallback to %s", standaloneRaw, variables.INGEST_FAIL_FAST_DEFAULT)
		standalone, _ = envutils.ParseBool(variables.STREAM_STANDALONE_DEFAULT)
	}

	return &StreamSystemConfig{
		Standalone: *standalone,

		IngestStandalone: &IngestStandaloneConfig{
			Deployment:        envutils.Env(variables.STREAM_STANDALONE_INGEST_DEPLOYMENT, variables.STREAM_STANDALONE_INGEST_DEPLOYMENT_DEFAULT),
			Namespace:         envutils.Env(variables.STREAM_STANDALONE_INGEST_NAMESPACE, variables.STREAM_STANDALONE_INGEST_NAMESPACE_DEFAULT),
			IngestUri:         envutils.Env(variables.STREAM_STANDALONE_INGEST_URI, variables.STREAM_STANDALONE_INGEST_URI_DEFAULT),
			IngestTemplate:    envutils.Env(variables.STREAM_INGEST_TEMPLATE, variables.STREAM_INGEST_TEMPLATE_DEFAULT),
			IngestWebrtcRoute: envutils.Env(variables.STREAM_STANDALONE_INGEST_EGRESS_WEBRTC, variables.STREAM_STANDALONE_INGEST_EGRESS_WEBRTC_DEFAULT),
			IngestHLSRoute:    envutils.Env(variables.STREAM_STANDALONE_INGEST_EGRESS_HLS, variables.STREAM_STANDALONE_INGEST_EGRESS_HLS_DEFAULT),
		},
	}
}

var StreamSystemModule = fx.Module("system",
	fx.Provide(NewStreamSystemConfig),
)
