
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

FROM golang:1.20.6-alpine3.18 as stream-builder

WORKDIR /app

COPY go.mod ./
COPY pkg ./pkg/
COPY gen/ ./gen/

WORKDIR /app/services/stream

COPY services/stream/go.mod ./
COPY services/stream/go.sum ./
COPY services/stream/main.go ./main.go
COPY services/stream/internal/ ./internal/
COPY services/stream/cmd/ ./cmd/

RUN go mod download

RUN apkArch="$(apk --print-arch)"; \
      case "$apkArch" in \
        aarch64) export GOARCH='arm64' ;; \
        *) export GOARCH='amd64' ;; \
      esac; \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o stream ./main.go

CMD ["/app/services/stream/stream"]

FROM scratch as stream

WORKDIR /app

COPY --from=stream-builder /app/services/stream/stream /usr/bin/

EXPOSE 8082/tcp

CMD ["stream"]

FROM golang:1.20.6-alpine3.18 as identity-builder

WORKDIR /app

COPY go.mod ./
COPY pkg ./pkg/
COPY gen/ ./gen/

WORKDIR /app/services/identity

COPY services/identity/go.mod ./
COPY services/identity/go.sum ./
COPY services/identity/main.go ./main.go
COPY services/identity/internal/ ./internal/
COPY services/identity/pkg/ ./pkg/

RUN go mod download

RUN apkArch="$(apk --print-arch)"; \
      case "$apkArch" in \
        aarch64) export GOARCH='arm64' ;; \
        *) export GOARCH='amd64' ;; \
      esac; \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o identity ./main.go

CMD ["/app/services/identity/identity"]

FROM scratch as identity

WORKDIR /app

COPY --from=identity-builder /app/services/identity/identity /usr/bin/

EXPOSE 8083/tcp

CMD ["identity"]

FROM golang:1.19 as ingestion-operator-builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod ./
COPY pkg ./pkg/
COPY gen/ ./gen/

RUN go mod download

WORKDIR /app/operators/ingestion-operator

COPY operators/ingestion-operator/go.mod go.mod
COPY operators/ingestion-operator/go.sum go.sum

RUN go mod download

COPY operators/ingestion-operator/cmd/main.go cmd/main.go
COPY operators/ingestion-operator/api/ api/
COPY operators/ingestion-operator/internal/ internal/
COPY operators/ingestion-operator/grpcserver/ grpcserver/

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager ./cmd/main.go

FROM gcr.io/distroless/static:nonroot as ingestion-operator

WORKDIR /

COPY --from=ingestion-operator-builder /app/operators/ingestion-operator/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]


FROM node:20-alpine3.17 as client-builder

WORKDIR /app/client

COPY client ./

RUN npm install
RUN npm run build

FROM node:20-alpine3.17 as client

WORKDIR /app

COPY --from=client-builder /app/client/build .
COPY --from=client-builder /app/client/package.json ./package.json
COPY --from=client-builder /app/client/.env.local ./.env.local

EXPOSE 3000
CMD ["node", "index.js"]

FROM golang:1.20.6-alpine3.18 as migrate-builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
COPY cmd/migrate ./cmd/migrate

COPY services/stream/migrations cmd/migrate/stream
COPY services/identity/migrations cmd/migrate/identity

RUN go mod download

WORKDIR /app/cmd/migrate

RUN apkArch="$(apk --print-arch)"; \
      case "$apkArch" in \
        aarch64) export GOARCH='arm64' ;; \
        *) export GOARCH='amd64' ;; \
      esac; \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o migrate ./main.go

FROM scratch as migrate

WORKDIR /app

COPY --from=migrate-builder /app/cmd/migrate /usr/bin/

COPY services/stream/migrations stream
COPY services/identity/migrations identity


