# syntax=docker/dockerfile:1
# Copyright (C) 2023 - 2026 Carl-Philip Haensch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

FROM golang:1.24-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
WORKDIR /build
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Keep this list explicit in addition to .dockerignore. A MemCP checkout often
# sits next to real database files which must never enter a builder context.
COPY *.go ./
COPY scm/ ./scm/
COPY storage/ ./storage/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -ldflags="-s -w" -o memcp .

FROM alpine:3.22

ARG VERSION=dev
ARG REVISION=unknown
LABEL org.opencontainers.image.title="MemCP" \
      org.opencontainers.image.description="Smart clusterable distributed database" \
      org.opencontainers.image.url="https://github.com/launix-de/memcp" \
      org.opencontainers.image.source="https://github.com/launix-de/memcp" \
      org.opencontainers.image.version="$VERSION" \
      org.opencontainers.image.revision="$REVISION" \
      org.opencontainers.image.licenses="GPL-3.0-or-later"

RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 memcp \
    && adduser -S -D -H -u 10001 -G memcp memcp \
    && install -d -o memcp -g memcp -m 0700 /data

WORKDIR /app
COPY --from=builder --chown=root:root /build/memcp ./memcp
COPY --chown=root:root lib/ ./lib/
COPY --chown=root:root assets/ ./assets/
COPY --chown=root:root packaging/docker-entrypoint.sh /usr/local/bin/memcp-entrypoint

USER 10001:10001
VOLUME /data
EXPOSE 4321 3307
STOPSIGNAL SIGTERM

ENV MEMCP_DATA_DIR=/data \
    MEMCP_ROOT_PASSWORD_FILE=/run/secrets/memcp_root_password

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:4321/ || exit 1

ENTRYPOINT ["/usr/local/bin/memcp-entrypoint"]
CMD ["lib/main.scm"]
