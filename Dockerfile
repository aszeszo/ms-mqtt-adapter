# syntax=docker.io/docker/dockerfile:1

ARG BUILD_FROM=ghcr.io/home-assistant/amd64-base:3.19
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ms-mqtt-adapter ./cmd/ms-mqtt-adapter

# Final stage
FROM ${BUILD_FROM}

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/ms-mqtt-adapter /usr/local/bin/ms-mqtt-adapter

# Copy run script
COPY rootfs /

# Make run script executable
RUN chmod a+x /run.sh

# Set the entrypoint
CMD ["/run.sh"]

# Labels
LABEL \
    io.hass.name="MySensors MQTT Adapter" \
    io.hass.description="Bridge between MySensors network and MQTT with Home Assistant auto-discovery" \
    io.hass.arch="aarch64|amd64|armhf|armv7|i386" \
    io.hass.type="addon" \
    io.hass.version="1.0.0" \
    maintainer="aszeszo" \
    org.opencontainers.image.title="MySensors MQTT Adapter" \
    org.opencontainers.image.description="Bridge between MySensors network and MQTT with Home Assistant auto-discovery" \
    org.opencontainers.image.vendor="aszeszo" \
    org.opencontainers.image.authors="aszeszo" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.url="https://github.com/aszeszo/ms-mqtt-adapter" \
    org.opencontainers.image.source="https://github.com/aszeszo/ms-mqtt-adapter" \
    org.opencontainers.image.documentation="https://github.com/aszeszo/ms-mqtt-adapter" \
    org.opencontainers.image.created="$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
    org.opencontainers.image.revision="$(git rev-parse HEAD)"