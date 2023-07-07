package whip

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

// Each broadcaster sends sdp offer with `m=` field which describe stream as example:
// m=audio 54208 UDP/TLS/RTP/SAVPF 111
// a=rtpmap:111 OPUS/48000/2
//
// 54208 - is suggested port number. The port for assigning determine the ICE (Interactive Connectivity Establishment) server for the peers.
// 111 - WebRTC payload type. Where do i find this number ?
// 48,000 Hz for codec
// 2 audio channels

// m=video 54208 UDP/TLS/RTP/SAVPF 96
// a=rtpmap:96 H264/90000

type whipHandler interface {
	Handler(res http.ResponseWriter, r *http.Request)
}

type handler struct {
	whipHandler

	webrtcAPI *webrtc.API

	streamMutex sync.Mutex
}

func NewHandler(webrtcAPI *webrtc.API) *handler {
	return &handler{webrtcAPI: webrtcAPI}
}

type AudioChannel struct {
	Track *webrtc.TrackLocalStaticRTP

	channelMutex sync.RWMutex
}

type VideoChannel struct {
	Track          *webrtc.TrackLocalStaticRTP
	Layer          atomic.Value
	Timestamp      uint32
	SequenceNumber uint16

	packetLossCh chan any
	channelMutex sync.RWMutex
}

func (c *VideoChannel) SendVideoPacket(rtpPkt *rtp.Packet, layer string, timeDiff uint32) {
	if c.Layer.Load() == "" {
		c.Layer.Store(layer)
	} else if c.Layer.Load() != layer {
		return
	}

	c.SequenceNumber += 1
	c.Timestamp += timeDiff

	rtpPkt.SequenceNumber = c.SequenceNumber
	rtpPkt.Timestamp = c.Timestamp

	if err := c.Track.WriteRTP(rtpPkt); err != nil && !errors.Is(err, io.ErrClosedPipe) {
		log.Println(err)
	}
}

type Stream struct {
	VideoChannel
	AudioChannel
}

func GetStream(streamKey string) *Stream {
	stream := &Stream{}
	stream.VideoChannel = VideoChannel{}
	stream.AudioChannel = AudioChannel{}

	// TODO: Remove hard coding
	audioTrack, _ := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
	stream.AudioChannel.Track = audioTrack

	return stream
}

const (
	defaultVideoTrack = "default"
)

func audioWriter(remoteTrack *webrtc.TrackRemote, s *Stream) {

	rtpBuf := make([]byte, 1500)

	for {
		rtpRead, _, err := remoteTrack.Read(rtpBuf)

		// log.Println("Write audio")

		switch {
		case errors.Is(err, io.EOF):
			return

		case err != nil:
			log.Println(err)
			return
		}

		if _, writeErr := s.AudioChannel.Track.Write(rtpBuf[:rtpRead]); writeErr != nil && !errors.Is(writeErr, io.ErrClosedPipe) {
			log.Println(writeErr)
			return
		}
	}
}

func videoWriter(remoteTrack *webrtc.TrackRemote, peerConnection *webrtc.PeerConnection, s *Stream) {

	id := remoteTrack.RID()
	if id == "" {
		id = defaultVideoTrack
	}

	rtpBuf := make([]byte, 1500)
	rtpPkt := &rtp.Packet{}
	lastTimestamp := uint32(0)
	for {
		rtpRead, _, err := remoteTrack.Read(rtpBuf)

		// log.Println("Write video")

		switch {
		case errors.Is(err, io.EOF):
			return
		case err != nil:
			log.Println(err)
			return
		}

		if err = rtpPkt.Unmarshal(rtpBuf[:rtpRead]); err != nil {
			log.Println(err)
			return
		}

		timeDiff := rtpPkt.Timestamp - lastTimestamp
		if lastTimestamp == 0 {
			timeDiff = 0
		}
		lastTimestamp = rtpPkt.Timestamp

		s.VideoChannel.channelMutex.RLock()
		s.VideoChannel.SendVideoPacket(rtpPkt, id, timeDiff)
		s.VideoChannel.channelMutex.RUnlock()
	}
}

func (h *handler) Handler(res http.ResponseWriter, r *http.Request) {
	h.streamMutex.Lock()
	defer h.streamMutex.Unlock()

	streamKey := r.Header.Get("Authorization")

	if streamKey == "" {
		log.Println("Authorization header not set")
		return
	}

	offer, err := io.ReadAll(r.Body)

	if err != nil {
		log.Println("SDP offer is empty")
		return
	}

	peerConnection, err := h.webrtcAPI.NewPeerConnection(webrtc.Configuration{})

	stream := GetStream(streamKey)

	peerConnection.OnTrack(
		func(track *webrtc.TrackRemote, rtp *webrtc.RTPReceiver) {
			log.Println(rtp.GetParameters().Codecs)

			// NOTE: Here is idea how to dynamicly set codecs
			// NOTE: here i get mime type like video/audio and webrtc payload type which sets in media engine at startup
			// for _, codecs := range rtp.GetParameters().Codecs {
			// 	log.Println(codecs.MimeType, codecs.PayloadType)
			// }

			if strings.HasPrefix(track.Codec().RTPCodecCapability.MimeType, "audio") {
				audioWriter(track, stream)

			} else {
				videoWriter(track, peerConnection, stream)
			}
		},
	)

	if err := peerConnection.SetRemoteDescription(webrtc.SessionDescription{
		SDP:  string(offer),
		Type: webrtc.SDPTypeOffer,
	}); err != nil {
		return
	}

	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	answer, err := peerConnection.CreateAnswer(nil)

	if err != nil {
		log.Println(err)
		return
	} else if err = peerConnection.SetLocalDescription(answer); err != nil {
		log.Println(err)
		return
	}

	<-gatherComplete

	res.WriteHeader(http.StatusCreated)
	fmt.Fprint(res, peerConnection.LocalDescription().SDP)
}
