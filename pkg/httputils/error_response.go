package httputils

import (
	"encoding/json"
	"net/http"
	"strings"
)

type ErrorResponse struct {
	Message string `json:"message"`
}

func WriteErrorResponse(w http.ResponseWriter, statusCode int, errorMessage ...string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	json.NewEncoder(w).Encode(ErrorResponse{
		Message: strings.Join(errorMessage, " "),
	})
}
