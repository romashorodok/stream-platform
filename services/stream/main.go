package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/romashorodok/stream-platform/pkg/auth"
	"github.com/romashorodok/stream-platform/pkg/middleware/openapi"
	"github.com/romashorodok/stream-platform/services/stream/internal/handler/stream"
)

func WithOpenAPI3Options() openapi3filter.Options {
	authProvider, _ := auth.NewFakeAuthenticator()
	options := openapi3filter.Options{
		AuthenticationFunc: auth.NewAuthenticator(authProvider),
		MultiError:         true,
	}
	return options
}

func main() {
	port := flag.String("port", "8082", "Port for test HTTP server")
	flag.Parse()

	streamSwagger, err := stream.GetSwagger()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading swagger spec\n: %s", err)
		os.Exit(1)
	}

	streamSwagger.Servers = nil

	r := chi.NewRouter()

	r.Use(openapi.NewOpenAPIRequestMiddleware(streamSwagger, &openapi.Options{
		Options: WithOpenAPI3Options(),
	}))

	stream.HandlerFromMux(stream.NewStreamingService(), r)

	s := &http.Server{
		Handler: r,
		Addr:    net.JoinHostPort("0.0.0.0", *port),
	}

	log.Fatal(s.ListenAndServe())
}
