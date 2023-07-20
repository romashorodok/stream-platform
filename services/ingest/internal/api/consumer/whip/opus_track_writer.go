package whip

import (
	"io"
	"log"
	"sync"
	"time"

	"github.com/at-wat/ebml-go/webm"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
)

func opusTrackWriter(remoteTrack *webrtc.TrackRemote, peerConnection *webrtc.PeerConnection, pipeWriter *io.PipeWriter) {
	var writerMutex sync.RWMutex

	audioBuilder := samplebuilder.New(10, &codecs.OpusPacket{}, 48000)

	// NOTE: It's packing opus into webm(EBML) container, but how to pass it like plain bytes.
	ws, _ := webm.NewSimpleBlockWriter(pipeWriter, []webm.TrackEntry{
		{
			Name:            "Audio",
			TrackNumber:     1,
			TrackUID:        12345,
			CodecID:         "A_OPUS",
			TrackType:       2,
			DefaultDuration: 20000000,
			Audio: &webm.Audio{
				SamplingFrequency: 48000.0,
				Channels:          2,
			},
		},
	})

	audioWEBMWriter := ws[0]
	var audioTimestamp time.Duration

	for {
		rtp, _, _ := remoteTrack.ReadRTP()

		audioBuilder.Push(rtp)

		sample := audioBuilder.Pop()

		if sample == nil {
			continue
		}

		writerMutex.RLock()

		audioTimestamp += sample.Duration

		_, err := audioWEBMWriter.Write(true, int64(audioTimestamp/time.Millisecond), sample.Data)

		if err != nil {
			log.Println("Unable write the audio into pipe. Err:", err)
		}

		writerMutex.RUnlock()
	}
}
