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

// package main

// import (
//   "encoding/json"
//   "io"
//   "net/http"
//   "os"
// )

// func main() {
//   http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//     request := make(map[string]interface{})

// 	// Read from body and write into stdout
// 	bodyreader := io.TeeReader(r.Body, os.Stdout)
// 	secondbodyreader := io.TeeReader(bodyreader, os.Stdout)
// 	// It's may be like many writes by read from pipe and write in some other place

// 	// I read from pipe and this is also reader

// 	// Read body from TeeReader and make request obj
//     json.NewDecoder(secondbodyreader).
//       Decode(&request)

//     response := map[string]string{
//         "message": "Looks very good!",
//     }

// 	multiwr := io.MultiWriter(os.Stdout, w)

// 	json.NewEncoder(multiwr).Encode(response)

// 	// My replay peers
// 	myReplay := io.MultiWriter(os.Stdout, os.Stdout)

// 	// Read my track and replay it to the peers and have all replayed data here
// 	track := io.TeeReader(r.Body, myReplay)

// 	_ = track

//     // json.NewEncoder(io.MultiWriter(os.Stdout, w)).
//     //   Encode(response)
//   })

//   http.ListenAndServe(":8888", nil)
// }

