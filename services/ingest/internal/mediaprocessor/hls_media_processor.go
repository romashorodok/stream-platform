package mediaprocessor

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
)

type HSLMediaProcessor struct {
}

func (HSLMediaProcessor) Transcode(mediaSourcePipe *io.PipeReader) error {
	ffmpeg := exec.Command("ffmpeg",
		"-f", "lavfi", "-re",
		"-i", "anullsrc=channel_layout=stereo:sample_rate=44100",
		"-i", "pipe:0",
		"-c:a", "aac",
		"-loglevel", "debug",
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-crf", "30",
		"-maxrate", "3000k",
		"-bufsize", "6000k",
		"-pix_fmt", "yuv420p",
		"-an",
		"-f", "hls",
		"-hls_time", "8",
		"-hls_list_size", "4",
		"-hls_flags", "delete_segments",
		"-hls_start_number_source", "datetime",
		"-hls_segment_filename", "output_%03d.ts",
		"output.m3u8")

	stdin, _ := ffmpeg.StdinPipe()
	stderr, _ := ffmpeg.StderrPipe()

	go func() {
		scanner := bufio.NewScanner(stderr)

		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	go func() {
		defer func() {
			log.Println("Close ffmpeg pipe")
			stdin.Close()
		}()

		io.Copy(stdin, mediaSourcePipe)
	}()

	if err := ffmpeg.Run(); err != nil {
		log.Println("Error when running ffmpeg. Err:", err)
		return err
	}

	return nil
}
