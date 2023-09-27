package hls

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/google/uuid"
	"github.com/romashorodok/stream-platform/services/ingest/pkg/namedpipe"
	"go.uber.org/fx"
)

type FFmpegHLSMediaProcessor struct {
	SourceDirectory  string
	ManifestFile     string
	SegmentPrefixURL string
	audioNamedPipe   *namedpipe.NamedPipe
}

func (processor *FFmpegHLSMediaProcessor) Transcode(ctx context.Context, videoSourcePipe *io.PipeReader, audioSourcePipe *io.PipeReader) (err error) {
	// Need if here will be nil ptr
	defer processor.Destroy()

	dir, err := os.MkdirTemp("", fmt.Sprintf("%s-*", uuid.New()))
	if err != nil {
		log.Println("[HLS Proceessor] Cannot create temp dir")
	}
	processor.SourceDirectory = dir
	processor.ManifestFile = fmt.Sprintf("%s/%s.m3u8",
		processor.SourceDirectory,
		uuid.NewString(),
	)
	processor.SegmentPrefixURL = "hls/"

	log.Println("[HLS Proceessor] Setup output directory to", processor.SourceDirectory)

	ffmpeg := exec.Command("ffmpeg",
		"-fflags", "nobuffer+genpts",
		"-threads", "0",
		"-re",
		"-i", "pipe:0",
		"-i", "pipe:3",
		"-loglevel", "info",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-crf", "30",
		"-maxrate", "2000k",
		"-bufsize", "1500k",
		"-pix_fmt", "yuv420p",
		"-c:a", "libopus",
		"-err_detect", "ignore_err",
		"-muxdelay", "0",
		"-map_metadata", "0",
		"-copyts",
		"-copytb", "0",
		"-strftime", "1",
		"-f", "hls",
		"-hls_time", "4",
		"-hls_list_size", "8",
		"-hls_flags", "delete_segments+independent_segments",
		"-hls_start_number_source", "datetime",
		"-hls_allow_cache", "0",
		"-hls_base_url", processor.SegmentPrefixURL,
		"-hls_segment_filename", "output_%03d.ts",
		"-hls_segment_filename", fmt.Sprintf("%s/%s", processor.SourceDirectory, "%Y-%m-%d-%s.ts"),
		processor.ManifestFile,
	)

	ffmpeg.Stdin = videoSourcePipe

	audioPipe, err := namedpipe.NewNamedPipe()
	processor.audioNamedPipe = audioPipe
	audioPipeFile, err := audioPipe.OpenAsWriteOnly()

	videoStderr, err := ffmpeg.StderrPipe()

	if err != nil {
		log.Println("Failed to open pipes. Err", err)
	}

	ffmpeg.ExtraFiles = []*os.File{audioPipeFile}

	go func() {
		io.Copy(audioPipeFile, audioSourcePipe)
	}()

	go func() {
		select {
		case <-ctx.Done():
			log.Println("[HLS Proceessor] Stop by context")
			_ = ffmpeg.Process.Kill()
		}
	}()

	go func() {
		scanner := bufio.NewScanner(videoStderr)
		for scanner.Scan() {
			log.Println("[HLS]", scanner.Text())
		}
	}()

	if err := ffmpeg.Run(); err != nil {
		log.Println("Error when running ffmpeg. Err:", err)
		return err
	}

	return nil
}

func (processor *FFmpegHLSMediaProcessor) Destroy() {
	log.Println("[HLS Proceessor] Removing", processor.SourceDirectory)
	os.RemoveAll(processor.SourceDirectory)
	if processor.audioNamedPipe != nil {
		processor.audioNamedPipe.Close()
	}
}

type FFmpegHLSMediaProcessorParams struct {
	fx.In
}

func NewFFmpegHLSMediaProcessor(params FFmpegHLSMediaProcessorParams) *FFmpegHLSMediaProcessor {
	return &FFmpegHLSMediaProcessor{}
}
