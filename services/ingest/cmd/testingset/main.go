package main

import (
	"github.com/romashorodok/stream-platform/pkg/httputils"
	"github.com/romashorodok/stream-platform/pkg/shutdown"
	"github.com/romashorodok/stream-platform/services/ingest/internal/egress/hls"
	"github.com/romashorodok/stream-platform/services/ingest/internal/egress/whep"
	"github.com/romashorodok/stream-platform/services/ingest/internal/ingress/whip"
	"github.com/romashorodok/stream-platform/services/ingest/internal/mediaprocessor"
	"github.com/romashorodok/stream-platform/services/ingest/internal/statefulstream"
	"github.com/romashorodok/stream-platform/services/ingest/internal/statefulstream/webrtcstatefulstream"
	"github.com/romashorodok/stream-platform/services/ingest/pkg/service"
	"go.uber.org/fx"
)

func main() {
	shdown := shutdown.NewShutdown()

	app := fx.New(
		service.NatsModule,
		service.ServerModule,
		service.IngestSystemModule,
		// Standalone modules must not depend on internal modules
		// Internal modules may use service modules. And may follow their protocols

		// Internal providing must be here
		fx.Provide(webrtcstatefulstream.NewWebrtcAllocatorFunc),
		fx.Provide(statefulstream.NewStatefulStreamGlobal),

		// Handlers
		fx.Provide(httputils.AsHttpHandler(whip.NewWhipHandler)),
		fx.Provide(httputils.AsHttpHandler(whep.NewWhepHandler)),
		fx.Provide(httputils.AsHttpHandler(hls.NewHLSHandler)),

		// Media processors
		fx.Provide(mediaprocessor.FxDefaultHLSMediaProcessor),

		fx.Provide(func() *shutdown.Shutdown {
			return shdown
		}),
	)

	go app.Run()

	<-app.Wait()

	shdown.Gracefully()
}
