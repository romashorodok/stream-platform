package main

import (
	"bufio"
	"log"
	"os/exec"
)

// This need because ffmepg don't support multipe rtp muxer for stream and i need start manually start separate process for audio and video. I think in OBS same approach there is Video/Audio soruces which handling separately

var (
	ffmpegVideoRTPCommand = []string{
		"-re",
		"-protocol_whitelist", "file,udp,rtp",
		"-f", "lavfi",
		"-i", "testsrc=size=1280x720:rate=60[out0];sine=frequency=1000:sample_rate=48000[out1]",
		"-vf", "[in]drawtext=fontsize=96:box=1:boxcolor=black@0.75:boxborderw=5:fontcolor=white:x=(w-text_w)/2:y=((h-text_h)/2)+((h-text_h)/-2):text='Hello from FFmpeg',drawtext=fontsize=96:box=1:boxcolor=black@0.75:boxborderw=5:fontcolor=white:x=(w-text_w)/2:y=((h-text_h)/2)+((h-text_h)/2):text='%{gmtime\\:%H\\\\\\:%M\\\\\\:%S} UTC'[out]",
		"-nal-hrd", "cbr",
		"-metadata:s:v",
		"encoder=test",
		"-vcodec", "libx264",
		"-acodec", "libopus",
		"-preset", "veryfast",
		"-profile:v", "baseline",
		"-pix_fmt", "yuv420p",
		"-color_range", "1",
		"-colorspace", "1",
		"-color_primaries", "1",
		"-color_trc", "1",
		"-flags:v", "+global_header",
		"-bsf:v", "dump_extra",
		"-x264-params",
		"nal-hrd=cbr:min-keyint=2:keyint=2:scenecut=0:bframes=0",
		"-an",
		"-f", "rtp",
		"-payload_type", "111",
		"-pkt_size", "1200",
		"-buffer_size", "1200",
		FFMPEG_VIDEO_RTP_LISTENER_HOST,
	}

	ffmpegAudioRTPCommand = []string{
		"-re",
		"-f", "lavfi",
		"-i", "anullsrc=channel_layout=stereo:sample_rate=44100",
		"-acodec", "libopus",
		"-f", "rtp",
		"-payload_type", "111",
		"-pkt_size", "1200",
		"-buffer_size", "1200",
		FFMPEG_AUDIO_RTP_LISTENER_HOST,
	}
)

func startFFmpegAudioRTPSource() {
	ffmpeg := exec.Command("ffmpeg", ffmpegAudioRTPCommand...)
	stderr, _ := ffmpeg.StderrPipe()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Println("[RTP AUDIO]", scanner.Text())
		}
	}()

	if err := ffmpeg.Run(); err != nil {
		log.Println("Error when process audio. Err:", err)
	}
}

func startFFmpegVideoRTPSource() {
	ffmpeg := exec.Command("ffmpeg", ffmpegVideoRTPCommand...)
	stderr, _ := ffmpeg.StderrPipe()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Println("[RTP VIDEO]", scanner.Text())
		}
	}()

	if err := ffmpeg.Run(); err != nil {
		log.Println("Error when process video. Err:", err)
	}
}
