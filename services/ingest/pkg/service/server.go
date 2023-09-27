package service

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pion/ice/v2"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
	"github.com/romashorodok/stream-platform/pkg/envutils"
	"github.com/romashorodok/stream-platform/pkg/httputils"
	"github.com/romashorodok/stream-platform/pkg/netutils"
	"github.com/romashorodok/stream-platform/pkg/variables"
	"go.uber.org/fx"
)

type IngestWebrtcConfig struct {
	NATPublicIP string
	UDPPort     uint16
	TCPPort     uint16
}

func NewIngestWebrtcConfig() (*IngestWebrtcConfig, error) {
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

	return &IngestWebrtcConfig{
		NATPublicIP: envutils.Env(variables.INGEST_NAT_PUBLIC_IP, variables.INGEST_NAT_PUBLIC_IP_DEFAULT),
		UDPPort:     *udpPort,
		TCPPort:     *tcpPort,
	}, nil
}

type IngestHttpConfig struct {
	Host string
	Port string
}

func (s IngestHttpConfig) GetAddr() string {
	return net.JoinHostPort(s.Host, s.Port)
}

func NewIngestHttpConfig() (*IngestHttpConfig, error) {
	return &IngestHttpConfig{
		Host: envutils.Env(variables.INGEST_HTTP_HOST, variables.INGEST_HTTP_HOST_DEFAULT),
		Port: envutils.Env(variables.INGEST_HTTP_PORT, variables.INGEST_HTTP_PORT_DEFAULT),
	}, nil
}

func populateMediaEngine(m *webrtc.MediaEngine) error {
	for _, codec := range []webrtc.RTPCodecParameters{
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2, SDPFmtpLine: "minptime=10;useinbandfec=1", RTCPFeedback: nil},
			PayloadType:        111,
		},
	} {
		if err := m.RegisterCodec(codec, webrtc.RTPCodecTypeAudio); err != nil {
			return err
		}
	}

	videoRTCPFeedback := []webrtc.RTCPFeedback{{Type: "goog-remb", Parameter: ""}, {Type: "ccm", Parameter: "fir"}, {Type: "nack", Parameter: ""}, {Type: "nack", Parameter: "pli"}}

	for _, codec := range []webrtc.RTPCodecParameters{
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000, Channels: 0, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        102,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000, Channels: 0, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        127,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000, Channels: 0, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        125,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000, Channels: 0, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42e01f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        108,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: "video/rtx", ClockRate: 90000, Channels: 0, SDPFmtpLine: "apt=108", RTCPFeedback: nil},
			PayloadType:        109,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000, Channels: 0, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=0;profile-level-id=42001f", RTCPFeedback: videoRTCPFeedback},
			PayloadType:        127,
		},
		{
			RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000, Channels: 0, SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=640032", RTCPFeedback: videoRTCPFeedback},
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

type IngestWebrtcAPIParams struct {
	fx.In

	Config *IngestWebrtcConfig
}

func NewIngestWebrtcAPI(params IngestWebrtcAPIParams) *webrtc.API {
	config := params.Config

	mediaEngine := &webrtc.MediaEngine{}
	mediaSettings := webrtc.SettingEngine{}

	mediaEngine.RegisterDefaultCodecs()

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
	} else {
		if ips, err := netutils.GetLocalIPAddresses(false, nil); err == nil && ips != nil {
			mediaSettings.SetNAT1To1IPs(ips, webrtc.ICECandidateTypeHost)
		}
	}

	interceptorRegistry := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(mediaEngine, interceptorRegistry); err != nil {
		log.Fatal(err)
	}

	return webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(mediaSettings),
	)
}

type IngestHttpParams struct {
	fx.In

	Config    *IngestHttpConfig
	Handler   http.Handler
	Lifecycle fx.Lifecycle
	// watch httputils.httpHandlerGroup if it's []
	Handlers []httputils.HttpHandler `group:"http.handler"`
}

func StartIngestHttp(params IngestHttpParams) {
	server := &http.Server{
		Addr:    params.Config.GetAddr(),
		Handler: params.Handler,
	}

	for _, handler := range params.Handlers {
		handler.GetOption()(server.Handler)
	}

	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		panic(err)
	}

	go server.Serve(ln)

	params.Lifecycle.Append(
		fx.StopHook(func(ctx context.Context) error {
			return server.Shutdown(ctx)
		}),
	)
}

var router = mux.NewRouter().StrictSlash(true)

func NewRouter() *mux.Router {
	return router
}

func StartIngestWebrtc(*webrtc.API) {}

var ServerModule = fx.Module("server",
	fx.Provide(
		NewIngestWebrtcConfig,
		NewIngestHttpConfig,

		fx.Annotate(
			NewRouter,
			fx.As(new(http.Handler)),
			fx.From(new(mux.Router)),
		),

		NewIngestWebrtcAPI,
	),
)
