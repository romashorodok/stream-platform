package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"github.com/pion/ice/v2"
	"github.com/pion/webrtc/v3"
	subjectpb "github.com/romashorodok/stream-platform/gen/golang/subject/v1alpha"
	"github.com/romashorodok/stream-platform/pkg/envutils"
	"github.com/romashorodok/stream-platform/pkg/shutdown"
	"github.com/romashorodok/stream-platform/pkg/subject"
	"github.com/romashorodok/stream-platform/pkg/variables"
	"github.com/romashorodok/stream-platform/services/ingest/internal/api/consumer/whip"
	"github.com/romashorodok/stream-platform/services/ingest/internal/api/egress/whep"
	"github.com/romashorodok/stream-platform/services/ingest/internal/config"
	"github.com/romashorodok/stream-platform/services/ingest/internal/orchestrator"
)

func populateMediaEngine(m *webrtc.MediaEngine) error {
	for _, codec := range []webrtc.RTPCodecParameters{
		{
			// nolint
			// Opus related codec settings
			RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeOpus, 48000, 2, "minptime=10;useinbandfec=1", nil},
			PayloadType:        111,
		},
	} {
		if err := m.RegisterCodec(codec, webrtc.RTPCodecTypeAudio); err != nil {
			return err
		}
	}

	// nolint
	videoRTCPFeedback := []webrtc.RTCPFeedback{{"goog-remb", ""}, {"ccm", "fir"}, {"nack", ""}, {"nack", "pli"}}

	for _, codec := range []webrtc.RTPCodecParameters{
		{
			// nolint
			RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeH264, 90000, 0, "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f", videoRTCPFeedback},
			PayloadType:        102,
		},
		{
			// nolint
			RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeH264, 90000, 0, "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f", videoRTCPFeedback},
			PayloadType:        127,
		},
		{
			// nolint
			RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeH264, 90000, 0, "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f", videoRTCPFeedback},
			PayloadType:        125,
		},
		{
			// nolint
			RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeH264, 90000, 0, "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42e01f", videoRTCPFeedback},
			PayloadType:        108,
		},
		{
			// nolint
			RTPCodecCapability: webrtc.RTPCodecCapability{"video/rtx", 90000, 0, "apt=108", nil},
			PayloadType:        109,
		},
		{
			// nolint
			RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeH264, 90000, 0, "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f", videoRTCPFeedback},
			PayloadType:        127,
		},
		{
			// nolint
			RTPCodecCapability: webrtc.RTPCodecCapability{webrtc.MimeTypeH264, 90000, 0, "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=640032", videoRTCPFeedback},
			PayloadType:        123,
		},
	} {
		if err := m.RegisterCodec(codec, webrtc.RTPCodecTypeVideo); err != nil {
			return err
		}
	}

	for _, extension := range []string{
		"urn:ietf:params:rtp-hdrext:sdes:mid",
		"urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id",
		"urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id",
	} {
		if err := m.RegisterHeaderExtension(webrtc.RTPHeaderExtensionCapability{URI: extension}, webrtc.RTPCodecTypeVideo); err != nil {
			return err
		}
	}

	return nil
}

var (
	webrtcAPI *webrtc.API
)

func Configure(config *IngestConfig) {
	mediaEngine := &webrtc.MediaEngine{}
	mediaSettings := webrtc.SettingEngine{}

	udpMux, err := ice.NewMultiUDPMuxFromPort(int(config.UDPPort))
	if err != nil {
		panic(err)
	}
	mediaSettings.SetICEUDPMux(udpMux)
	log.Printf("Listening UDP for WebRTC traffic on :%d\n", config.UDPPort)

	tcpListener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IP{0, 0, 0, 0},
		Port: int(config.TCPPort),
	})
	if err != nil {
		panic(err)
	}
	tcpMux := webrtc.NewICETCPMux(nil, tcpListener, 8)
	mediaSettings.SetICETCPMux(tcpMux)
	log.Printf("Listening TCP for WebRTC traffic on :%d\n", config.TCPPort)

	mediaSettings.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeTCP4,
		webrtc.NetworkTypeUDP4,
		webrtc.NetworkTypeTCP6,
		webrtc.NetworkTypeUDP6,
	})

	if err := populateMediaEngine(mediaEngine); err != nil {
		panic(err)
	}

	if config.NATPublicIP != "" {
		mediaSettings.SetNAT1To1IPs([]string{config.NATPublicIP}, webrtc.ICECandidateTypeHost)
	}

	webrtcAPI = webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(mediaSettings),
	)
}

type NatsConfig struct {
	Port string
	Host string
}

func (c *NatsConfig) GetUrl() string {
	return fmt.Sprintf("nats://%s:%s", c.Host, c.Port)
}

func NewNatsConfig() *NatsConfig {
	return &NatsConfig{
		Host: envutils.Env(variables.NATS_HOST, variables.NATS_HOST_DEFAULT),
		Port: envutils.Env(variables.NATS_PORT, variables.NATS_PORT_DEFAULT),
	}
}

type NatsConnectionParams struct {
	Config *NatsConfig
}

func WithNatsConnection(params NatsConnectionParams) *nats.Conn {
	conn, err := nats.Connect(params.Config.GetUrl(), nats.Timeout(time.Second*5), nats.RetryOnFailedConnect(true))
	if err != nil {
		log.Panicf("Unable start nats connection. Err: %s", err)
		os.Exit(1)
	}

	return conn
}

type IngestConfig struct {
	BroadcasterID string
	Username      string
	NATPublicIP   string
	UDPPort       uint16
	TCPPort       uint16
}

func NewIngestConfig() *IngestConfig {
	udpPortRaw := envutils.Env(variables.INGEST_UDP_PORT, variables.INGEST_UDP_PORT_DEFAULT)
	udpPort, err := envutils.ParseUint16(udpPortRaw)
	if err != nil {
		log.Printf("[ERROR] wrong udp port %s. Fallback to %s", udpPortRaw, variables.INGEST_UDP_PORT_DEFAULT)
		port, _ := envutils.ParseUint16(variables.INGEST_UDP_PORT_DEFAULT)
		udpPort = port
	}

	tcpPortRaw := envutils.Env(variables.INGEST_TCP_PORT, variables.INGEST_TCP_PORT_DEFAULT)
	tcpPort, err := envutils.ParseUint16(tcpPortRaw)
	if err != nil {
		log.Printf("[ERROR] wrong tcp port %s. Fallback to %s", tcpPortRaw, variables.INGEST_TCP_PORT_DEFAULT)
		port, _ := envutils.ParseUint16(variables.INGEST_TCP_PORT)
		tcpPort = port
	}

	return &IngestConfig{
		BroadcasterID: envutils.Env(variables.INGEST_BROADCASTER_ID, variables.INGEST_BROADCASTER_ID_DEFAULT),
		Username:      envutils.Env(variables.INGEST_USERNAME, variables.INGEST_USERNAME_DEFAULT),
		NATPublicIP:   envutils.Env(variables.INGEST_NAT_PUBLIC_IP, variables.INGEST_NAT_PUBLIC_IP_DEFAULT),
		UDPPort:       *udpPort,
		TCPPort:       *tcpPort,
	}
}

func main() {
	natsconf := NewNatsConfig()

	natsconn := WithNatsConnection(NatsConnectionParams{natsconf})

	shutdown := shutdown.NewShutdown()

	router := mux.NewRouter().StrictSlash(true)

	ingestconf := NewIngestConfig()
	go func() {
		msg := &subject.IngestDeployed{Deployed: true, Meta: &subjectpb.BroadcasterMeta{BroadcasterId: ingestconf.BroadcasterID, Username: ingestconf.Username}}
		_ = subject.PublishProtobuf(natsconn, subject.NewIngestDeployed(ingestconf.BroadcasterID), msg)
	}()

	Configure(ingestconf)

	orchestrator := orchestrator.NewOrchestrator(router, shutdown)

	var whip = whip.NewHandler(
		config.NewConfig(),
		orchestrator,
		webrtcAPI,
	)

	var whep = whep.NewHandler(webrtcAPI, orchestrator.WebrtcStream, config.NewConfig())

	router.HandleFunc("/api/consumer/whip", whip.Handler)
	router.HandleFunc("/api/egress/whep", whep.Handler)

	router.HandleFunc("/hello-world", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "hello world!")
	})

	server := &http.Server{
		Handler: router,
		Addr:    ":8089",
	}

	go func() {
		log.Println("Server is listening on :8089")
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	shutdown.Gracefully()
	log.Println("Gracefull shutdown complete...")
}
