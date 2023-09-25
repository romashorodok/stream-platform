package mediaprocessor

import (
	"context"
	"errors"
	"io"

	"github.com/romashorodok/stream-platform/services/ingest/internal/mediaprocessor/hls"
	"go.uber.org/fx"
)

type MediaProcessor interface {
	Transcode(context.Context, *io.PipeReader, *io.PipeReader) error
	Destroy()
}

var _ MediaProcessor = (*hls.FFmpegHLSMediaProcessor)(nil)

func CastMediaProcessor[F any](target any) (*F, error) {
	processor, ok := target.(*F)
	if !ok {
		return nil, errors.New("invalid processor type")
	}
	return processor, nil
}

func AsMediaProcessor[F any](mediaProcessor F, labels ...string) any {
	return fx.Annotate(
		mediaProcessor,
		fx.As(new(MediaProcessor)),
		// NOTE: cannot use fx.From with fx.ResultTags to provide it as typed struct
		fx.ResultTags(labels...),
	)
}

var FxDefaultHLSMediaProcessor = AsMediaProcessor(
	hls.NewFFmpegHLSMediaProcessor,
	`name:"mediaprocessor.hls.default"`,

	`group:"mediaprocessor.hls"`,
	`group:"mediaprocessor"`,
)
