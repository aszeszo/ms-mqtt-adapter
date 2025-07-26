ARG BUILD_FROM
ARG VERSION
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o ms-mqtt-adapter ./cmd/ms-mqtt-adapter

FROM $BUILD_FROM

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /

# Copy the binary from builder
COPY --from=builder /app/ms-mqtt-adapter .

# Copy run script
COPY run.sh /
RUN chmod a+x /run.sh

# Home Assistant addon labels
LABEL \
    io.hass.name="MySensors MQTT Adapter" \
    io.hass.description="Bridge between MySensors networks and MQTT with Home Assistant auto-discovery" \
    io.hass.arch="amd64|aarch64|armhf|armv7|i386" \
    io.hass.type="addon" \
    io.hass.version="$VERSION"

CMD ["/run.sh"]
