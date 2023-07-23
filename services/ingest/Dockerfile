
FROM golang:1.20.6-alpine3.18 as ingest-builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY internal/ internal/
COPY pkg/ pkg/
COPY cmd/ingest/main.go cmd/ingest/

RUN apk add --no-cache ffmpeg

RUN apkArch="$(apk --print-arch)"; \
      case "$apkArch" in \
        aarch64) export GOARCH='arm64' ;; \
        *) export GOARCH='amd64' ;; \
      esac; \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ingest ./cmd/ingest/main.go

CMD ["/app/ingest"]

FROM scratch as ingest

WORKDIR /app

COPY --from=ingest-builder /app/ingest /usr/bin/

EXPOSE 8089/tcp
EXPOSE 8443/udp

CMD ["ingest"]


FROM golang:1.20.6-alpine3.18 as webrtc-client

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

RUN apk add --no-cache ffmpeg
RUN apk add font-jetbrains-mono-nerd

COPY cmd/webrtc-client/ cmd/webrtc-client/

RUN apkArch="$(apk --print-arch)"; \
      case "$apkArch" in \
        aarch64) export GOARCH='arm64' ;; \
        *) export GOARCH='amd64' ;; \
      esac; \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o webrtc-client ./cmd/webrtc-client/main.go ./cmd/webrtc-client/start_ffmpeg.go

CMD ["/app/webrtc-client"]

