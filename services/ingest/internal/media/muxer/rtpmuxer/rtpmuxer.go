package rtpmuxer

import (
	"errors"
	"io"
	"sync"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type PionWebrtcLocalTrack = webrtc.TrackLocalStaticRTP
type PionWebrtcRemoteTrack = webrtc.TrackRemote

type rtpMuxer struct {
	rtpCodec rtp.Packet
	reader   *io.PipeReader
	writer   *io.PipeWriter
	mx       sync.Mutex
	writers  []io.Writer
}

func (d *rtpMuxer) Bind(track *PionWebrtcRemoteTrack) {
	buf := make([]byte, 4096)

	for {
		d.mx.Lock()
		n, _, err := track.Read(buf)

		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		_, _ = d.Write(buf[:n])
		d.mx.Unlock()
	}
}

func (d *rtpMuxer) BindLocal(track *PionWebrtcLocalTrack) {
	d.writers = append(d.writers, track)
}

func (d *rtpMuxer) Read(p []byte) (n int, err error) {
	return d.reader.Read(p)
}

func (d *rtpMuxer) Write(p []byte) (n int, err error) {
	if locked := d.mx.TryLock(); locked {
		return 0, errors.New("locked")
	}

	if err := d.rtpCodec.Unmarshal(p); err != nil {
		return 0, err
	}

	return io.MultiWriter(d.writers...).Write(d.rtpCodec.Payload)
}

func NewRtpMuxer() *rtpMuxer {
	demuxer := rtpMuxer{}
	demuxer.reader, demuxer.writer = io.Pipe()
	demuxer.writers = append(demuxer.writers, demuxer.writer)
	return &demuxer
}
