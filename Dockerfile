
FROM golang:1.20.6-alpine3.18 as ingest-builder

WORKDIR /app

COPY go.mod ./
COPY pkg ./pkg/

RUN go mod download

WORKDIR /app/services/ingest

COPY services/ingest/go.mod ./
COPY services/ingest/go.sum ./
COPY services/ingest/pkg/ ./pkg/
COPY services/ingest/internal/ ./internal/
COPY services/ingest/cmd/ ./cmd/

RUN go mod download

RUN apkArch="$(apk --print-arch)"; \
      case "$apkArch" in \
        aarch64) export GOARCH='arm64' ;; \
        *) export GOARCH='amd64' ;; \
      esac; \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o ingest ./cmd/ingest/main.go


RUN apk add --no-cache ffmpeg

CMD ["/app/services/ingest/ingest"]

FROM scratch as ingest

WORKDIR /app

COPY --from=ingest-builder /app/services/ingest/ingest /usr/bin/

EXPOSE 8089/tcp
EXPOSE 8443/udp

CMD ["ingest"]


FROM golang:1.20.6-alpine3.18 as webrtc-client

WORKDIR /app

COPY go.mod ./
COPY pkg ./pkg/

RUN go mod download

WORKDIR /app/services/ingest

COPY services/ingest/go.mod ./
COPY services/ingest/go.sum ./
COPY services/ingest/pkg/ ./pkg/
COPY services/ingest/internal/ ./internal/
COPY services/ingest/cmd/ ./cmd/

RUN go mod download

RUN apkArch="$(apk --print-arch)"; \
      case "$apkArch" in \
        aarch64) export GOARCH='arm64' ;; \
        *) export GOARCH='amd64' ;; \
      esac; \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o webrtc-client ./cmd/webrtc-client/main.go ./cmd/webrtc-client/start_ffmpeg.go

RUN apk add --no-cache ffmpeg
RUN apk add font-jetbrains-mono-nerd

CMD ["/app/services/ingest/webrtc-client"]

