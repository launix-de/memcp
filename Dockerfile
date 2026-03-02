# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Install git for go modules that might need it
RUN apk add --no-cache git

# Copy go mod files and local replace dependencies first for better caching
COPY go.mod go.sum ./
COPY third_party/ ./third_party/
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o memcp .

# Runtime stage
FROM alpine:3.21

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /build/memcp .
# Copy Scheme library (runtime scripts)
COPY --from=builder /build/lib ./lib
# Copy dashboard and static assets
COPY --from=builder /build/assets ./assets

# Create data directory
RUN mkdir -p /data

# Set up volumes and expose ports
VOLUME /data
EXPOSE 4321
EXPOSE 3307

# Set environment variables (overridable via docker-compose)
# ROOT_PASSWORD is only considered on the first run with a fresh data directory.
# The image only contains the well-known default "admin"; the actual secret is
# supplied at runtime via -e ROOT_PASSWORD=... and never baked into the image.
# hadolint ignore=DL3002
ENV PARAMS=
ENV ROOT_PASSWORD=admin
ENV APP=lib/main.scm

# Run the application (load default Scheme entrypoint)
# --no-repl prevents the process from exiting when stdin is closed (required in containers)
# exec replaces the shell so SIGTERM from docker stop reaches memcp directly
CMD ["sh", "-c", "exec ./memcp --no-repl -data /data --root-password=\"$ROOT_PASSWORD\" $PARAMS $APP"]
