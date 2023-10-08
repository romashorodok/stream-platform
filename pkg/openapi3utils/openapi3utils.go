package openapi3utils

import (
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
)

type Spec = openapi3.T

type HandlerFunc = func(http.Handler) http.Handler

type HandlerSpecValidator func(spec *Spec) HandlerFunc
