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
