package transport

import (
	"context"
	"ms-mqtt-adapter/internal/mysensors"
)

type Transport interface {
	Connect(ctx context.Context) error
	Disconnect() error
	Send(message *mysensors.Message) error
	Receive() <-chan *mysensors.Message
	IsConnected() bool
}

type MessageHandler func(*mysensors.Message)

type TransportConfig struct {
	Type     string
	Ethernet EthernetConfig
	RS485    RS485Config
}

type EthernetConfig struct {
	Host string
	Port int
}

type RS485Config struct {
	Device   string
	BaudRate int
}
