package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

const (
	endpoint = "http://127.0.0.1:8089/api/consumer/whip"
)

func NewWebrtcAPI() *webrtc.API {
	mediaEngine := &webrtc.MediaEngine{}
	mediaSettings := webrtc.SettingEngine{}

	mediaEngine.RegisterDefaultCodecs()

	return webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(mediaSettings),
	)
}

func WHIPRequest(sdpOffer string) (sdpAnswer string) {
	reader := strings.NewReader(sdpOffer)

	client := http.Client{}

	request, err := http.NewRequest("POST", endpoint, reader)

	if err != nil {
		log.Panicf("Unable create request. Err: %s", err)
	}

	request.Header.Set("Content-Type", "application/sdp")

	resp, err := client.Do(request)

	if err != nil {
		log.Panicf("Unable to send request. Err: %s", err)
	}

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		log.Panicf("Invalid request server should create resource. Resp body: %s", body)
	}

	whipSDPAnswer := string(body)

	return whipSDPAnswer
}

const (
	FFMPEG_VIDEO_RTP_LISTENER_HOST = "rtp://localhost:16384"
	FFMPEG_AUDIO_RTP_LISTENER_HOST = "rtp://localhost:16385"

	RTP_VIDEO_PORT = ":16384"
	RTP_AUDIO_PORT = ":16385"
)

func WriteUDPToTrack(conn *net.UDPConn, track *webrtc.TrackLocalStaticRTP) {
	rtpBuff := make([]byte, 1500)
	rtpPacket := &rtp.Packet{}

	for {
		rtpBuffN, err := conn.Read(rtpBuff)

		if err != nil {
			log.Println("Error reading RTP packet:", err)
			continue
		}

		if err = rtpPacket.Unmarshal(rtpBuff[:rtpBuffN]); err != nil {
			log.Println("On rtp reading cannot deserialize the rtp packet. Err:", err)
			continue
		}

		track.WriteRTP(rtpPacket)
	}
}

func main() {

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	videoAddr, _ := net.ResolveUDPAddr("udp", RTP_VIDEO_PORT)
	videoConn, _ := net.ListenUDP("udp", videoAddr)
	defer videoConn.Close()

	audioAddr, _ := net.ResolveUDPAddr("udp", RTP_AUDIO_PORT)
	audioConn, _ := net.ListenUDP("udp", audioAddr)
	defer audioConn.Close()

	webrtcAPI := NewWebrtcAPI()

	peerConnection, err := webrtcAPI.NewPeerConnection(webrtc.Configuration{})

	if err != nil {
		log.Fatal("Cannot create peer connection. Err:", err)
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Printf("PeerConnection State has changed %s \n", connectionState.String())
	})

	videoTrack, _ := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264},
		"video",
		"pion",
	)

	audioTrack, _ := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		"audio",
		"pion",
	)

	_, _ = peerConnection.AddTrack(videoTrack)

	_, _ = peerConnection.AddTrack(audioTrack)

	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("PeerConnection State has changed %s \n", state.String())
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			log.Printf("PeerConnection ice candidate", candidate.String)
		}
	})

	sdpOffer, _ := peerConnection.CreateOffer(nil)
	sdpAnswer := WHIPRequest(sdpOffer.SDP)

	answer := webrtc.SessionDescription{}
	answer.Type = webrtc.SDPTypeAnswer
	answer.SDP = sdpAnswer

	if err = peerConnection.SetLocalDescription(sdpOffer); err != nil {
		log.Fatal("PeerConnection could not set local offer. ", err)
	}

	if err = peerConnection.SetRemoteDescription(answer); err != nil {
		log.Fatal("Peer could not set remote sdp answer. Err:", err)
	}

	<-webrtc.GatheringCompletePromise(peerConnection)

	go startFFmpegAudioRTPSource()
	go startFFmpegVideoRTPSource()

	go WriteUDPToTrack(videoConn, videoTrack)
	go WriteUDPToTrack(audioConn, audioTrack)

	<-interrupt
}
