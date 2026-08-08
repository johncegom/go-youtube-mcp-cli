# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ENV CGO_ENABLED=0
RUN go build -ldflags "-s -w -X main.version=${VERSION}" -o /out/youtube-cli ./cmd/youtube-cli
RUN go build -ldflags "-s -w -X main.version=${VERSION}" -o /out/youtube-mcp ./cmd/youtube-mcp

# Debian-slim, not scratch/alpine: go-ytdlp's lazy installer downloads and
# runs a real yt-dlp/ffmpeg binary at container runtime, so libc and
# ca-certificates need to actually be present, not just the Go binaries.
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/youtube-cli /out/youtube-mcp /usr/local/bin/
ENTRYPOINT []
CMD ["youtube-mcp"]
