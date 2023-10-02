package stream

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	streamingpb "github.com/romashorodok/stream-platform/gen/golang/streaming/v1alpha"
	"github.com/romashorodok/stream-platform/pkg/auth"
	"github.com/romashorodok/stream-platform/pkg/httputils"
	"github.com/romashorodok/stream-platform/pkg/subject"
	"github.com/romashorodok/stream-platform/pkg/tokenutils"
	"github.com/romashorodok/stream-platform/services/stream/pkg/wspeer"
)

func notifyPeerWhenIngestDeployed(peer *wspeer.WebsocketPeer, conn *nats.Conn, auth *auth.TokenPayload) {
	log.Printf("[%s] Subscribe to ingest deploymend.", auth.Sub)

	subscription, err := conn.Subscribe(subject.NewIngestDeployed(auth.UserID.String()), func(msg *nats.Msg) {
		var req subject.IngestDeployed

		if err := subject.DeserializeProtobufMsg(&req, msg); err != nil {
			log.Printf("[%s] Unable deserialize protobuf message. Err: %s", auth.Sub, err)
			return
		}

		status := &streamingpb.StreamStatus{Running: false, Deployed: req.Deployed}

		if err := peer.WriteProtobuf(status); err != nil {
			log.Printf("[%s] Unable send ingest deployed notification protobuf message to peer. Err: %s", auth.Sub, err)
			return
		}

		log.Printf("[%s] Peer notified for %s", auth.Sub, subject.NewIngestDeployed(auth.UserID.String()))
	})
	defer subscription.Drain()

	if err != nil {
		log.Printf("[%s] Unable start subscription when ingest deployed. Err: %s", auth.Sub, err)
	}

	<-peer.Done()
}

func notifyPeerWhenStreamDestroyedNotification(peer *wspeer.WebsocketPeer, conn *nats.Conn, auth *auth.TokenPayload) {
	log.Printf("[%s] Subscribe to stream destroyed.", auth.Sub)

	subscription, err := conn.Subscribe(subject.NewStreamDestroyedNotification(auth.UserID.String()), func(msg *nats.Msg) {
		status := &streamingpb.StreamStatus{Running: false, Deployed: false}

		if err := peer.WriteProtobuf(status); err != nil {
			log.Printf("[%s] Unable send stream destroyed notification protobuf message to peer. Err: %s", auth.Sub, err)
			return
		}

		log.Printf("[%s] Peer notified for %s", auth.Sub, subject.NewStreamDestroyedNotification(auth.UserID.String()))
	})

	defer subscription.Drain()

	if err != nil {
		log.Printf("[%s] Unable start subscription when stream destroyed. Err: %s", auth.Sub, err)
	}

	<-peer.Done()
}

func (s *StreamingService) StreamingServiceStreamChannel(w http.ResponseWriter, r *http.Request) {
	plainToken, err := r.Cookie(tokenutils.REFRESH_TOKEN_COOKIE_NAME)
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusPreconditionRequired, "Unable get refresh token cookie.", err.Error())
		return
	}

	tokenPayload, err := s.refreshTokenAuth.Validate(r.Context(), plainToken.Value)
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusPreconditionRequired, "Invalid refresh token.", err.Error())
		return
	}

	payload, err := auth.WithRawTokenPayload(tokenPayload)
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "Unable deserialize token payload", err.Error())
		return
	}

	log.Printf("[%s] Subscriber identity: %+v", payload.Sub, payload)

	conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)

	peer := wspeer.NewWebsocketPeer(conn, r.Context())
	go peer.Start()

	go notifyPeerWhenIngestDeployed(peer, s.nats, payload)
	go notifyPeerWhenStreamDestroyedNotification(peer, s.nats, payload)

	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "Unable upgrade http request", err.Error())
		return
	}

	_ = peer.WriteProtobuf(s.streamStatus.IsRunning(payload))

	go func() {
		for {
			msg := <-peer.Recv()

			log.Println(string(msg.Data))

		}
	}()

	<-peer.Done()
	log.Println("Done")
}
