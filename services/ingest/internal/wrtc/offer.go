package wrtc

import (
	"context"
	"errors"

	"github.com/pion/webrtc/v3"
)

var (
	PipeRemoteTrackEOF = errors.New("End of remote track")
)

func Answer(peer *webrtc.PeerConnection, offer string) (string, error) {

	if err := peer.SetRemoteDescription(webrtc.SessionDescription{
		SDP:  string(offer),
		Type: webrtc.SDPTypeOffer,
	}); err != nil {
		return "", err
	}

	session, err := peer.CreateAnswer(nil)
	if err != nil {
		return "", err
	}

	if err := peer.SetLocalDescription(session); err != nil {
		return "", err
	}

	<-webrtc.GatheringCompletePromise(peer)

	return peer.LocalDescription().SDP, nil
}

func PipeRemoteTrack(ctx context.Context, remoteTrack *webrtc.TrackRemote, localTrack *webrtc.TrackLocalStaticRTP) error {
	for {
		select {
		case <-ctx.Done():
			return PipeRemoteTrackEOF
		default:
			packet, _, err := remoteTrack.ReadRTP()
			if err != nil {
				return err
			}

			if err := localTrack.WriteRTP(packet); err != nil {
				return PipeRemoteTrackEOF
			}
		}
	}
}
