package streamchannels

import (
	"net/http"

	"github.com/romashorodok/stream-platform/pkg/httputils"
	"github.com/romashorodok/stream-platform/services/stream/internal/streamchannelssvc"
)

func unableGetActiveStreamsList(w http.ResponseWriter, err error) {
	switch err {
	case streamchannelssvc.UnableGetActiveStreamsListError:
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "Unable get active streams.", err.Error())
	default:
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "Something went wrong at GetActiveStreamsList .", err.Error())
	}
	return
}

func unableGetActiveStream(w http.ResponseWriter, err error) {
	switch err {
	case streamchannelssvc.UnableGetActiveStreamError:
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "Unable get active stream.", err.Error())
	case streamchannelssvc.NotFoundGetActiveStreamError:
		httputils.WriteErrorResponse(w, http.StatusNotFound, "Unable get active stream.", err.Error())
	case streamchannelssvc.StandaloneOnlyOperationError:
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "Server in standalone mode.", err.Error())
	default:
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, "Unable get active stream.", err.Error())
	}
}
