FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o ms-mqtt-adapter ./cmd/ms-mqtt-adapter

FROM alpine:3.22
RUN apk --no-cache add ca-certificates
WORKDIR /

COPY --from=builder /app/ms-mqtt-adapter .

CMD ["/ms-mqtt-adapter", "-config", "/data/options.json"]
