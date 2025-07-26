FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o ms-mqtt-adapter ./cmd/ms-mqtt-adapter

FROM alpine:3.22
RUN apk --no-cache add ca-certificates
WORKDIR /

COPY --from=builder /app/ms-mqtt-adapter .

# Home Assistant addon labels
LABEL \
    io.hass.name="MySensors MQTT Adapter" \
    io.hass.description="Bridge between MySensors networks and MQTT with Home Assistant auto-discovery" \
    io.hass.arch="amd64|aarch64|armhf|armv7|i386" \
    io.hass.type="addon" \
    io.hass.version="2.0.2"
CMD ["/ms-mqtt-adapter", "-config", "/data/options.json"]
