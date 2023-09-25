package httputils

import (
	"net/http"

	"go.uber.org/fx"
)

type HttpHandlerOption func(http.Handler)

type HttpHandler interface {
	GetOption() HttpHandlerOption
}

const httpHandlerGroup = `group:"http.handler"`

func AsHttpHandler(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(HttpHandler)),
		fx.ResultTags(httpHandlerGroup),
	)
}
