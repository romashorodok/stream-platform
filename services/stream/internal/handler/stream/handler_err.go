package stream

import (
	"net/http"

	"github.com/romashorodok/stream-platform/pkg/httputils"
	"github.com/romashorodok/stream-platform/services/stream/internal/streamsvc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func unableStartIngestServerErrorHandler(w http.ResponseWriter, err error) {
	if e, ok := status.FromError(err); ok {
		switch e.Code() {
		case codes.Unavailable:
			httputils.WriteErrorResponse(w, http.StatusServiceUnavailable, "Ingest operator is not available.", e.Message())
		case codes.Aborted:
			httputils.WriteErrorResponse(w, http.StatusConflict, "Ingest server already running or something went wrong on ingest operator.", e.Message())
		default:
			httputils.WriteErrorResponse(w, http.StatusInternalServerError, "Something went wrong on ingest operator", e.Message())
		}
		return
	}

	switch err {
	case streamsvc.UnableInsertStream:
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
	case streamsvc.StreamAlredyExists:
		httputils.WriteErrorResponse(w, http.StatusAccepted, err.Error())
	default:
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
	}
	return
}

func unableStopIngestServerErrorHandler(w http.ResponseWriter, err error) {
	switch err {
	case streamsvc.NotFoundActiveStream:
		httputils.WriteErrorResponse(w, http.StatusNotFound, err.Error())
	case streamsvc.UnableStopStream:
		httputils.WriteErrorResponse(w, http.StatusAccepted, err.Error())
	case streamsvc.UnableDeleteStream:
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
	default:
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
	}
}
