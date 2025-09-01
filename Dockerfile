# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Install git for go modules that might need it
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o memcp .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /build/memcp .
# Copy Scheme library (runtime scripts)
COPY --from=builder /build/lib ./lib

# Create data directory
RUN mkdir -p /data

# Set up volumes and expose ports
VOLUME /data
EXPOSE 4332
EXPOSE 3307

# Set environment variables (overridable via docker-compose)
ENV PARAMS=
ENV ROOT_PASSWORD=

# Run the application (load default Scheme entrypoint)
# If ROOT_PASSWORD is set, pass it as --root-password; otherwise rely on default in lib/sql.scm
CMD ["sh", "-c", "RP_OPT=; [ -n \"$ROOT_PASSWORD\" ] && RP_OPT=\"--root-password=$ROOT_PASSWORD\"; exec ./memcp -data /data $RP_OPT ${PARAMS} lib/main.scm"]
