ARG BUILD_FROM=alpine:3.19
ARG VERSION=dev

# ── Local path: build frontend from source ──────────────────────────────────

FROM node:25-alpine AS frontend-builder

WORKDIR /frontend
COPY web/frontend/package.json web/frontend/package-lock.json* ./
RUN npm ci
COPY web/frontend/ ./
RUN npm run build

# Go build using frontend built in previous stage
FROM golang:1.24-alpine AS builder-local

WORKDIR /app
COPY . .
COPY --from=frontend-builder /dist ./web/dist/

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o ms-mqtt-adapter ./cmd/ms-mqtt-adapter

# ── CI path: frontend pre-built in context ───────────────────────────────────

FROM golang:1.25-alpine AS builder-ci

WORKDIR /app
COPY . .
# web/dist/ must already exist in the build context (from CI artifact)

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o ms-mqtt-adapter ./cmd/ms-mqtt-adapter

# ── Final images ─────────────────────────────────────────────────────────────

FROM $BUILD_FROM AS local

RUN apk --no-cache add ca-certificates

WORKDIR /
COPY --from=builder-local /app/ms-mqtt-adapter .
COPY run.sh /
RUN chmod a+x /run.sh

LABEL \
    io.hass.name="MySensors MQTT Adapter" \
    io.hass.description="Bridge between MySensors networks and MQTT with Home Assistant auto-discovery" \
    io.hass.arch="amd64|aarch64|armhf|armv7|i386" \
    io.hass.type="addon" \
    io.hass.version="$VERSION"

CMD ["/run.sh"]

FROM $BUILD_FROM AS ci

RUN apk --no-cache add ca-certificates

WORKDIR /
COPY --from=builder-ci /app/ms-mqtt-adapter .
COPY run.sh /
RUN chmod a+x /run.sh

LABEL \
    io.hass.name="MySensors MQTT Adapter" \
    io.hass.description="Bridge between MySensors networks and MQTT with Home Assistant auto-discovery" \
    io.hass.arch="amd64|aarch64|armhf|armv7|i386" \
    io.hass.type="addon" \
    io.hass.version="$VERSION"

CMD ["/run.sh"]
