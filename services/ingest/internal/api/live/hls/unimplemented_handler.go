package hls

import (
	"encoding/json"
	"log"
	"net/http"
)

type Handler interface {
	Manifest(res http.ResponseWriter, req *http.Request)
	Segment(res http.ResponseWriter, req *http.Request)
}

type HandlerUnimplemented struct{}

type unimplementedResponse struct {
	Message string `json:"message"`
}

var unimplResposne = &unimplementedResponse{
	Message: "Broadcaster end the stream",
}

func (*HandlerUnimplemented) Manifest(res http.ResponseWriter, req *http.Request) {
	resp, err := json.Marshal(unimplResposne)
	if err != nil {
		http.Error(res, "Failed to serialize JSON response", http.StatusInternalServerError)
		return
	}

	log.Println("Manifest router inactive")

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusPreconditionRequired)
	res.Write(resp)
}

func (*HandlerUnimplemented) Segment(res http.ResponseWriter, req *http.Request) {
	resp, err := json.Marshal(unimplResposne)
	if err != nil {
		http.Error(res, "Failed to serialize JSON response", http.StatusInternalServerError)
		return
	}

	log.Println("Segment router inactive")

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusPreconditionRequired)
	res.Write(resp)
}
