package main

import (
	"log"
	"net/http"

	"github.com/romashorodok/stream-platform/services/ingest/internal/api/consumer/whip"
)


func main() {
	mux := http.NewServeMux()

	var whip = whip.NewHandler()

	mux.HandleFunc("/api/consumer/whip", whip.Handler)

	server := &http.Server{
		Handler: mux,
		Addr:    ":8089",
	}

	log.Fatal(server.ListenAndServe())
}
