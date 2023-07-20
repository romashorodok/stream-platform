package mediaprocessor

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/romashorodok/stream-platform/services/ingest/internal/orchestrator"
	"github.com/romashorodok/stream-platform/services/ingest/pkg/namedpipe"
)

type HSLMediaProcessor struct {
	orchestrator.MediaProcessor
}

func (HSLMediaProcessor) Transcode(videoSourcePipe *io.PipeReader, audioSourcePipe *io.PipeReader) (err error) {

	ffmpeg := exec.Command("ffmpeg",
		"-fflags", "nobuffer+genpts",
		"-threads", "0",
		"-i", "pipe:0",
		"-i", "pipe:3",
		"-loglevel", "info",
		"-c:v", "copy",
		"-preset", "ultrafast",
		"-tune", "zerolatency",
		"-crf", "30",
		"-maxrate", "2000k",
		"-bufsize", "1500k",
		"-pix_fmt", "yuv420p",
		"-c:a", "copy",
		"-err_detect", "ignore_err",
		"-muxdelay", "0",
		"-map_metadata", "0",
		"-copyts",
		"-copytb", "0",
		"-f", "hls",
		"-hls_time", "4",
		"-hls_list_size", "8",
		"-hls_flags", "delete_segments+independent_segments",
		"-hls_start_number_source", "datetime",
		"-hls_segment_filename", "output_%03d.ts",
		"output.m3u8",
	)

	ffmpeg.Stdin = videoSourcePipe

	audioPipe, err := namedpipe.NewNamedPipe()
	audioPipeFile, err := audioPipe.OpenAsWriteOnly()

	videoStderr, err := ffmpeg.StderrPipe()

	if err != nil {
		log.Println("Failed to open pipes. Err", err)
	}

	ffmpeg.ExtraFiles = []*os.File{audioPipeFile}

	go func() {
		defer audioPipe.Close()

		io.Copy(audioPipeFile, audioSourcePipe)
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
